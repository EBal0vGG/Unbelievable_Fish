package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	identityauth "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/auth"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
	httpapi "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http"
)

type spyRepo struct {
	auction   *auction.Auction
	loadCount int
	saveCount int
	lastSaved *auction.Auction
}

func (s *spyRepo) Load(ctx context.Context, id app.AuctionID) (*auction.Auction, error) {
	s.loadCount++
	if s.auction == nil {
		return nil, app.ErrNotFound
	}
	return s.auction, nil
}

func (s *spyRepo) LoadForUpdate(ctx context.Context, id app.AuctionID) (*auction.Auction, error) {
	s.loadCount++
	if s.auction == nil {
		return nil, app.ErrNotFound
	}
	return s.auction, nil
}

func (s *spyRepo) Save(ctx context.Context, a *auction.Auction) error {
	s.saveCount++
	s.lastSaved = a
	return nil
}

type spyBidRepo struct {
	saveCount int
}

func (s *spyBidRepo) Save(ctx context.Context, auctionID app.AuctionID, bid auction.Bid) error {
	s.saveCount++
	return nil
}

func (s *spyBidRepo) TopBids(ctx context.Context, auctionID app.AuctionID) ([]auction.Bid, error) {
	return nil, nil
}

type spyOutbox struct {
	saveCount int
}

func (s *spyOutbox) Add(ctx context.Context, events []auction.Event) error {
	s.saveCount++
	return nil
}

type spyTx struct {
	repo    *spyRepo
	bids    *spyBidRepo
	outbox  *spyOutbox
	winners *spyWinners
}

func (s *spyTx) Auctions() app.AuctionRepository       { return s.repo }
func (s *spyTx) Bids() app.BidRepository               { return s.bids }
func (s *spyTx) Outbox() app.OutboxRepository          { return s.outbox }
func (s *spyTx) Winners() app.AuctionWinnersRepository { return s.winners }

type spyUOW struct {
	tx *spyTx
}

func (s *spyUOW) Do(ctx context.Context, fn func(app.Tx) error) error {
	return fn(s.tx)
}

type spyWinners struct {
	saveCount int
}

func (s *spyWinners) Save(ctx context.Context, auctionID app.AuctionID, winners []app.WinnerRecord) error {
	s.saveCount++
	return nil
}

