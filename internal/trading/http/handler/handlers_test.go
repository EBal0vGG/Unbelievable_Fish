package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	return s.auction, nil
}

func (s *spyRepo) Save(ctx context.Context, a *auction.Auction) error {
	s.saveCount++
	s.lastSaved = a
	return nil
}

type spyPublisher struct {
	publishCount int
	lastEvents   []auction.Event
}

func (s *spyPublisher) Publish(ctx context.Context, events []auction.Event) error {
	s.publishCount++
	s.lastEvents = events
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

func TestCreateAuctionHandlerSuccess(t *testing.T) {
	logTest(t)
	repo := &spyRepo{}
	publisher := &spyPublisher{}
	uc := app.NewCreateAuction(repo, publisher)
	handler := NewCreateAuctionHandler(uc)

	startsAt := time.Now().Add(-time.Hour).UTC()
	endsAt := startsAt.Add(time.Hour)
	logf(t, "request lot_id=%s starts_at=%s ends_at=%s", "lot-1", startsAt, endsAt)
	body, _ := json.Marshal(httpapi.CreateAuctionRequest{
		LotID:    "lot-1",
		StartsAt: startsAt,
		EndsAt:   endsAt,
	})
	req := httptest.NewRequest(http.MethodPost, "/auctions", bytes.NewReader(body))
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	logf(t, "status=%d save_count=%d", rec.Code, repo.saveCount)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if repo.saveCount != 1 {
		t.Fatalf("expected save to be called once, got %d", repo.saveCount)
	}
}

func TestPublishAuctionHandlerMissingCompanyID(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("a-1", "lot-1", startsAt, endsAt)
	repo := &spyRepo{auction: a}
	publisher := &spyPublisher{}
	uc := app.NewPublishAuction(repo, publisher)
	handler := NewPublishAuctionHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/auctions/a-1/publish", nil)
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
	publisher := &spyPublisher{}
	bidRepo := &spyBidRepo{}
	uc := app.NewPlaceBid(repo, bidRepo, publisher)
	handler := NewPlaceBidHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/auctions/a-1/bids", bytes.NewBufferString("{"))
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
	publisher := &spyPublisher{}
	uc := app.NewPublishAuction(repo, publisher)
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
