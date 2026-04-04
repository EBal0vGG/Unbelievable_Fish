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

type spyOutbox struct {
	calls     *[]string
	saveCount int
	lastSaved EventEnvelope
}

func (s *spyOutbox) Save(ctx context.Context, envelope EventEnvelope) error {
	s.saveCount++
	s.lastSaved = envelope
	*s.calls = append(*s.calls, "outbox")
	return nil
}

type spyWinners struct {
	calls     *[]string
	saveCount int
	lastSaved []WinnerRecord
}

func (s *spyWinners) Save(ctx context.Context, auctionID AuctionID, winners []WinnerRecord) error {
	s.saveCount++
	s.lastSaved = winners
	*s.calls = append(*s.calls, "winners")
	return nil
}

type spyTx struct {
	repo    *spyRepo
	bids    *spyBidRepo
	outbox  *spyOutbox
	winners *spyWinners
}

func (s *spyTx) Auctions() AuctionRepository { return s.repo }
func (s *spyTx) Bids() BidRepository         { return s.bids }
func (s *spyTx) Outbox() OutboxRepository    { return s.outbox }
func (s *spyTx) Winners() AuctionWinnersRepository { return s.winners }

type spyUOW struct {
	tx *spyTx
}

func (s *spyUOW) Do(ctx context.Context, fn func(Tx) error) error {
	return fn(s.tx)
}

type fakeIDFactory struct {
	id AuctionID
}

func (f fakeIDFactory) NewID() (AuctionID, error) { return f.id, nil }

func TestCreateAuctionSavesAggregate(t *testing.T) {
	logTest(t)
	calls := []string{}
	repo := &spyRepo{calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewCreateAuction(uow, fakeIDFactory{id: "gen-1"})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	logf(t, "create auction lot_id=%s starts_at=%s ends_at=%s", "lot-1", startsAt, endsAt)
	if err := uc.Execute(context.Background(), testMeta(), "lot-1", startsAt, endsAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertCalls(t, calls, []string{"save"})
	assertCreatedAggregate(t, repo, "lot-1")
	if outbox.saveCount != 0 {
		t.Fatalf("expected no outbox save, got %d", outbox.saveCount)
	}
}

func TestPublishAuctionOrchestratesLoadSavePublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewPublishAuction(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logf(t, "calls=%v", calls)
	assertCalls(t, calls, []string{"load", "save", "outbox"})
	assertSavedAggregate(t, repo)
	assertOutbox(t, outbox, testMeta())
}

func TestPlaceBidOrchestratesLoadSavePublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewPlaceBid(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	placedAt := endsAt.Add(-time.Minute)
	logf(t, "place bid amount=%d placed_at=%s", 100, placedAt)
	if err := uc.Execute(context.Background(), testMeta(), "1", 100, placedAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logf(t, "calls=%v bid_saved_amount=%d", calls, bidRepo.lastSaved.Amount())
	assertCalls(t, calls, []string{"load", "save_bid", "save", "outbox"})
	assertSavedAggregate(t, repo)
	assertOutbox(t, outbox, testMeta())
}

func TestCloseAuctionOrchestratesLoadSavePublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	_, _ = a.Publish()
	bid, _ := auction.NewBid("bidder-1", 100, time.Now())
	_, _ = a.PlaceBid(bid)
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls, topBids: []auction.Bid{bid}}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewCloseAuction(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logf(t, "calls=%v", calls)
	assertCalls(t, calls, []string{"load", "top_bids", "winners", "save", "outbox"})
	assertSavedAggregate(t, repo)
	assertOutbox(t, outbox, testMeta())
}

func TestCancelAuctionOrchestratesLoadSavePublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewCancelAuction(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := uc.Execute(context.Background(), testMeta(), "1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logf(t, "calls=%v", calls)
	assertCalls(t, calls, []string{"load", "save", "outbox"})
	assertSavedAggregate(t, repo)
	assertOutbox(t, outbox, testMeta())
}

func assertCalls(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected calls %v, got %v", want, got)
	}
}

func assertOutbox(t *testing.T, outbox *spyOutbox, meta CommandMeta) {
	t.Helper()
	if outbox.saveCount == 0 {
		t.Fatal("expected outbox to be saved")
	}
	if outbox.lastSaved.Meta.CorrelationID != meta.CorrelationID {
		t.Fatalf("expected correlation id %s, got %s", meta.CorrelationID, outbox.lastSaved.Meta.CorrelationID)
	}
	if len(outbox.lastSaved.Events) == 0 {
		t.Fatal("expected outbox events to be non-empty")
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
