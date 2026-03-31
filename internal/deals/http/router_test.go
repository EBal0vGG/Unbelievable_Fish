package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"unbelievable_fish/internal/deals/app"
	"unbelievable_fish/internal/deals/deal"
	httpapi "unbelievable_fish/internal/deals/http"
	"unbelievable_fish/internal/deals/http/handler"
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

type spyPublisher struct {
	publishCount int
}

func (s *spyPublisher) Publish(ctx context.Context, events []deal.Event) error {
	_ = ctx
	_ = events
	s.publishCount++
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
	dealRepo := &spyDealRepo{}
	projectionRepo := &spyProjectionRepo{projection: projection}
	publisher := &spyPublisher{}

	createDealUC := app.NewCreateDealFromAuctionWon(dealRepo, projectionRepo, publisher)
	confirmUC := app.NewConfirmDeal(dealRepo, publisher)

	router := httpapi.NewRouter(
		handler.NewCreateProjectionHandler(app.NewCreateProjection(projectionRepo)),
		handler.NewGetProjectionByAuctionIDHandler(app.NewGetProjectionByAuctionID(projectionRepo)),
		handler.NewCreateDealFromAuctionWonHandler(createDealUC),
		handler.NewGetDealByIDHandler(app.NewGetDealByID(dealRepo)),
		handler.NewGetDealByAuctionIDHandler(app.NewGetDealByAuctionID(dealRepo)),
		handler.NewConfirmDealHandler(confirmUC),
		handler.NewPrepareContractHandler(app.NewPrepareContract(dealRepo, publisher)),
		handler.NewSignContractHandler(app.NewSignContract(dealRepo, publisher)),
		handler.NewRequestPaymentHandler(app.NewRequestPayment(dealRepo, publisher)),
		handler.NewMarkDealPaidHandler(app.NewMarkDealPaid(dealRepo, publisher)),
		handler.NewRequestShipmentHandler(app.NewRequestShipment(dealRepo, publisher)),
		handler.NewMarkDealShippedHandler(app.NewMarkDealShipped(dealRepo, publisher)),
		handler.NewCompleteDealHandler(app.NewCompleteDeal(dealRepo, publisher)),
		handler.NewCancelDealHandler(app.NewCancelDeal(dealRepo, publisher)),
		handler.NewUpdateDealPriceHandler(app.NewUpdateDealPrice(dealRepo, publisher)),
	)

	body, _ := json.Marshal(httpapi.CreateDealFromAuctionWonRequest{
		AuctionID:       "auc-1",
		WinnerCompanyID: "buyer-1",
		FinalPrice:      120,
		WonAt:           now,
	})
	req := httptest.NewRequest(http.MethodPost, "/deals/from-auction-won", bytes.NewReader(body))
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	logf(t, "status=%d publish_count=%d", rec.Code, publisher.publishCount)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if dealRepo.deal == nil {
		t.Fatal("expected deal to be saved")
	}
	if publisher.publishCount != 1 {
		t.Fatalf("expected publish count 1, got %d", publisher.publishCount)
	}
}
