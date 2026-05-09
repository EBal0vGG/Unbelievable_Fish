package app

import (
	"context"
	"errors"
	"reflect"
	"sync"
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
	if s.auction == nil {
		return nil, ErrNotFound
	}
	return s.auction, nil
}

func (s *spyRepo) LoadForUpdate(ctx context.Context, id AuctionID) (*auction.Auction, error) {
	s.loadCount++
	*s.calls = append(*s.calls, "load_for_update")
	if s.auction == nil {
		return nil, ErrNotFound
	}
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
	calls     *[]string
	saveCount int
	topCount  int
	lastSaved auction.Bid
	topBids   []auction.Bid
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
	calls      *[]string
	saveCount  int
	lastEvents []auction.Event
	lastMeta   CommandMeta
}

func (s *spyOutbox) Add(ctx context.Context, events []auction.Event) error {
	s.saveCount++
	s.lastEvents = events
	if meta, ok := CommandMetaFromContext(ctx); ok {
		s.lastMeta = meta
	}
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

type spyDepositService struct {
	calls *[]string
	err   error
}

func (s *spyDepositService) ReserveAuctionDeposit(ctx context.Context, companyID, auctionID string, startPrice int64) error {
	_ = ctx
	_ = companyID
	_ = auctionID
	_ = startPrice
	if s.calls != nil {
		*s.calls = append(*s.calls, "reserve_deposit")
	}
	return s.err
}

type spyTx struct {
	repo    *spyRepo
	bids    *spyBidRepo
	outbox  *spyOutbox
	winners *spyWinners
}

func (s *spyTx) Auctions() AuctionRepository       { return s.repo }
func (s *spyTx) Bids() BidRepository               { return s.bids }
func (s *spyTx) Outbox() OutboxRepository          { return s.outbox }
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
	auctionID, err := uc.Execute(context.Background(), testMeta(), "lot-1", startsAt, endsAt, 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auctionID != "gen-1" {
		t.Fatalf("expected auction id gen-1, got %s", auctionID)
	}
	assertCalls(t, calls, []string{"load", "save"})
	assertCreatedAggregate(t, repo, "lot-1")
	if outbox.saveCount != 0 {
		t.Fatalf("expected no outbox save, got %d", outbox.saveCount)
	}
}

func TestCreateAuctionIsIdempotentWhenAuctionExists(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	existing, err := auction.NewAuction("gen-1", "lot-1", startsAt, endsAt)
	if err != nil {
		t.Fatalf("unexpected auction constructor error: %v", err)
	}
	repo := &spyRepo{calls: &calls, auction: existing}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewCreateAuction(uow, fakeIDFactory{id: "gen-1"})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	auctionID, err := uc.Execute(context.Background(), testMeta(), "lot-1", startsAt, endsAt, 100, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auctionID != "gen-1" {
		t.Fatalf("expected auction id gen-1, got %s", auctionID)
	}
	assertCalls(t, calls, []string{"load"})
	if repo.saveCount != 0 {
		t.Fatalf("expected no save, got %d", repo.saveCount)
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
	assertCalls(t, calls, []string{"load_for_update", "save", "outbox"})
	assertSavedAggregate(t, repo)
	assertOutbox(t, outbox, testMeta())
}

func TestPlaceBidOrchestratesLoadSavePublish(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, err := auction.NewAuctionWithPricing("1", "lot-1", startsAt, endsAt, 100, 10)
	if err != nil {
		t.Fatalf("auction: %v", err)
	}
	logf(t, "auction id=%s lot_id=%s state=%s", a.ID, a.LotID, a.State())
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	deposits := &spyDepositService{calls: &calls}

	uc, err := NewPlaceBid(uow, deposits)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	placedAt := endsAt.Add(-time.Minute)
	logf(t, "place bid amount=%d placed_at=%s", 100, placedAt)
	if err := uc.Execute(context.Background(), testMeta(), "1", 100, placedAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	logf(t, "calls=%v bid_saved_amount=%d", calls, bidRepo.lastSaved.Amount())
	assertCalls(t, calls, []string{"load_for_update", "reserve_deposit", "save_bid", "save", "outbox"})
	assertSavedAggregate(t, repo)
	assertOutbox(t, outbox, testMeta())
}

func TestPlaceBidSkipsPersistWhenDepositFails(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, err := auction.NewAuctionWithPricing("1", "lot-1", startsAt, endsAt, 100, 10)
	if err != nil {
		t.Fatalf("auction: %v", err)
	}
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}
	deposits := &spyDepositService{calls: &calls, err: ErrInsufficientFundsForDeposit}

	uc, err := NewPlaceBid(uow, deposits)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	placedAt := endsAt.Add(-time.Minute)
	err = uc.Execute(context.Background(), testMeta(), "1", 100, placedAt)
	if !errors.Is(err, ErrInsufficientFundsForDeposit) {
		t.Fatalf("expected ErrInsufficientFundsForDeposit, got %v", err)
	}
	assertCalls(t, calls, []string{"load_for_update", "reserve_deposit"})
	if bidRepo.saveCount != 0 {
		t.Fatalf("expected no bid save, got %d", bidRepo.saveCount)
	}
	if repo.saveCount != 0 {
		t.Fatalf("expected no auction save, got %d", repo.saveCount)
	}
	if outbox.saveCount != 0 {
		t.Fatalf("expected no outbox, got %d", outbox.saveCount)
	}
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
	assertCalls(t, calls, []string{"load_for_update", "top_bids", "winners", "save", "outbox"})
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
	assertCalls(t, calls, []string{"load_for_update", "save", "outbox"})
	assertSavedAggregate(t, repo)
	assertOutbox(t, outbox, testMeta())
}

func TestPlaceBidRejectsLowerAmount(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewPlaceBid(uow, NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	placedAt := endsAt.Add(-time.Minute)
	if err := uc.Execute(context.Background(), testMeta(), "1", 100, placedAt); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = uc.Execute(context.Background(), testMeta(), "1", 50, placedAt)
	if err != auction.ErrBidStepTooSmall {
		t.Fatalf("expected ErrBidStepTooSmall, got %v", err)
	}
}

func TestPlaceBidRejectsAfterEnd(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewPlaceBid(uow, NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	placedAt := endsAt.Add(time.Minute)
	err = uc.Execute(context.Background(), testMeta(), "1", 100, placedAt)
	if err != auction.ErrAuctionAlreadyEnded {
		t.Fatalf("expected ErrAuctionAlreadyEnded, got %v", err)
	}
}

func TestConcurrentBidsKeepConsistentAuctionState(t *testing.T) {
	logTest(t)
	startsAt := time.Now().Add(-time.Hour)
	endsAt := time.Now().Add(time.Hour)
	a, err := auction.NewAuctionWithPricing("race-1", "lot-race", startsAt, endsAt, 100, 10)
	if err != nil {
		t.Fatalf("unexpected auction constructor error: %v", err)
	}
	_, _ = a.Publish()

	repo := &raceAuctionRepo{auction: a}
	bidRepo := &syncBidRepo{}
	outbox := &syncOutbox{}
	winners := &syncWinners{}
	uow := &syncUOW{tx: &syncTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewPlaceBid(uow, NoopDepositService{})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	placedAt := endsAt.Add(-time.Minute)
	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error, 2)

	go func() {
		defer wg.Done()
		errCh <- uc.Execute(context.Background(), CommandMeta{CompanyID: "buyer-a", UserID: "u-a"}, "race-1", 110, placedAt)
	}()
	go func() {
		defer wg.Done()
		errCh <- uc.Execute(context.Background(), CommandMeta{CompanyID: "buyer-b", UserID: "u-b"}, "race-1", 120, placedAt)
	}()
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil && err != auction.ErrBidStepTooSmall {
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}

	if got := repo.auction.CurrentPrice(); got != 120 {
		t.Fatalf("expected final current price 120, got %d", got)
	}
	if got := repo.auction.LeaderCompanyID(); got != "buyer-b" {
		t.Fatalf("expected final leader buyer-b, got %s", got)
	}
}

func TestPublishAuctionRejectsAlreadyPublished(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
	_, _ = a.Publish()
	repo := &spyRepo{auction: a, calls: &calls}
	bidRepo := &spyBidRepo{calls: &calls}
	outbox := &spyOutbox{calls: &calls}
	winners := &spyWinners{calls: &calls}
	uow := &spyUOW{tx: &spyTx{repo: repo, bids: bidRepo, outbox: outbox, winners: winners}}

	uc, err := NewPublishAuction(uow)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	err = uc.Execute(context.Background(), testMeta(), "1")
	if err != auction.ErrAuctionCannotBePublished {
		t.Fatalf("expected ErrAuctionCannotBePublished, got %v", err)
	}
}

func TestCloseAuctionRejectsAlreadyClosed(t *testing.T) {
	logTest(t)
	calls := []string{}
	startsAt := time.Now().Add(-time.Hour)
	endsAt := startsAt.Add(time.Hour)
	a, _ := auction.NewAuction("1", "lot-1", startsAt, endsAt)
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
	err = uc.Execute(context.Background(), testMeta(), "1")
	if err != auction.ErrCannotCloseAuction {
		t.Fatalf("expected ErrCannotCloseAuction, got %v", err)
	}
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
	if outbox.lastMeta.CorrelationID != meta.CorrelationID {
		t.Fatalf("expected correlation id %s, got %s", meta.CorrelationID, outbox.lastMeta.CorrelationID)
	}
	if len(outbox.lastEvents) == 0 {
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

type raceAuctionRepo struct {
	mu      sync.Mutex
	auction *auction.Auction
}

func (r *raceAuctionRepo) Load(context.Context, AuctionID) (*auction.Auction, error) {
	if r.auction == nil {
		return nil, ErrNotFound
	}
	return r.auction, nil
}

func (r *raceAuctionRepo) LoadForUpdate(context.Context, AuctionID) (*auction.Auction, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.auction == nil {
		return nil, ErrNotFound
	}
	return r.auction, nil
}

func (r *raceAuctionRepo) Save(context.Context, *auction.Auction) error {
	return nil
}

type syncBidRepo struct {
	mu   sync.Mutex
	bids []auction.Bid
}

func (r *syncBidRepo) Save(_ context.Context, _ AuctionID, bid auction.Bid) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bids = append(r.bids, bid)
	return nil
}

func (r *syncBidRepo) TopBids(context.Context, AuctionID) ([]auction.Bid, error) {
	return nil, nil
}

type syncOutbox struct{}

func (syncOutbox) Add(context.Context, []auction.Event) error { return nil }

type syncWinners struct{}

func (syncWinners) Save(context.Context, AuctionID, []WinnerRecord) error { return nil }

type syncTx struct {
	repo    *raceAuctionRepo
	bids    *syncBidRepo
	outbox  *syncOutbox
	winners *syncWinners
}

func (s *syncTx) Auctions() AuctionRepository       { return s.repo }
func (s *syncTx) Bids() BidRepository               { return s.bids }
func (s *syncTx) Outbox() OutboxRepository          { return s.outbox }
func (s *syncTx) Winners() AuctionWinnersRepository { return s.winners }

type syncUOW struct {
	mu sync.Mutex
	tx *syncTx
}

func (s *syncUOW) Do(ctx context.Context, fn func(Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(s.tx)
}
