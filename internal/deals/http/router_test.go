package httpapi_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http/handler"
)

type spyDealRepo struct {
	deal *deal.Deal
}

func (s *spyDealRepo) Save(ctx context.Context, item *deal.Deal) error {
	_ = ctx
	s.deal = item
	return nil
}

func (s *spyDealRepo) GetByID(ctx context.Context, dealID string) (*deal.Deal, error) {
	_ = ctx
	_ = dealID
	return s.deal, nil
}

func (s *spyDealRepo) GetByIDForUpdate(ctx context.Context, dealID string) (*deal.Deal, error) {
	return s.GetByID(ctx, dealID)
}

func (s *spyDealRepo) GetActiveDealByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error) {
	_ = ctx
	_ = auctionID
	if s.deal == nil || s.deal.Status() == deal.DealStatusCancelled {
		return nil, app.ErrDealNotFound
	}
	return s.deal, nil
}

type spyProjectionRepo struct {
	projection *deal.DealProjection
}

type spyConfirmationRepo struct {
	confirmation *deal.DealConfirmation
}

func (s *spyConfirmationRepo) Save(ctx context.Context, item *deal.DealConfirmation) error {
	_ = ctx
	s.confirmation = item
	return nil
}

func (s *spyConfirmationRepo) GetByID(ctx context.Context, confirmationID string) (*deal.DealConfirmation, error) {
	_ = ctx
	_ = confirmationID
	if s.confirmation == nil {
		return nil, deal.ErrConfirmationNotFound
	}
	return s.confirmation, nil
}

func (s *spyConfirmationRepo) GetPendingByDealAndStage(ctx context.Context, dealID string, stage deal.DealConfirmationStage) (*deal.DealConfirmation, error) {
	_ = ctx
	_ = dealID
	_ = stage
	if s.confirmation == nil || s.confirmation.Status() != deal.DealConfirmationStatusPending {
		return nil, deal.ErrConfirmationNotFound
	}
	return s.confirmation, nil
}

func (s *spyConfirmationRepo) ListByDealID(ctx context.Context, dealID string) ([]*deal.DealConfirmation, error) {
	_ = ctx
	_ = dealID
	if s.confirmation == nil {
		return []*deal.DealConfirmation{}, nil
	}
	return []*deal.DealConfirmation{s.confirmation}, nil
}

func (s *spyProjectionRepo) Save(ctx context.Context, item *deal.DealProjection) error {
	_ = ctx
	s.projection = item
	return nil
}

func (s *spyProjectionRepo) GetByAuctionID(ctx context.Context, auctionID string) (*deal.DealProjection, error) {
	_ = ctx
	_ = auctionID
	return s.projection, nil
}

type spySelectionRepo struct{}

func (spySelectionRepo) Save(ctx context.Context, item *deal.WinnerSelection) error {
	_ = ctx
	_ = item
	return nil
}

func (spySelectionRepo) GetByAuctionID(ctx context.Context, auctionID string) (*deal.WinnerSelection, error) {
	_ = ctx
	_ = auctionID
	return nil, deal.ErrSelectionNotFound
}

func (spySelectionRepo) GetByAuctionIDForUpdate(ctx context.Context, auctionID string) (*deal.WinnerSelection, error) {
	return spySelectionRepo{}.GetByAuctionID(ctx, auctionID)
}

type spyOutbox struct {
	addCount int
}

func (s *spyOutbox) Add(ctx context.Context, events []deal.Event) error {
	_ = ctx
	_ = events
	s.addCount++
	return nil
}

