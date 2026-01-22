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
	auction    *auction.Auction
	loadCount  int
	saveCount  int
}

func (s *spyRepo) Load(ctx context.Context, id app.AuctionID) (*auction.Auction, error) {
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

func TestCommandFlowSmoke(t *testing.T) {
	a := auction.NewAuction("a-1")
	_, _ = a.Publish()

	repo := &spyRepo{auction: a}
	publisher := &spyPublisher{}

	createUC := app.NewCreateAuction(repo, publisher)
	publishUC := app.NewPublishAuction(repo, publisher)
	placeBidUC := app.NewPlaceBid(repo, publisher)
	closeUC := app.NewCloseAuction(repo, publisher)
	cancelUC := app.NewCancelAuction(repo, publisher)

	router := httpapi.NewRouter(
		handler.NewCreateAuctionHandler(createUC),
		handler.NewPublishAuctionHandler(publishUC),
		handler.NewPlaceBidHandler(placeBidUC),
		handler.NewCloseAuctionHandler(closeUC),
		handler.NewCancelAuctionHandler(cancelUC),
	)

	body, _ := json.Marshal(httpapi.PlaceBidRequest{
		Amount:   100,
		PlacedAt: time.Now().UTC(),
	})
	req := httptest.NewRequest(http.MethodPost, "/auctions/a-1/bids", bytes.NewReader(body))
	req.Header.Set("X-Company-ID", "company-1")
	req.Header.Set("X-User-ID", "user-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
	}
	if repo.loadCount != 1 {
		t.Fatalf("expected Load to be called once, got %d", repo.loadCount)
	}
	if repo.saveCount != 1 {
		t.Fatalf("expected Save to be called once, got %d", repo.saveCount)
	}
	if publisher.publishCount != 1 {
		t.Fatalf("expected Publish to be called once, got %d", publisher.publishCount)
	}
}
