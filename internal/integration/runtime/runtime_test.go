package runtime

import (
	"context"
	"testing"
	"time"

	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	deal "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/eventbus/inmemory"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
)

type dealRepoMem struct {
	data map[string]*deal.Deal
}

func (r *dealRepoMem) Save(_ context.Context, item *deal.Deal) error {
	r.data[item.ID()] = item
	return nil
}

func (r *dealRepoMem) GetByID(_ context.Context, dealID string) (*deal.Deal, error) {
	item, ok := r.data[dealID]
	if !ok {
		return nil, dealsapp.ErrDealNotFound
	}
	return item, nil
}

func (r *dealRepoMem) GetByIDForUpdate(ctx context.Context, dealID string) (*deal.Deal, error) {
	return r.GetByID(ctx, dealID)
}

func (r *dealRepoMem) GetActiveDealByAuctionID(_ context.Context, auctionID string) (*deal.Deal, error) {
	var found []*deal.Deal
	for _, item := range r.data {
		if item.AuctionID() == auctionID && item.Status() != deal.DealStatusCancelled {
			found = append(found, item)
		}
	}
	switch len(found) {
	case 0:
		return nil, dealsapp.ErrDealNotFound
	case 1:
		return found[0], nil
	default:
		return nil, dealsapp.ErrMultipleActiveDealsForAuction
	}
}

type projectionRepoMem struct {
	data map[string]*deal.DealProjection
}

type confirmationRepoMem struct {
	data map[string]*deal.DealConfirmation
}

func (r *confirmationRepoMem) Save(_ context.Context, item *deal.DealConfirmation) error {
	r.data[item.ID()] = item
	return nil
}

func (r *confirmationRepoMem) GetByID(_ context.Context, confirmationID string) (*deal.DealConfirmation, error) {
	item, ok := r.data[confirmationID]
	if !ok {
		return nil, deal.ErrConfirmationNotFound
	}
	return item, nil
}

func (r *confirmationRepoMem) GetPendingByDealAndStage(_ context.Context, dealID string, stage deal.DealConfirmationStage) (*deal.DealConfirmation, error) {
	for _, item := range r.data {
		if item.DealID() == dealID && item.Stage() == stage && item.Status() == deal.DealConfirmationStatusPending {
			return item, nil
		}
	}
	return nil, deal.ErrConfirmationNotFound
}

