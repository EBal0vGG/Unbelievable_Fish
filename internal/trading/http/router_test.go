package httpapi_test

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
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/http/handler"
)

type spyRepo struct {
	auction   *auction.Auction
	loadCount int
	saveCount int
}

func (s *spyRepo) Load(ctx context.Context, id app.AuctionID) (*auction.Auction, error) {
	s.loadCount++
	return s.auction, nil
}

func (s *spyRepo) LoadForUpdate(ctx context.Context, id app.AuctionID) (*auction.Auction, error) {
	s.loadCount++
	return s.auction, nil
}

func (s *spyRepo) Save(ctx context.Context, a *auction.Auction) error {
	s.saveCount++
	return nil
}

type spyPublisher struct {
	publishCount int
}

func (s *spyPublisher) Publish(ctx context.Context, events []auction.Event) error {
	s.publishCount++
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
	return []auction.Bid{}, nil
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

func TestCommandFlowSmoke(t *testing.T) {
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

	placeBidUC, err := app.NewPlaceBid(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	router := httpapi.NewRouter(
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		handler.NewPlaceBidHandler(placeBidUC),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
		http.NotFoundHandler(),
	)

	body, _ := json.Marshal(httpapi.PlaceBidRequest{
		Amount:   100,
		PlacedAt: endsAt.Add(-time.Minute).UTC(),
	})
	logf(t, "request auction_id=%s amount=%d placed_at=%s", "a-1", 100, endsAt.Add(-time.Minute).UTC())
	req := httptest.NewRequest(http.MethodPost, "/auctions/a-1/bids", bytes.NewReader(body))
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	logf(t, "status=%d load=%d save=%d outbox=%d bid_save=%d", rec.Code, repo.loadCount, repo.saveCount, outbox.saveCount, bidRepo.saveCount)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if repo.loadCount != 1 {
		t.Fatalf("expected Load to be called once, got %d", repo.loadCount)
	}
	if repo.saveCount != 1 {
		t.Fatalf("expected Save to be called once, got %d", repo.saveCount)
	}
	if outbox.saveCount != 1 {
		t.Fatalf("expected Outbox.Save to be called once, got %d", outbox.saveCount)
	}
}