func TestPublishAuctionHandlerMissingCompanyID(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("a-1", "lot-1", startsAt, endsAt)
	repo := &spyRepo{auction: a}
	bidRepo := &spyBidRepo{}
	outbox := &spyOutbox{}
	winners := &spyWinners{}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	uc, err := app.NewPublishAuction(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewPublishAuctionHandler(uc)

	req := withURLParam(httptest.NewRequest(http.MethodPost, "/auctions/a-1/publish", nil), "auctionID", "a-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logf(t, "status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertErrorCode(t, rec, "MISSING_COMPANY_ID")
}

func TestPlaceBidHandlerInvalidJSON(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("a-1", "lot-1", startsAt, endsAt)
	repo := &spyRepo{auction: a}
	bidRepo := &spyBidRepo{}
	outbox := &spyOutbox{}
	winners := &spyWinners{}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	uc, err := app.NewPlaceBid(uow, app.NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewPlaceBidHandler(uc)

	req := withURLParam(httptest.NewRequest(http.MethodPost, "/auctions/a-1/bids", bytes.NewBufferString("{")), "auctionID", "a-1")
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logf(t, "status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertErrorCode(t, rec, "INVALID_BODY")
}

func TestPublishAuctionHandlerInvalidPath(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("a-1", "lot-1", startsAt, endsAt)
	repo := &spyRepo{auction: a}
	bidRepo := &spyBidRepo{}
	outbox := &spyOutbox{}
	winners := &spyWinners{}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	uc, err := app.NewPublishAuction(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewPublishAuctionHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/auctions//publish", nil)
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logf(t, "status=%d body=%s", rec.Code, rec.Body.String())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	assertErrorCode(t, rec, "INVALID_PATH")
}

func TestPlaceBidHandlerUsesIdentityFromContext(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	a, _ := auction.NewAuction("a-1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()

	repo := &spyRepo{auction: a}
	bidRepo := &spyBidRepo{}
	outbox := &spyOutbox{}
	winners := &spyWinners{}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	uc, err := app.NewPlaceBid(uow, app.NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	handler := NewPlaceBidHandler(uc)

	body, _ := json.Marshal(httpapi.PlaceBidRequest{Amount: 100})
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/auctions/a-1/bids", bytes.NewReader(body)), "auctionID", "a-1")
	req = req.WithContext(identityauth.WithIdentity(req.Context(), identityauth.Identity{
		UserID:    "user-1",
		CompanyID: "company-1",
		Role:      identity.RoleBuyer,
	}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if bidRepo.saveCount != 1 {
		t.Fatalf("expected bid save once, got %d", bidRepo.saveCount)
	}
}

func TestProtectedPlaceBidEndpointWithValidToken(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	a, _ := auction.NewAuction("a-1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()

	repo := &spyRepo{auction: a}
	bidRepo := &spyBidRepo{}
	outbox := &spyOutbox{}
	winners := &spyWinners{}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	uc, err := app.NewPlaceBid(uow, app.NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	tokenProvider := identityauth.NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleBuyer, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := tokenProvider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	protected := identityauth.NewMiddleware(tokenProvider, func(w http.ResponseWriter, r *http.Request, err error) {
		httpErr := httpapi.MapError(err)
		w.WriteHeader(httpErr.Status)
	}).RequireRole(identity.RoleBuyer, NewPlaceBidHandler(uc))

	body, _ := json.Marshal(httpapi.PlaceBidRequest{Amount: 100})
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/auctions/a-1/bids", bytes.NewReader(body)), "auctionID", "a-1")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestProtectedPlaceBidEndpointForbiddenForWrongRole(t *testing.T) {
	logTest(t)
	repo := &spyRepo{}
	bidRepo := &spyBidRepo{}
	outbox := &spyOutbox{}
	winners := &spyWinners{}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	uc, err := app.NewPlaceBid(uow, app.NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	tokenProvider := identityauth.NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-1", "company-1", "Alice", identity.RoleSeller, "alice@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := tokenProvider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	protected := identityauth.NewMiddleware(tokenProvider, func(w http.ResponseWriter, r *http.Request, err error) {
		httpErr := httpapi.MapError(err)
		w.WriteHeader(httpErr.Status)
	}).RequireRole(identity.RoleBuyer, NewPlaceBidHandler(uc))

	body, _ := json.Marshal(httpapi.PlaceBidRequest{Amount: 100})
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/auctions/a-1/bids", bytes.NewReader(body)), "auctionID", "a-1")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestProtectedPlaceBidEndpointAllowsBuyerSellerRole(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	a, _ := auction.NewAuction("a-2", "lot-2", startsAt, endsAt)
	_, _ = a.Publish()

	repo := &spyRepo{auction: a}
	bidRepo := &spyBidRepo{}
	outbox := &spyOutbox{}
	winners := &spyWinners{}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	uc, err := app.NewPlaceBid(uow, app.NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	tokenProvider := identityauth.NewTokenProvider("secret", time.Hour)
	user, err := identity.NewUser("user-2", "company-2", "Bob", identity.RoleBuyerSeller, "bob@example.com", "hash")
	if err != nil {
		t.Fatalf("unexpected user error: %v", err)
	}
	token, err := tokenProvider.Generate(user)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	protected := identityauth.NewMiddleware(tokenProvider, func(w http.ResponseWriter, r *http.Request, err error) {
		httpErr := httpapi.MapError(err)
		w.WriteHeader(httpErr.Status)
	}).RequireRole(identity.RoleBuyer, NewPlaceBidHandler(uc))

	body, _ := json.Marshal(httpapi.PlaceBidRequest{Amount: 100})
	req := withURLParam(httptest.NewRequest(http.MethodPost, "/auctions/a-2/bids", bytes.NewReader(body)), "auctionID", "a-2")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var resp httpapi.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp.Code != want {
		t.Fatalf("expected error code %s, got %s", want, resp.Code)
	}
}
