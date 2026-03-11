package app

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type spyRepo struct {
	auction   *auction.Auction
	calls     *[]string
	loadCount int
	saveCount int
	lastSaved *auction.Auction
}

func (s *spyRepo) Load(ctx context.Context, id AuctionID) (*auction.Auction, error) {
	s.loadCount++
	*s.calls = append(*s.calls, "load")
	return s.auction, nil
}

func (s *spyRepo) Save(ctx context.Context, a *auction.Auction) error {
	s.saveCount++
	s.lastSaved = a
	*s.calls = append(*s.calls, "save")
	return nil
}

type spyPublisher struct {
	calls     *[]string
	published [][]auction.Event
}

func (s *spyPublisher) Publish(ctx context.Context, events []auction.Event) error {
	*s.calls = append(*s.calls, "publish")
	s.published = append(s.published, events)
	return nil
}

type spyBidRepo struct {
	calls      *[]string
	saveCount  int
	topCount   int
	lastSaved  auction.Bid
	topBids    []auction.Bid
}

func (s *spyBidRepo) Save(ctx context.Context, auctionID AuctionID, bid auction.Bid) error {
	s.saveCount++
	s.lastSaved = bid
	*s.calls = append(*s.calls, "save_bid")
	return nil
}

func (s *spyBidRepo) TopBids(ctx context.Context, auctionID AuctionID) ([]auction.Bid, error) {
	s.topCount++
	*s.calls = append(*s.calls, "top_bids")
	if len(s.topBids) > 0 {
		return s.topBids, nil
	}
	if (s.lastSaved != auction.Bid{}) {
		return []auction.Bid{s.lastSaved}, nil
	}
	return nil, nil
}

func TestCreateAuctionSavesAggregate(t *testing.T) {
	calls := []string{}
	repo := &spyRepo{calls: &calls}
	publisher := &spyPublisher{calls: &calls}

	uc := NewCreateAuction(repo, publisher)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	if err := uc.Execute(context.Background(), testMeta(), "lot-1", startsAt, endsAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCalls(t, calls, []string{"save"})
	assertCreatedAggregate(t, repo, "lot-1")
	if len(publisher.published) != 0 {
		t.Fatalf("expected no events to be published, got %d", len(publisher.published))
	}
}

func TestPublishAuctionOrchestratesLoadSavePublish(t *testing.T) {
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	repo := &spyRepo{auction: a, calls: &calls}
	publisher := &spyPublisher{calls: &calls}

	uc := NewPublishAuction(repo, publisher)
	if err := uc.Execute(context.Background(), testMeta(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCalls(t, calls, []string{"load", "save", "publish"})
	assertSavedAggregate(t, repo)
	assertPublished(t, publisher)
}

func TestPlaceBidOrchestratesLoadSavePublish(t *testing.T) {
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	publisher := &spyPublisher{calls: &calls}

	uc := NewPlaceBid(repo, bidRepo, publisher)
	placedAt := endsAt.Add(-time.Minute)
	if err := uc.Execute(context.Background(), testMeta(), "1", 100, placedAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCalls(t, calls, []string{"load", "save_bid", "save", "publish"})
	assertSavedAggregate(t, repo)
	assertPublished(t, publisher)
}

func TestCloseAuctionOrchestratesLoadSavePublish(t *testing.T) {
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()
	bid, _ := auction.NewBid("bidder-1", 100, time.Now())
	_, _ = a.PlaceBid(bid)
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls, topBids: []auction.Bid{bid}}
	publisher := &spyPublisher{calls: &calls}

	uc := NewCloseAuction(repo, bidRepo, publisher)
	if err := uc.Execute(context.Background(), testMeta(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCalls(t, calls, []string{"load", "top_bids", "save", "publish"})
	assertSavedAggregate(t, repo)
	assertPublished(t, publisher)
}

func TestCancelAuctionOrchestratesLoadSavePublish(t *testing.T) {
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	publisher := &spyPublisher{calls: &calls}

	uc := NewCancelAuction(repo, publisher)
	if err := uc.Execute(context.Background(), testMeta(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCalls(t, calls, []string{"load", "save", "publish"})
	assertSavedAggregate(t, repo)
	assertPublished(t, publisher)
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected calls %v, got %v", want, got)
	}
}

func assertPublished(t *testing.T, publisher *spyPublisher) {
	t.Helper()
	if len(publisher.published) == 0 {
		t.Fatal("expected events to be published")
	}
	if len(publisher.published[0]) == 0 {
		t.Fatal("expected published events to be non-empty")
	}
}

func assertSavedAggregate(t *testing.T, repo *spyRepo) {
	t.Helper()
	if repo.lastSaved == nil {
		t.Fatal("expected auction to be saved")
	}
	if repo.lastSaved != repo.auction {
		t.Fatal("expected saved aggregate to match loaded aggregate instance")
	}
}

func assertCreatedAggregate(t *testing.T, repo *spyRepo, lotID string) {
	t.Helper()
	if repo.lastSaved == nil {
		t.Fatal("expected auction to be saved")
	}
	if repo.lastSaved == repo.auction {
		t.Fatal("expected created aggregate to be a new instance")
	}
	if repo.lastSaved.ID == "" {
		t.Fatal("expected auction id to be generated")
	}
	if repo.lastSaved.LotID != lotID {
		t.Fatalf("expected lot id %s, got %s", lotID, repo.lastSaved.LotID)
	}
}

func testMeta() CommandMeta {
	return CommandMeta{
		CompanyID:     "company-1",
		UserID:        "user-1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}
