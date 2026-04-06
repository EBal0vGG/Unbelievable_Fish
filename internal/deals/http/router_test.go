package httpapi_test

import (
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

func (s *spyDealRepo) GetByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error) {
	_ = ctx
	_ = auctionID
	return s.deal, nil
}

type spyProjectionRepo struct {
	projection *deal.DealProjection
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
	projectionRepo := &spyProjectionRepo{projection: projection}
	outbox := &spyOutbox{}
	uow := app.NewSimpleUnitOfWork(dealRepo, projectionRepo, spySelectionRepo{}, outbox)

	confirmUC, err := app.NewConfirmDeal(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	router := httpapi.NewRouter(
		handler.NewGetProjectionByAuctionIDHandler(app.NewGetProjectionByAuctionID(projectionRepo)),
		handler.NewGetDealByIDHandler(app.NewGetDealByID(dealRepo)),
		handler.NewGetDealByAuctionIDHandler(app.NewGetDealByAuctionID(dealRepo)),
		handler.NewConfirmDealHandler(confirmUC),
		handler.NewPrepareContractHandler(mustNewPrepareContract(t, uow)),
		handler.NewSignContractHandler(mustNewSignContract(t, uow)),
		handler.NewRequestPaymentHandler(mustNewRequestPayment(t, uow)),
		handler.NewMarkDealPaidHandler(mustNewMarkDealPaid(t, uow)),
		handler.NewRequestShipmentHandler(mustNewRequestShipment(t, uow)),
		handler.NewMarkDealShippedHandler(mustNewMarkDealShipped(t, uow)),
		handler.NewCompleteDealHandler(mustNewCompleteDeal(t, uow)),
		handler.NewCancelDealHandler(mustNewCancelDeal(t, uow)),
		handler.NewUpdateDealPriceHandler(mustNewUpdateDealPrice(t, uow)),
	)

	req := httptest.NewRequest(http.MethodPost, "/deals/"+createdDeal.ID()+"/confirm", nil)
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	logf(t, "status=%d outbox_count=%d", rec.Code, outbox.addCount)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if outbox.addCount != 1 {
		t.Fatalf("expected outbox count 1, got %d", outbox.addCount)
	}
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

func mustNewMarkDealPaid(t *testing.T, uow app.UnitOfWork) *app.MarkDealPaid {
	t.Helper()
	uc, err := app.NewMarkDealPaid(uow)
	if err != nil {
		t.Fatalf("mark paid constructor error: %v", err)
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

func mustNewMarkDealShipped(t *testing.T, uow app.UnitOfWork) *app.MarkDealShipped {
	t.Helper()
	uc, err := app.NewMarkDealShipped(uow)
	if err != nil {
		t.Fatalf("mark shipped constructor error: %v", err)
	}
	return uc
}

func mustNewCompleteDeal(t *testing.T, uow app.UnitOfWork) *app.CompleteDeal {
	t.Helper()
	uc, err := app.NewCompleteDeal(uow)
	if err != nil {
		t.Fatalf("complete deal constructor error: %v", err)
	}
	return uc
}

func mustNewCancelDeal(t *testing.T, uow app.UnitOfWork) *app.CancelDeal {
	t.Helper()
	uc, err := app.NewCancelDeal(uow)
	if err != nil {
		t.Fatalf("cancel deal constructor error: %v", err)
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
