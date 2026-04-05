package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

func TestAuctionRepositorySaveAndLoad(t *testing.T) {
	db := openIntegrationDB(t, "auction-repo")
	repo := NewAuctionRepository(db)

	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	item, err := auction.NewAuction("auc-1", "lot-1", startsAt, endsAt)
	if err != nil {
		t.Fatalf("new auction error: %v", err)
	}
	if err := repo.Save(context.Background(), item); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := repo.Load(context.Background(), app.AuctionID("auc-1"))
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.ID != "auc-1" || loaded.LotID != "lot-1" {
		t.Fatalf("unexpected loaded auction: %+v", loaded)
	}
	if loaded.State() != auction.StateDraft {
		t.Fatalf("expected draft state, got %s", loaded.State())
	}
}

func TestBidRepositorySaveAndTopBids(t *testing.T) {
	db := openIntegrationDB(t, "bid-repo")
	repo := NewBidRepository(db)

	auctionID := app.AuctionID("auc-2")
	now := time.Now()
	bid1, _ := auction.NewBid("c1", 100, now.Add(time.Second))
	bid2, _ := auction.NewBid("c2", 150, now)
	if err := repo.Save(context.Background(), auctionID, bid1); err != nil {
		t.Fatalf("save bid1 error: %v", err)
	}
	if err := repo.Save(context.Background(), auctionID, bid2); err != nil {
		t.Fatalf("save bid2 error: %v", err)
	}
	bids, err := repo.TopBids(context.Background(), auctionID)
	if err != nil {
		t.Fatalf("top bids error: %v", err)
	}
	if len(bids) != 2 {
		t.Fatalf("expected 2 bids, got %d", len(bids))
	}
	if bids[0].BidderCompanyID() != "c2" {
		t.Fatalf("expected c2 first, got %s", bids[0].BidderCompanyID())
	}
}

func TestWinnersRepositorySave(t *testing.T) {
	db := openIntegrationDB(t, "winners-repo")
	repo := NewWinnersRepository(db)

	auctionID := app.AuctionID("auc-3")
	now := time.Now()
	winners := []app.WinnerRecord{
		{Place: 1, CompanyID: "c1", Amount: 200, PlacedAt: now},
		{Place: 2, CompanyID: "c2", Amount: 150, PlacedAt: now.Add(time.Second)},
	}
	if err := repo.Save(context.Background(), auctionID, winners); err != nil {
		t.Fatalf("save winners error: %v", err)
	}

	store := tradingTestDriver.store("winners-repo")
	if len(store.winners) != 2 {
		t.Fatalf("expected 2 winners, got %d", len(store.winners))
	}
}