func (r *confirmationRepoMem) ListByDealID(_ context.Context, dealID string) ([]*deal.DealConfirmation, error) {
	items := make([]*deal.DealConfirmation, 0)
	for _, item := range r.data {
		if item.DealID() == dealID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (r *projectionRepoMem) Save(_ context.Context, item *deal.DealProjection) error {
	r.data[item.AuctionID] = item
	return nil
}

func (r *projectionRepoMem) GetByAuctionID(_ context.Context, auctionID string) (*deal.DealProjection, error) {
	item, ok := r.data[auctionID]
	if !ok {
		return nil, deal.ErrProjectionNotFound
	}
	return item, nil
}

type selectionRepoMem struct {
	data map[string]*deal.WinnerSelection
}

func (r *selectionRepoMem) Save(_ context.Context, item *deal.WinnerSelection) error {
	r.data[item.AuctionID] = item
	return nil
}

func (r *selectionRepoMem) GetByAuctionID(_ context.Context, auctionID string) (*deal.WinnerSelection, error) {
	item, ok := r.data[auctionID]
	if !ok {
		return nil, deal.ErrSelectionNotFound
	}
	return item, nil
}

func (r *selectionRepoMem) GetByAuctionIDForUpdate(ctx context.Context, auctionID string) (*deal.WinnerSelection, error) {
	return r.GetByAuctionID(ctx, auctionID)
}

type outboxMem struct {
	events []deal.Event
}

func (o *outboxMem) Add(_ context.Context, events []deal.Event) error {
	o.events = append(o.events, events...)
	return nil
}

type fakeDealLister struct {
	ids []string
}

func (l fakeDealLister) ListExpiredForFallback(_ context.Context, _ time.Time, _ int) ([]string, error) {
	return l.ids, nil
}

type fakeAuctionLister struct{}

func (fakeAuctionLister) ListExpired(_ context.Context, _ time.Time, _ int) ([]tradingapp.AuctionID, error) {
	return nil, nil
}

func TestRunCancelExpiredDealsCancelsDeal(t *testing.T) {
	ctx := context.Background()
	projections := &projectionRepoMem{data: map[string]*deal.DealProjection{}}
	confirmations := &confirmationRepoMem{data: map[string]*deal.DealConfirmation{}}
	deals := &dealRepoMem{data: map[string]*deal.Deal{}}
	selections := &selectionRepoMem{data: map[string]*deal.WinnerSelection{}}
	outbox := &outboxMem{}
	uow := dealsapp.NewSimpleUnitOfWork(deals, confirmations, projections, selections, outbox)

	factory := deal.NewFactory()
	projection := deal.NewDealProjection(
		"auc-deadline",
		"seller-1",
		deal.ProductSnapshot{ProductID: "prod-1", Name: "Fish"},
		100,
		time.Now().Add(-2*time.Hour),
	)
	item, _, err := factory.CreateFromProjection(projection, "buyer-1", 120, time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("create deal: %v", err)
	}
	if err := deals.Save(ctx, item); err != nil {
		t.Fatalf("save deal: %v", err)
	}

	cancelUC, err := dealsapp.NewCancelDeal(uow)
	if err != nil {
		t.Fatalf("cancel constructor: %v", err)
	}
	r := &Runtime{
		cancelDeal: cancelUC,
		dealLister: fakeDealLister{ids: []string{item.ID()}},
	}

	if err := r.RunCancelExpiredDeals(ctx, time.Now().UTC(), 100); err != nil {
		t.Fatalf("run cancel expired deals: %v", err)
	}

	updated, err := deals.GetByID(ctx, item.ID())
	if err != nil {
		t.Fatalf("load updated deal: %v", err)
	}
	if updated.Status() != deal.DealStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", updated.Status())
	}
}

func TestDealCancelledEventMovesToNextWinner(t *testing.T) {
	ctx := context.Background()
	projections := &projectionRepoMem{data: map[string]*deal.DealProjection{}}
	confirmations := &confirmationRepoMem{data: map[string]*deal.DealConfirmation{}}
	deals := &dealRepoMem{data: map[string]*deal.Deal{}}
	selections := &selectionRepoMem{data: map[string]*deal.WinnerSelection{}}
	outbox := &outboxMem{}
	uow := dealsapp.NewSimpleUnitOfWork(deals, confirmations, projections, selections, outbox)

	createSelection, err := dealsapp.NewCreateDealSelectionFromAuctionWon(uow)
	if err != nil {
		t.Fatalf("selection constructor: %v", err)
	}
	handleDeclined, err := dealsapp.NewHandleDealDeclined(uow)
	if err != nil {
		t.Fatalf("declined constructor: %v", err)
	}

	auctionID := "auc-fallback"
	if err := projections.Save(ctx, deal.NewDealProjection(
		auctionID,
		"seller-1",
		deal.ProductSnapshot{ProductID: "prod-1", Name: "Fish"},
		100,
		time.Now().Add(-2*time.Hour),
	)); err != nil {
		t.Fatalf("save projection: %v", err)
	}

	if err := createSelection.Execute(ctx, dealsapp.CommandMeta{}, auctionID, []string{"buyer-1", "buyer-2"}, 140, time.Now()); err != nil {
		t.Fatalf("create selection deal: %v", err)
	}
	initial, err := deals.GetActiveDealByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("load initial deal: %v", err)
	}
	if initial.CustomerID() != "buyer-1" {
		t.Fatalf("expected first deal for buyer-1, got %s", initial.CustomerID())
	}

	cancelUC, err := dealsapp.NewCancelDeal(uow)
	if err != nil {
		t.Fatalf("cancel deal constructor: %v", err)
	}
	cancelMeta := dealsapp.CommandMeta{CompanyID: "buyer-1", UserID: "user-1", CorrelationID: "test", CausationID: "test"}
	if err := cancelUC.Execute(ctx, cancelMeta, initial.ID(), "manual cancellation"); err != nil {
		t.Fatalf("cancel first deal: %v", err)
	}
	var cancelledEvt deal.DealCancelled
	for _, e := range outbox.events {
		if dc, ok := e.(deal.DealCancelled); ok {
			cancelledEvt = dc
		}
	}
	if cancelledEvt.DealID == "" {
		t.Fatal("expected DealCancelled in outbox after cancel")
	}

	bus := inmemory.NewBus()
	registerIntegrationHandlers(
		bus,
		Dependencies{AuctionLister: fakeAuctionLister{}, DealLister: fakeDealLister{}},
		nil,
		nil,
		createSelection,
		handleDeclined,
		nil,
	)

	if err := bus.Publish(ctx, events.Envelope{
		Type:       "deals.DealCancelled",
		Payload:    cancelledEvt,
		OccurredAt: time.Now().UTC(),
		Meta:       map[string]string{"auction_id": auctionID},
	}); err != nil {
		t.Fatalf("publish deal cancelled: %v", err)
	}

	next, err := deals.GetActiveDealByAuctionID(ctx, auctionID)
	if err != nil {
		t.Fatalf("load next deal: %v", err)
	}
	if next.CustomerID() != "buyer-2" {
		t.Fatalf("expected fallback deal for buyer-2, got %s", next.CustomerID())
	}
}
