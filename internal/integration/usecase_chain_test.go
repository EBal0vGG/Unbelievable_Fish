package integration

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	catalogapp "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
	catalog "github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	dealsapp "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	deal "github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/eventbus/inmemory"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
	tradingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	auction "github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

const (
	eventTypeLotPublished = "catalog.LotPublished"
	eventTypeAuctionWon   = "trading.AuctionWon"
)

func TestUseCaseChainHappyPathAndWinnerFallback(t *testing.T) {
	logTest(t)
	ctx := catalogapp.WithCompanyID(context.Background(), "seller-1")
	now := time.Now()
	startsAt := now.Add(-time.Hour)
	endsAt := now.Add(time.Hour)

	catalogSvc, catalogOutbox, lotID := setupCatalog(ctx, t, startsAt)

	auctionID := "auc-1"
	tradingUOW, tradingOutbox, tradingRepos := setupTrading(auctionID, lotID, startsAt, endsAt)
	dealsRepos := setupDeals()
	bus := inmemory.NewBus()

	t.Run("happy_path_chain", func(t *testing.T) {
		logTest(t)

		if err := catalogSvc.AssignAuctionID(ctx, lotID, auctionID); err != nil {
			t.Fatalf("assign auction id error: %v", err)
		}
		if err := catalogSvc.PublishLot(ctx, lotID); err != nil {
			t.Fatalf("publish lot error: %v", err)
		}

		ucCreateAuction, _ := tradingapp.NewCreateAuction(tradingUOW, fakeAuctionIDFactory{auctionID: tradingapp.AuctionID(auctionID)})
		ucPublish, _ := tradingapp.NewPublishAuction(tradingUOW)
		ucCreateProjection := dealsapp.NewCreateProjection(dealsRepos.projections)
		dealsUOW := dealsapp.NewSimpleUnitOfWork(
			dealsRepos.deals,
			dealsRepos.projections,
			dealsRepos.selections,
			dealsRepos.outbox,
		)
		ucSelection, err := dealsapp.NewCreateDealSelectionFromAuctionWon(dealsUOW)
		if err != nil {
			t.Fatalf("selection constructor error: %v", err)
		}

		bus.Subscribe(eventTypeLotPublished, func(ctx context.Context, envelope events.Envelope) error {
			evt, ok := envelope.Payload.(catalog.LotPublished)
			if !ok {
				return errors.New("unexpected payload for LotPublished")
			}
			if _, err := ucCreateAuction.Execute(ctx, tradingMeta(), evt.LotID, startsAt, endsAt); err != nil {
				return err
			}
			if err := ucPublish.Execute(ctx, tradingMeta(), tradingapp.AuctionID(evt.AuctionID)); err != nil {
				return err
			}
			return ucCreateProjection.Execute(ctx, dealsMeta(), evt.AuctionID, evt.SellerCompanyID, deal.ProductSnapshot{Name: "Fish"}, evt.StartPrice, envelope.OccurredAt)
		})

		bus.Subscribe(eventTypeAuctionWon, func(ctx context.Context, envelope events.Envelope) error {
			evt, ok := envelope.Payload.(auction.AuctionWon)
			if !ok {
				return errors.New("unexpected payload for AuctionWon")
			}
			if len(evt.WinnerCompanyID) == 0 {
				return errors.New("winner list empty")
			}
			if err := ucSelection.Execute(ctx, dealsMeta(), evt.AuctionID, evt.WinnerCompanyID, evt.FinalPrice, envelope.OccurredAt); err != nil {
				return err
			}
			return catalogSvc.HandleAuctionWon(ctx, catalogapp.AuctionWonDTO{
				AuctionID:       evt.AuctionID,
				FinalPrice:      evt.FinalPrice,
				WinnerCompanyID: evt.WinnerCompanyID[0],
			})
		})

		lotPublished := findLotPublished(t, catalogOutbox)
		if err := bus.Publish(context.Background(), envelopeFrom(eventTypeLotPublished, lotPublished, now)); err != nil {
			t.Fatalf("publish LotPublished error: %v", err)
		}

		ucPlaceBid, _ := tradingapp.NewPlaceBid(tradingUOW)
		if err := ucPlaceBid.Execute(context.Background(), tradingMeta(), tradingapp.AuctionID(auctionID), 120, endsAt.Add(-time.Minute)); err != nil {
			t.Fatalf("place bid error: %v", err)
		}
		if err := ucPlaceBid.Execute(context.Background(), tradingMetaWithCompany("buyer-2"), tradingapp.AuctionID(auctionID), 150, endsAt.Add(-time.Minute)); err != nil {
			t.Fatalf("place second bid error: %v", err)
		}

		ucClose, _ := tradingapp.NewCloseAuction(tradingUOW)
		if err := ucClose.Execute(context.Background(), tradingMeta(), tradingapp.AuctionID(auctionID)); err != nil {
			t.Fatalf("close auction error: %v", err)
		}

		auctionWon := findAuctionWon(t, tradingOutbox)
		if len(auctionWon.WinnerCompanyID) == 0 {
			t.Fatal("expected winner list")
		}

		if err := bus.Publish(context.Background(), envelopeFrom(eventTypeAuctionWon, auctionWon, now)); err != nil {
			t.Fatalf("publish AuctionWon error: %v", err)
		}

		if dealsRepos.deals.lastSaved == nil {
			t.Fatal("expected deal to be saved")
		}
		if dealsRepos.deals.lastSaved.CustomerID() != "buyer-2" {
			t.Fatalf("expected first winner deal for buyer-2, got %s", dealsRepos.deals.lastSaved.CustomerID())
		}
		if dealsRepos.selections.lastSaved == nil {
			t.Fatal("expected selection to be saved")
		}
		if tradingRepos.auctions[tradingapp.AuctionID(auctionID)] == nil {
			t.Fatal("expected auction to exist")
		}
		if catalogOutbox.countEvent(func(ev catalog.Event) bool {
			_, ok := ev.(catalog.LotPublished)
			return ok
		}) == 0 {
			t.Fatal("expected LotPublished event")
		}
		if tradingOutbox.countEvent(func(ev auction.Event) bool {
			_, ok := ev.(auction.AuctionPublished)
			return ok
		}) == 0 {
			t.Fatal("expected AuctionPublished event")
		}
		if tradingOutbox.countEvent(func(ev auction.Event) bool {
			_, ok := ev.(auction.BidPlaced)
			return ok
		}) == 0 {
			t.Fatal("expected BidPlaced event")
		}
		if tradingOutbox.countEvent(func(ev auction.Event) bool {
			_, ok := ev.(auction.AuctionWon)
			return ok
		}) == 0 {
			t.Fatal("expected AuctionWon event")
		}
	})

	t.Run("winner_declined_moves_to_next", func(t *testing.T) {
		logTest(t)

		dealsUOW := dealsapp.NewSimpleUnitOfWork(
			dealsRepos.deals,
			dealsRepos.projections,
			dealsRepos.selections,
			dealsRepos.outbox,
		)
		ucDeclined, err := dealsapp.NewHandleDealDeclined(dealsUOW)
		if err != nil {
			t.Fatalf("declined constructor error: %v", err)
		}
		if err := ucDeclined.Execute(context.Background(), dealsMeta(), auctionID, ""); err != nil {
			t.Fatalf("declined error: %v", err)
		}
		if dealsRepos.deals.lastSaved == nil || dealsRepos.deals.lastSaved.CustomerID() != "buyer-1" {
			t.Fatalf("expected next winner deal, got %v", dealsRepos.deals.lastSaved)
		}
	})
}