func TestCommandFlowSmoke(t *testing.T) {
	logTest(t)
	now := time.Now().UTC()
	projection := deal.NewDealProjection(
		"auc-1",
		"sup-1",
		deal.ProductSnapshot{ProductID: "prod-1", Name: "Fish"},
		100,
		now.Add(-time.Hour),
	)
	factory := deal.NewFactory()
	createdDeal, _, err := factory.CreateFromProjection(projection, "buyer-1", 120, now)
	if err != nil {
		t.Fatalf("unexpected factory error: %v", err)
	}
	dealRepo := &spyDealRepo{}
	dealRepo.deal = createdDeal
	confirmationRepo := &spyConfirmationRepo{}
	projectionRepo := &spyProjectionRepo{projection: projection}
	outbox := &spyOutbox{}
	uow := app.NewSimpleUnitOfWork(dealRepo, confirmationRepo, projectionRepo, spySelectionRepo{}, outbox)

	requestConfirmationUC, err := app.NewRequestDealConfirmation(uow, app.NoopConfirmationNotifier{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	approveConfirmationUC, err := app.NewApproveDealConfirmation(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	rejectConfirmationUC, err := app.NewRejectDealConfirmation(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	router := httpapi.NewRouter(httpapi.Handlers{
		GetDealProjection:   handler.NewGetProjectionByAuctionIDHandler(app.NewGetProjectionByAuctionID(projectionRepo)),
		GetDealByAuction:    handler.NewGetDealByAuctionIDHandler(app.NewGetDealByAuctionID(uow)),
		GetDeal:             handler.NewGetDealByIDHandler(app.NewGetDealByID(dealRepo)),
		GetConfirmations:    handler.NewGetDealConfirmationsHandler(app.NewGetDealConfirmations(dealRepo, confirmationRepo)),
		RequestConfirmation: handler.NewRequestDealConfirmationHandler(requestConfirmationUC),
		ApproveConfirmation: handler.NewApproveDealConfirmationHandler(approveConfirmationUC),
		RejectConfirmation:  handler.NewRejectDealConfirmationHandler(rejectConfirmationUC),
		PrepareContract:     handler.NewPrepareContractHandler(mustNewPrepareContract(t, uow)),
		SignContract:        handler.NewSignContractHandler(mustNewSignContract(t, uow)),
		RequestPayment:      handler.NewRequestPaymentHandler(mustNewRequestPayment(t, uow)),
		RequestShipment:     handler.NewRequestShipmentHandler(mustNewRequestShipment(t, uow)),
		UpdateDealPrice:     handler.NewUpdateDealPriceHandler(mustNewUpdateDealPrice(t, uow)),
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/deals/"+createdDeal.ID()+"/confirmations",
		bytes.NewBufferString(`{"stage":"confirmed","verification_method":"manual","comment":"seller ready"}`),
	)
	req.Header.Set("X-Company-ID", createdDeal.SupplierID())
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	logf(t, "status=%d outbox_count=%d", rec.Code, outbox.addCount)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
	if outbox.addCount != 1 {
		t.Fatalf("expected outbox count 1, got %d", outbox.addCount)
	}
}

func TestDirectStatusEndpointRemoved(t *testing.T) {
	dealRepo := &spyDealRepo{deal: createDealForRouter(t)}
	confirmationRepo := &spyConfirmationRepo{}
	projectionRepo := &spyProjectionRepo{}
	outbox := &spyOutbox{}
	uow := app.NewSimpleUnitOfWork(dealRepo, confirmationRepo, projectionRepo, spySelectionRepo{}, outbox)
	requestConfirmationUC, _ := app.NewRequestDealConfirmation(uow, app.NoopConfirmationNotifier{})
	approveConfirmationUC, _ := app.NewApproveDealConfirmation(uow)
	rejectConfirmationUC, _ := app.NewRejectDealConfirmation(uow)

	router := httpapi.NewRouter(httpapi.Handlers{
		GetDealProjection:   handler.NewGetProjectionByAuctionIDHandler(app.NewGetProjectionByAuctionID(projectionRepo)),
		GetDealByAuction:    handler.NewGetDealByAuctionIDHandler(app.NewGetDealByAuctionID(uow)),
		GetDeal:             handler.NewGetDealByIDHandler(app.NewGetDealByID(dealRepo)),
		GetConfirmations:    handler.NewGetDealConfirmationsHandler(app.NewGetDealConfirmations(dealRepo, confirmationRepo)),
		RequestConfirmation: handler.NewRequestDealConfirmationHandler(requestConfirmationUC),
		ApproveConfirmation: handler.NewApproveDealConfirmationHandler(approveConfirmationUC),
		RejectConfirmation:  handler.NewRejectDealConfirmationHandler(rejectConfirmationUC),
		PrepareContract:     handler.NewPrepareContractHandler(mustNewPrepareContract(t, uow)),
		SignContract:        handler.NewSignContractHandler(mustNewSignContract(t, uow)),
		RequestPayment:      handler.NewRequestPaymentHandler(mustNewRequestPayment(t, uow)),
		RequestShipment:     handler.NewRequestShipmentHandler(mustNewRequestShipment(t, uow)),
		UpdateDealPrice:     handler.NewUpdateDealPriceHandler(mustNewUpdateDealPrice(t, uow)),
	})

	req := httptest.NewRequest(http.MethodPost, "/deals/"+dealRepo.deal.ID()+"/confirm", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func createDealForRouter(t *testing.T) *deal.Deal {
	t.Helper()
	now := time.Now().UTC()
	projection := deal.NewDealProjection(
		"auc-router",
		"sup-router",
		deal.ProductSnapshot{ProductID: "prod-router", Name: "Fish"},
		100,
		now.Add(-time.Hour),
	)
	factory := deal.NewFactory()
	item, _, err := factory.CreateFromProjection(projection, "buyer-router", 120, now)
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}
	return item
}

func mustNewPrepareContract(t *testing.T, uow app.UnitOfWork) *app.PrepareContract {
	t.Helper()
	uc, err := app.NewPrepareContract(uow)
	if err != nil {
		t.Fatalf("prepare contract constructor error: %v", err)
	}
	return uc
}

func mustNewSignContract(t *testing.T, uow app.UnitOfWork) *app.SignContract {
	t.Helper()
	uc, err := app.NewSignContract(uow)
	if err != nil {
		t.Fatalf("sign contract constructor error: %v", err)
	}
	return uc
}

func mustNewRequestPayment(t *testing.T, uow app.UnitOfWork) *app.RequestPayment {
	t.Helper()
	uc, err := app.NewRequestPayment(uow)
	if err != nil {
		t.Fatalf("request payment constructor error: %v", err)
	}
	return uc
}

func mustNewRequestShipment(t *testing.T, uow app.UnitOfWork) *app.RequestShipment {
	t.Helper()
	uc, err := app.NewRequestShipment(uow)
	if err != nil {
		t.Fatalf("request shipment constructor error: %v", err)
	}
	return uc
}

func mustNewUpdateDealPrice(t *testing.T, uow app.UnitOfWork) *app.UpdateDealPrice {
	t.Helper()
	uc, err := app.NewUpdateDealPrice(uow)
	if err != nil {
		t.Fatalf("update price constructor error: %v", err)
	}
	return uc
}
