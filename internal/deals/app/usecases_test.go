package app

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

func TestCreateProjectionSavesProjection(t *testing.T) {
	logTest(t)
	calls := []string{}
	projections := &projectionRepoSpy{calls: &calls}

	uc := NewCreateProjection(projections)
	now := time.Now()
	logMsg(t, "create projection auction=auc-1 supplier=sup-1")
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", "sup-1", deal.ProductSnapshot{Name: "Fish"}, 100, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_projection", "save_projection"})
	if projections.lastSaved == nil || projections.lastSaved.AuctionID != "auc-1" {
		t.Fatal("expected projection to be saved")
	}
}

func TestCreateProjectionNoOpWhenAlreadyExists(t *testing.T) {
	logTest(t)
	calls := []string{}
	projection := deal.NewDealProjection("auc-1", "sup-1", deal.ProductSnapshot{Name: "Fish"}, 100, time.Now())
	projections := &projectionRepoSpy{calls: &calls, projection: projection}

	uc := NewCreateProjection(projections)
	now := time.Now()
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", "sup-1", deal.ProductSnapshot{Name: "Fish"}, 100, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_projection"})
	if projections.lastSaved != nil {
		t.Fatal("expected projection not to be saved")
	}
}

func TestCreateDealFromAuctionWonOrchestratesSaveAndPublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	projection := deal.NewDealProjection("auc-1", "sup-1", deal.ProductSnapshot{Name: "Fish"}, 100, time.Now().Add(-time.Hour))
	deals := &dealRepoSpy{calls: &calls}
	confirmations := &confirmationRepoSpy{calls: &calls}
	projections := &projectionRepoSpy{calls: &calls, projection: projection}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: projections, selections: &selectionRepoSpy{}, outbox: outbox}}

	uc, err := NewCreateDealFromAuctionWon(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", "cust-1", 120, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_projection", "save_deal", "save_projection", "outbox"})
	if deals.lastSaved == nil {
		t.Fatal("expected deal to be saved")
	}
	if len(outbox.saved) == 0 || len(outbox.saved[0]) == 0 {
		t.Fatal("expected events to be saved to outbox")
	}
}

func TestCreateDealFromAuctionWonRequiresProjection(t *testing.T) {
	logTest(t)
	calls := []string{}
	deals := &dealRepoSpy{calls: &calls}
	confirmations := &confirmationRepoSpy{calls: &calls}
	projections := &projectionRepoSpy{calls: &calls}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: projections, selections: &selectionRepoSpy{}, outbox: outbox}}

	uc, err := NewCreateDealFromAuctionWon(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	err = uc.Execute(context.Background(), testMeta(), "auc-1", "buyer-1", 120, time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if err != deal.ErrProjectionNotFound {
		t.Fatalf("expected ErrProjectionNotFound, got %v", err)
	}
	assertCalls(t, calls, []string{"load_projection"})
}

func TestCreateDealSelectionFromAuctionWonCreatesDealForFirstCandidate(t *testing.T) {
	logTest(t)
	calls := []string{}
	projection := deal.NewDealProjection("auc-1", "sup-1", deal.ProductSnapshot{Name: "Fish"}, 100, time.Now().Add(-time.Hour))
	deals := &dealRepoSpy{calls: &calls}
	confirmations := &confirmationRepoSpy{calls: &calls}
	projections := &projectionRepoSpy{calls: &calls, projection: projection}
	selections := &selectionRepoSpy{calls: &calls}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: projections, selections: selections, outbox: outbox}}

	uc, err := NewCreateDealSelectionFromAuctionWon(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", []string{"buyer-1", "buyer-2"}, 120, time.Now()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_projection", "load_selection", "save_deal", "save_projection", "save_selection", "outbox"})
	if selections.lastSaved == nil || selections.lastSaved.DealID == "" {
		t.Fatal("expected selection to be saved with deal id")
	}
	if deals.lastSaved == nil || deals.lastSaved.CustomerID() != "buyer-1" {
		t.Fatalf("expected deal for buyer-1, got %v", deals.lastSaved)
	}
}