func setupCatalog(ctx context.Context, t *testing.T, startsAt time.Time) (*catalogapp.CatalogService, *catalogOutbox, string) {
	t.Helper()
	fishRepo := newMemoryFishRepo()
	productRepo := newMemoryProductRepo()
	lotRepo := newMemoryLotRepo()
	unitRepo := newMemoryUnitRepo()
	processingRepo := newMemoryProcessingTypeRepo()
	outbox := &catalogOutbox{}
	tx := noopTx{}

	fish, _ := catalog.NewFish("fish-1", "Cod", "desc")
	_ = fishRepo.Save(ctx, fish)
	unitRepo.data["kg"] = struct{}{}
	processingRepo.data["frozen"] = struct{}{}

	service := catalogapp.NewCatalogService(
		fishRepo,
		unitRepo,
		processingRepo,
		productRepo,
		lotRepo,
		outbox,
		stubIDGenerator{fishID: "fish-1", productID: "prod-1", lotID: "lot-1"},
		tx,
	)

	_, _, err := service.CreateProduct(ctx, catalogapp.CreateProductCommand{
		FishID:         "fish-1",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != nil {
		t.Fatalf("create product error: %v", err)
	}
	if err := service.PublishProduct(ctx, "prod-1"); err != nil {
		t.Fatalf("publish product error: %v", err)
	}

	lotID, _, err := service.CreateLot(ctx, catalogapp.CreateLotCommand{
		ProductID:              "prod-1",
		Photo:                  "photo",
		Quantity:               10,
		StartPrice:             100,
		AuctionStartsAt:        startsAt,
		AuctionDurationMinutes: 60,
	})
	if err != nil {
		t.Fatalf("create lot error: %v", err)
	}

	return service, outbox, lotID
}

func setupTrading(auctionID, lotID string, startsAt, endsAt time.Time) (*memoryUOW, *tradingOutbox, *tradingRepoState) {
	auctions := map[tradingapp.AuctionID]*auction.Auction{}
	bids := map[tradingapp.AuctionID][]auction.Bid{}
	outbox := &tradingOutbox{}
	winners := &tradingWinners{}
	tx := &memoryTx{
		auctions: auctions,
		bids:     bids,
		outbox:   outbox,
		winners:  winners,
	}
	return &memoryUOW{tx: tx}, outbox, &tradingRepoState{auctions: auctions}
}

func setupDeals() *dealsRepoState {
	return &dealsRepoState{
		deals:       &dealRepoMemory{data: make(map[string]*deal.Deal)},
		projections: &projectionRepoMemory{data: make(map[string]*deal.DealProjection)},
		selections:  &selectionRepoMemory{data: make(map[string]*deal.WinnerSelection)},
		outbox:      &dealOutbox{},
	}
}

func findAuctionWon(t *testing.T, outbox *tradingOutbox) auction.AuctionWon {
	t.Helper()
	for i := len(outbox.events) - 1; i >= 0; i-- {
		if won, ok := outbox.events[i].(auction.AuctionWon); ok {
			return won
		}
	}
	t.Fatal("AuctionWon not found")
	return auction.AuctionWon{}
}

func findLotPublished(t *testing.T, outbox *catalogOutbox) catalog.LotPublished {
	t.Helper()
	for i := len(outbox.events) - 1; i >= 0; i-- {
		if evt, ok := outbox.events[i].(catalog.LotPublished); ok {
			return evt
		}
	}
	t.Fatal("LotPublished not found")
	return catalog.LotPublished{}
}

func envelopeFrom(eventType string, payload any, occurredAt time.Time) events.Envelope {
	return events.Envelope{
		Type:       eventType,
		Payload:    payload,
		OccurredAt: occurredAt,
	}
}

type stubIDGenerator struct {
	fishID    string
	productID string
	lotID     string
}

func (g stubIDGenerator) NewFishID() string    { return g.fishID }
func (g stubIDGenerator) NewProductID() string { return g.productID }
func (g stubIDGenerator) NewLotID() string     { return g.lotID }

type fakeAuctionIDFactory struct {
	auctionID tradingapp.AuctionID
}

func (f fakeAuctionIDFactory) NewID() (tradingapp.AuctionID, error) { return f.auctionID, nil }

type catalogOutbox struct {
	events []catalog.Event
}

func (o *catalogOutbox) Add(ctx context.Context, events []catalog.Event) error {
	_ = ctx
	o.events = append(o.events, events...)
	return nil
}

func (o *catalogOutbox) countEvent(match func(catalog.Event) bool) int {
	count := 0
	for _, ev := range o.events {
		if match(ev) {
			count++
		}
	}
	return count
}

type memoryFishRepo struct {
	data map[string]*catalog.Fish
}

func newMemoryFishRepo() *memoryFishRepo {
	return &memoryFishRepo{data: make(map[string]*catalog.Fish)}
}

func (r *memoryFishRepo) Get(ctx context.Context, fishID string) (*catalog.Fish, error) {
	_ = ctx
	return r.data[fishID], nil
}

func (r *memoryFishRepo) Exists(ctx context.Context, fishID string) (bool, error) {
	_ = ctx
	_, ok := r.data[fishID]
	return ok, nil
}

func (r *memoryFishRepo) Save(ctx context.Context, fish *catalog.Fish) error {
	_ = ctx
	r.data[fish.ID()] = fish
	return nil
}

type memoryProductRepo struct {
	data map[string]*catalog.Product
}

func newMemoryProductRepo() *memoryProductRepo {
	return &memoryProductRepo{data: make(map[string]*catalog.Product)}
}

func (r *memoryProductRepo) Get(ctx context.Context, productID string) (*catalog.Product, error) {
	_ = ctx
	return r.data[productID], nil
}

func (r *memoryProductRepo) Save(ctx context.Context, product *catalog.Product) error {
	_ = ctx
	r.data[product.ID()] = product
	return nil
}

type memoryLotRepo struct {
	data map[string]*catalog.Lot
}

func newMemoryLotRepo() *memoryLotRepo {
	return &memoryLotRepo{data: make(map[string]*catalog.Lot)}
}

func (r *memoryLotRepo) Get(ctx context.Context, lotID string) (*catalog.Lot, error) {
	_ = ctx
	return r.data[lotID], nil
}

func (r *memoryLotRepo) GetByAuctionID(ctx context.Context, auctionID string) (*catalog.Lot, error) {
	_ = ctx
	for _, lot := range r.data {
		if lot.AuctionID() == auctionID {
			return lot, nil
		}
	}
	return nil, catalogapp.ErrNotFound
}

func (r *memoryLotRepo) Save(ctx context.Context, lot *catalog.Lot) error {
	_ = ctx
	r.data[lot.ID()] = lot
	return nil
}

type memoryUnitRepo struct {
	data map[string]struct{}
}

func newMemoryUnitRepo() *memoryUnitRepo {
	return &memoryUnitRepo{data: make(map[string]struct{})}
}

func (r *memoryUnitRepo) Exists(ctx context.Context, unit string) (bool, error) {
	_ = ctx
	_, ok := r.data[unit]
	return ok, nil
}

type memoryProcessingTypeRepo struct {
	data map[string]struct{}
}

func newMemoryProcessingTypeRepo() *memoryProcessingTypeRepo {
	return &memoryProcessingTypeRepo{data: make(map[string]struct{})}
}

func (r *memoryProcessingTypeRepo) Exists(ctx context.Context, processingType string) (bool, error) {
	_ = ctx
	_, ok := r.data[processingType]
	return ok, nil
}

type noopTx struct{}

func (noopTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type tradingOutbox struct {
	events []auction.Event
}

func (o *tradingOutbox) Add(ctx context.Context, events []auction.Event) error {
	_ = ctx
	o.events = append(o.events, events...)
	return nil
}

func (o *tradingOutbox) countEvent(match func(auction.Event) bool) int {
	count := 0
	for _, ev := range o.events {
		if match(ev) {
			count++
		}
	}
	return count
}

type tradingWinners struct {
	lastSaved []tradingapp.WinnerRecord
}

func (w *tradingWinners) Save(ctx context.Context, auctionID tradingapp.AuctionID, winners []tradingapp.WinnerRecord) error {
	_ = ctx
	_ = auctionID
	w.lastSaved = winners
	return nil
}

type memoryAuctionRepo struct {
	data map[tradingapp.AuctionID]*auction.Auction
}

func (r *memoryAuctionRepo) Load(ctx context.Context, id tradingapp.AuctionID) (*auction.Auction, error) {
	_ = ctx
	item, ok := r.data[id]
	if !ok {
		return nil, tradingapp.ErrNotFound
	}
	return item, nil
}

func (r *memoryAuctionRepo) LoadForUpdate(ctx context.Context, id tradingapp.AuctionID) (*auction.Auction, error) {
	return r.Load(ctx, id)
}

func (r *memoryAuctionRepo) Save(ctx context.Context, a *auction.Auction) error {
	_ = ctx
	r.data[tradingapp.AuctionID(a.ID)] = a
	return nil
}

type memoryBidRepo struct {
	data map[tradingapp.AuctionID][]auction.Bid
}

func (r *memoryBidRepo) Save(ctx context.Context, auctionID tradingapp.AuctionID, bid auction.Bid) error {
	_ = ctx
	r.data[auctionID] = append(r.data[auctionID], bid)
	return nil
}

func (r *memoryBidRepo) TopBids(ctx context.Context, auctionID tradingapp.AuctionID) ([]auction.Bid, error) {
	_ = ctx
	bids := append([]auction.Bid(nil), r.data[auctionID]...)
	sort.Slice(bids, func(i, j int) bool {
		if bids[i].Amount() != bids[j].Amount() {
			return bids[i].Amount() > bids[j].Amount()
		}
		return bids[i].PlacedAt().Before(bids[j].PlacedAt())
	})
	return bids, nil
}

type memoryTx struct {
	auctions map[tradingapp.AuctionID]*auction.Auction
	bids     map[tradingapp.AuctionID][]auction.Bid
	outbox   *tradingOutbox
	winners  *tradingWinners
}

func (t *memoryTx) Auctions() tradingapp.AuctionRepository {
	return &memoryAuctionRepo{data: t.auctions}
}

func (t *memoryTx) Bids() tradingapp.BidRepository {
	return &memoryBidRepo{data: t.bids}
}

func (t *memoryTx) Outbox() tradingapp.OutboxRepository {
	return t.outbox
}

func (t *memoryTx) Winners() tradingapp.AuctionWinnersRepository {
	return t.winners
}

type memoryUOW struct {
	tx *memoryTx
}

func (u *memoryUOW) Do(ctx context.Context, fn func(tradingapp.Tx) error) error {
	return fn(u.tx)
}

type tradingRepoState struct {
	auctions map[tradingapp.AuctionID]*auction.Auction
}

type dealRepoMemory struct {
	data      map[string]*deal.Deal
	lastSaved *deal.Deal
}

func (r *dealRepoMemory) Save(ctx context.Context, item *deal.Deal) error {
	_ = ctx
	r.data[item.ID()] = item
	r.lastSaved = item
	return nil
}

func (r *dealRepoMemory) GetByID(ctx context.Context, dealID string) (*deal.Deal, error) {
	_ = ctx
	item, ok := r.data[dealID]
	if !ok {
		return nil, dealsapp.ErrDealNotFound
	}
	return item, nil
}

func (r *dealRepoMemory) GetByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error) {
	_ = ctx
	for _, item := range r.data {
		if item.AuctionID() == auctionID {
			return item, nil
		}
	}
	return nil, dealsapp.ErrDealNotFound
}

type projectionRepoMemory struct {
	data map[string]*deal.DealProjection
}

func (r *projectionRepoMemory) Save(ctx context.Context, item *deal.DealProjection) error {
	_ = ctx
	r.data[item.AuctionID] = item
	return nil
}

func (r *projectionRepoMemory) GetByAuctionID(ctx context.Context, auctionID string) (*deal.DealProjection, error) {
	_ = ctx
	item, ok := r.data[auctionID]
	if !ok {
		return nil, deal.ErrProjectionNotFound
	}
	return item, nil
}

type selectionRepoMemory struct {
	data      map[string]*deal.WinnerSelection
	lastSaved *deal.WinnerSelection
}

func (r *selectionRepoMemory) Save(ctx context.Context, item *deal.WinnerSelection) error {
	_ = ctx
	r.data[item.AuctionID] = item
	r.lastSaved = item
	return nil
}

func (r *selectionRepoMemory) GetByAuctionID(ctx context.Context, auctionID string) (*deal.WinnerSelection, error) {
	_ = ctx
	item, ok := r.data[auctionID]
	if !ok {
		return nil, deal.ErrSelectionNotFound
	}
	return item, nil
}

type dealOutbox struct {
	events [][]deal.Event
}

func (o *dealOutbox) Add(ctx context.Context, events []deal.Event) error {
	_ = ctx
	if len(events) > 0 {
		o.events = append(o.events, events)
	}
	return nil
}

type dealsRepoState struct {
	deals       *dealRepoMemory
	projections *projectionRepoMemory
	selections  *selectionRepoMemory
	outbox      *dealOutbox
}

func dealsMeta() dealsapp.CommandMeta {
	return dealsapp.CommandMeta{
		CompanyID:     "buyer-1",
		UserID:        "buyer-1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

func tradingMeta() tradingapp.CommandMeta {
	return tradingapp.CommandMeta{
		CompanyID:     "buyer-1",
		UserID:        "buyer-1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}

func tradingMetaWithCompany(companyID string) tradingapp.CommandMeta {
	return tradingapp.CommandMeta{
		CompanyID:     companyID,
		UserID:        companyID,
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
}
