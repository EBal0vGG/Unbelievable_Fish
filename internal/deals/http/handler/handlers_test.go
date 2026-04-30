package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/http"
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
	if s.deal == nil {
		return nil, app.ErrDealNotFound
	}
	return s.deal, nil
}

func (s *spyDealRepo) GetByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error) {
	_ = ctx
	_ = auctionID
	if s.deal == nil {
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
	if s.projection == nil {
		return nil, deal.ErrProjectionNotFound
	}
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

type spyOutbox struct{}

func (spyOutbox) Add(context.Context, []deal.Event) error { return nil }

func TestCreateProjectionHandlerSuccess(t *testing.T) {
	logTest(t)
	repo := &spyProjectionRepo{}
	h := NewCreateProjectionHandler(app.NewCreateProjection(repo))

	body, _ := json.Marshal(httpapi.CreateProjectionRequest{
		AuctionID:  "auc-1",
		SupplierID: "sup-1",
		ProductSnapshot: httpapi.ProductSnapshotDTO{
			ProductID: "prod-1",
			Name:      "Fish",
		},
		StartPrice:  100,
		PublishedAt: time.Now().UTC(),
	})
	req := httptest.NewRequest(http.MethodPost, "/deal-projections", bytes.NewReader(body))
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	logf(t, "status=%d", rec.Code)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestRequestDealConfirmationHandlerMissingCompanyID(t *testing.T) {
	logTest(t)
	repo := &spyDealRepo{deal: createPendingDeal(t)}
	uow := app.NewSimpleUnitOfWork(repo, &spyConfirmationRepo{}, &spyProjectionRepo{}, spySelectionRepo{}, spyOutbox{})
	uc, err := app.NewRequestDealConfirmation(uow, app.NoopConfirmationNotifier{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	h := NewRequestDealConfirmationHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/deals/"+repo.deal.ID()+"/confirmations", bytes.NewBufferString(`{"stage":"confirmed","verification_method":"manual"}`))
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	logf(t, "status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertErrorCode(t, rec, "MISSING_COMPANY_ID")
}

func TestCreateDealHandlerInvalidJSON(t *testing.T) {
	logTest(t)
	repo := &spyDealRepo{}
	projections := &spyProjectionRepo{}
	uow := app.NewSimpleUnitOfWork(repo, &spyConfirmationRepo{}, projections, spySelectionRepo{}, spyOutbox{})
	uc, err := app.NewCreateDealSelectionFromAuctionWon(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	h := NewCreateDealFromAuctionWonHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/deals/from-auction-won", bytes.NewBufferString("{"))
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	logf(t, "status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertErrorCode(t, rec, "INVALID_BODY")
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var resp httpapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp.Code != want {
		t.Fatalf("expected error code %s, got %s", want, resp.Code)
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