func TestHandleDealDeclinedMovesToNextCandidate(t *testing.T) {
	logTest(t)
	calls := []string{}
	selection := deal.NewWinnerSelection(
		"auc-1",
		[]string{"buyer-1", "buyer-2"},
		120,
		time.Now(),
		"sup-1",
		deal.ProductSnapshot{Name: "Fish"},
	)
	deals := &dealRepoSpy{calls: &calls}
	confirmations := &confirmationRepoSpy{calls: &calls}
	selections := &selectionRepoSpy{calls: &calls, selection: selection}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: &projectionRepoSpy{}, selections: selections, outbox: outbox}}

	uc, err := NewHandleDealDeclined(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_selection", "save_deal", "save_selection", "outbox"})
	if deals.lastSaved == nil || deals.lastSaved.CustomerID() != "buyer-2" {
		t.Fatalf("expected deal for buyer-2, got %v", deals.lastSaved)
	}
	if selections.lastSaved == nil || selections.lastSaved.CurrentIndex != 1 {
		t.Fatalf("expected selection to advance to index 1, got %v", selections.lastSaved)
	}
}

func TestHandleDealDeclinedNoOpWhenDealIDMismatch(t *testing.T) {
	logTest(t)
	calls := []string{}
	selection := deal.NewWinnerSelection(
		"auc-1",
		[]string{"buyer-1", "buyer-2"},
		120,
		time.Now(),
		"sup-1",
		deal.ProductSnapshot{Name: "Fish"},
	)
	selection.CurrentIndex = 1
	selection.DealID = "deal-2"

	selections := &selectionRepoSpy{calls: &calls, selection: selection}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: &dealRepoSpy{}, confirmations: &confirmationRepoSpy{}, projections: &projectionRepoSpy{}, selections: selections, outbox: outbox}}

	uc, err := NewHandleDealDeclined(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), "auc-1", "deal-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_selection"})
	if selections.lastSaved != nil {
		t.Fatal("expected no selection save")
	}
	if len(outbox.saved) != 0 {
		t.Fatal("expected no outbox events")
	}
}

func TestConfirmDealOrchestratesLoadSavePublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	item := createPendingDeal(t)
	deals := &dealRepoSpy{calls: &calls, deal: item}
	confirmations := &confirmationRepoSpy{calls: &calls}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: &projectionRepoSpy{}, selections: &selectionRepoSpy{}, outbox: outbox}}

	uc, err := NewConfirmDeal(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), item.ID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_deal", "save_deal", "outbox"})
	if deals.lastSaved.Status() != deal.DealStatusConfirmed {
		t.Fatalf("expected confirmed status, got %s", deals.lastSaved.Status())
	}
}

func TestRequestDealConfirmationCreatesPendingRecord(t *testing.T) {
	logTest(t)
	calls := []string{}
	item := createPendingDeal(t)
	deals := &dealRepoSpy{calls: &calls, deal: item}
	confirmations := &confirmationRepoSpy{calls: &calls}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: &projectionRepoSpy{}, selections: &selectionRepoSpy{}, outbox: outbox}}

	uc, err := NewRequestDealConfirmation(uow, NoopConfirmationNotifier{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	meta := CommandMeta{CompanyID: item.SupplierID(), UserID: "user-seller"}
	confirmation, err := uc.Execute(context.Background(), meta, item.ID(), RequestDealConfirmationCommand{
		Stage:              deal.DealConfirmationStageConfirmed,
		VerificationMethod: deal.VerificationMethodManual,
		Comment:            "seller says deal is ready",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_deal", "load_pending_confirmation", "save_confirmation", "outbox"})
	if confirmation.Status() != deal.DealConfirmationStatusPending {
		t.Fatalf("expected pending confirmation, got %s", confirmation.Status())
	}
}

func TestApproveDealConfirmationMovesDealStatus(t *testing.T) {
	logTest(t)
	calls := []string{}
	item := createPendingDeal(t)
	confirmation, _, err := item.RequestConfirmation(
		deal.DealConfirmationStageConfirmed,
		item.SupplierID(),
		"user-seller",
		deal.VerificationMethodManual,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("request confirmation error: %v", err)
	}
	deals := &dealRepoSpy{calls: &calls, deal: item}
	confirmations := &confirmationRepoSpy{calls: &calls, confirmation: confirmation}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: &projectionRepoSpy{}, selections: &selectionRepoSpy{}, outbox: outbox}}

	uc, err := NewApproveDealConfirmation(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	meta := CommandMeta{CompanyID: item.CustomerID(), UserID: "buyer-user"}
	if _, err := uc.Execute(context.Background(), meta, item.ID(), confirmation.ID()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_deal", "load_confirmation", "save_confirmation", "save_deal", "outbox"})
	if deals.lastSaved.Status() != deal.DealStatusConfirmed {
		t.Fatalf("expected confirmed status, got %s", deals.lastSaved.Status())
	}
}

func TestRequestDealConfirmationRejectsDuplicatePending(t *testing.T) {
	logTest(t)
	item := createPendingDeal(t)
	existing, _, err := item.RequestConfirmation(
		deal.DealConfirmationStageConfirmed,
		item.SupplierID(),
		"user-seller",
		deal.VerificationMethodManual,
		"",
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatalf("request confirmation error: %v", err)
	}
	deals := &dealRepoSpy{deal: item}
	confirmations := &confirmationRepoSpy{confirmation: existing}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: &projectionRepoSpy{}, selections: &selectionRepoSpy{}, outbox: &outboxSpy{}}}

	uc, err := NewRequestDealConfirmation(uow, NoopConfirmationNotifier{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	meta := CommandMeta{CompanyID: item.SupplierID(), UserID: "user-seller"}
	_, err = uc.Execute(context.Background(), meta, item.ID(), RequestDealConfirmationCommand{
		Stage:              deal.DealConfirmationStageConfirmed,
		VerificationMethod: deal.VerificationMethodManual,
	})
	if !errors.Is(err, deal.ErrConfirmationAlreadyPending) {
		t.Fatalf("expected ErrConfirmationAlreadyPending, got %v", err)
	}
}

func TestUpdateDealPriceUsesMetaActor(t *testing.T) {
	logTest(t)
	calls := []string{}
	item := createPendingDeal(t)
	deals := &dealRepoSpy{calls: &calls, deal: item}
	confirmations := &confirmationRepoSpy{calls: &calls}
	outbox := &outboxSpy{calls: &calls}
	uow := &spyUOW{tx: &spyTx{deals: deals, confirmations: confirmations, projections: &projectionRepoSpy{}, selections: &selectionRepoSpy{}, outbox: outbox}}

	uc, err := NewUpdateDealPrice(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), item.ID(), 130); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertCalls(t, calls, []string{"load_deal", "save_deal", "outbox"})
}

type dealRepoSpy struct {
	calls     *[]string
	deal      *deal.Deal
	lastSaved *deal.Deal
}

func (s *dealRepoSpy) Save(ctx context.Context, item *deal.Deal) error {
	_ = ctx
	s.lastSaved = item
	if s.calls != nil {
		*s.calls = append(*s.calls, "save_deal")
	}
	return nil
}

func (s *dealRepoSpy) GetByID(ctx context.Context, dealID string) (*deal.Deal, error) {
	_ = ctx
	_ = dealID
	if s.calls != nil {
		*s.calls = append(*s.calls, "load_deal")
	}
	if s.deal == nil {
		return nil, ErrDealNotFound
	}
	return s.deal, nil
}

func (s *dealRepoSpy) GetByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error) {
	_ = ctx
	_ = auctionID
	if s.deal == nil {
		return nil, ErrDealNotFound
	}
	return s.deal, nil
}

type projectionRepoSpy struct {
	calls      *[]string
	projection *deal.DealProjection
	lastSaved  *deal.DealProjection
}

type confirmationRepoSpy struct {
	calls        *[]string
	confirmation *deal.DealConfirmation
	lastSaved    *deal.DealConfirmation
}

func (s *confirmationRepoSpy) Save(ctx context.Context, item *deal.DealConfirmation) error {
	_ = ctx
	s.lastSaved = item
	s.confirmation = item
	if s.calls != nil {
		*s.calls = append(*s.calls, "save_confirmation")
	}
	return nil
}

func (s *confirmationRepoSpy) GetByID(ctx context.Context, confirmationID string) (*deal.DealConfirmation, error) {
	_ = ctx
	_ = confirmationID
	if s.calls != nil {
		*s.calls = append(*s.calls, "load_confirmation")
	}
	if s.confirmation == nil {
		return nil, deal.ErrConfirmationNotFound
	}
	return s.confirmation, nil
}

func (s *confirmationRepoSpy) GetPendingByDealAndStage(ctx context.Context, dealID string, stage deal.DealConfirmationStage) (*deal.DealConfirmation, error) {
	_ = ctx
	_ = dealID
	_ = stage
	if s.calls != nil {
		*s.calls = append(*s.calls, "load_pending_confirmation")
	}
	if s.confirmation == nil || s.confirmation.Status() != deal.DealConfirmationStatusPending {
		return nil, deal.ErrConfirmationNotFound
	}
	return s.confirmation, nil
}

func (s *confirmationRepoSpy) ListByDealID(ctx context.Context, dealID string) ([]*deal.DealConfirmation, error) {
	_ = ctx
	_ = dealID
	if s.confirmation == nil {
		return []*deal.DealConfirmation{}, nil
	}
	return []*deal.DealConfirmation{s.confirmation}, nil
}

type selectionRepoSpy struct {
	calls     *[]string
	selection *deal.WinnerSelection
	lastSaved *deal.WinnerSelection
}

func (s *selectionRepoSpy) Save(ctx context.Context, item *deal.WinnerSelection) error {
	_ = ctx
	s.lastSaved = item
	s.selection = item
	if s.calls != nil {
		*s.calls = append(*s.calls, "save_selection")
	}
	return nil
}

func (s *selectionRepoSpy) GetByAuctionID(ctx context.Context, auctionID string) (*deal.WinnerSelection, error) {
	_ = ctx
	_ = auctionID
	if s.calls != nil {
		*s.calls = append(*s.calls, "load_selection")
	}
	if s.selection == nil {
		return nil, deal.ErrSelectionNotFound
	}
	return s.selection, nil
}

func (s *projectionRepoSpy) Save(ctx context.Context, item *deal.DealProjection) error {
	_ = ctx
	s.lastSaved = item
	if s.calls != nil {
		*s.calls = append(*s.calls, "save_projection")
	}
	return nil
}

func (s *projectionRepoSpy) GetByAuctionID(ctx context.Context, auctionID string) (*deal.DealProjection, error) {
	_ = ctx
	_ = auctionID
	if s.calls != nil {
		*s.calls = append(*s.calls, "load_projection")
	}
	if s.projection == nil {
		return nil, deal.ErrProjectionNotFound
	}
	return s.projection, nil
}

type outboxSpy struct {
	calls *[]string
	saved [][]deal.Event
}

func (s *outboxSpy) Add(ctx context.Context, events []deal.Event) error {
	_ = ctx
	if s.calls != nil {
		*s.calls = append(*s.calls, "outbox")
	}
	if len(events) > 0 {
		s.saved = append(s.saved, events)
	}
	return nil
}

type spyTx struct {
	deals         DealRepository
	confirmations DealConfirmationRepository
	projections   ProjectionRepository
	selections    WinnerSelectionRepository
	outbox        OutboxRepository
}

func (s *spyTx) Deals() DealRepository                     { return s.deals }
func (s *spyTx) Confirmations() DealConfirmationRepository { return s.confirmations }
func (s *spyTx) Projections() ProjectionRepository         { return s.projections }
func (s *spyTx) Selections() WinnerSelectionRepository     { return s.selections }
func (s *spyTx) Outbox() OutboxRepository                  { return s.outbox }

type spyUOW struct {
	tx *spyTx
}

func (s *spyUOW) Do(ctx context.Context, fn func(Tx) error) error {
	return fn(s.tx)
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected calls %v, got %v", want, got)
	}
}

func testMeta() CommandMeta {
	return CommandMeta{
		CompanyID:     "company-1",
		UserID:        "user-1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

func createPendingDeal(t *testing.T) *deal.Deal {
	t.Helper()

	factory := deal.NewFactory()
	projection := deal.NewDealProjection(
		"auc-test",
		"sup-test",
		deal.ProductSnapshot{ProductID: "prod-test", Name: "Fish"},
		100,
		time.Now().Add(-time.Hour),
	)
	item, _, err := factory.CreateFromProjection(projection, "buyer-test", 120, time.Now())
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}
	return item
}
