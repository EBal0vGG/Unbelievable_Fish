package app

import (
	"context"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/domain/catalog"
)

type memoryFishRepo struct {
	data map[string]*catalog.Fish
}

type memoryProductRepo struct {
	data map[string]*catalog.Product
}

type memoryLotRepo struct {
	data map[string]*catalog.Lot
}

type memoryUnitRepo struct {
	data map[string]struct{}
}

type memoryProcessingTypeRepo struct {
	data map[string]struct{}
}

type memoryOutbox struct {
	events []catalog.Event
}

type noopTx struct{}

func (noopTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type stubIDGenerator struct {
	fishID    string
	productID string
	lotID     string
}

func (g stubIDGenerator) NewFishID() string    { return g.fishID }
func (g stubIDGenerator) NewProductID() string { return g.productID }
func (g stubIDGenerator) NewLotID() string     { return g.lotID }

func newMemoryFishRepo() *memoryFishRepo {
	return &memoryFishRepo{data: make(map[string]*catalog.Fish)}
}

func newMemoryProductRepo() *memoryProductRepo {
	return &memoryProductRepo{data: make(map[string]*catalog.Product)}
}

func newMemoryLotRepo() *memoryLotRepo {
	return &memoryLotRepo{data: make(map[string]*catalog.Lot)}
}

func newMemoryUnitRepo() *memoryUnitRepo {
	return &memoryUnitRepo{data: make(map[string]struct{})}
}

func newMemoryProcessingTypeRepo() *memoryProcessingTypeRepo {
	return &memoryProcessingTypeRepo{data: make(map[string]struct{})}
}

func (r *memoryFishRepo) Get(ctx context.Context, fishID string) (*catalog.Fish, error) {
	fish, ok := r.data[fishID]
	if !ok {
		return nil, ErrNotFound
	}
	return fish, nil
}

func (r *memoryFishRepo) Exists(ctx context.Context, fishID string) (bool, error) {
	_, ok := r.data[fishID]
	return ok, nil
}

func (r *memoryFishRepo) Save(ctx context.Context, fish *catalog.Fish) error {
	r.data[fish.ID()] = fish
	return nil
}

func (r *memoryProductRepo) Get(ctx context.Context, productID string) (*catalog.Product, error) {
	product, ok := r.data[productID]
	if !ok {
		return nil, ErrNotFound
	}
	return product, nil
}

func (r *memoryProductRepo) Save(ctx context.Context, product *catalog.Product) error {
	r.data[product.ID()] = product
	return nil
}

func (r *memoryLotRepo) Get(ctx context.Context, lotID string) (*catalog.Lot, error) {
	lot, ok := r.data[lotID]
	if !ok {
		return nil, ErrNotFound
	}
	return lot, nil
}

func (r *memoryLotRepo) GetByAuctionID(ctx context.Context, auctionID string) (*catalog.Lot, error) {
	for _, lot := range r.data {
		if lot.AuctionID() == auctionID {
			return lot, nil
		}
	}
	return nil, ErrNotFound
}

func (r *memoryLotRepo) Save(ctx context.Context, lot *catalog.Lot) error {
	r.data[lot.ID()] = lot
	return nil
}

func (r *memoryUnitRepo) Exists(ctx context.Context, unit string) (bool, error) {
	_, ok := r.data[unit]
	return ok, nil
}

func (r *memoryUnitRepo) Add(unit string) {
	r.data[unit] = struct{}{}
}

func (r *memoryProcessingTypeRepo) Exists(ctx context.Context, processingType string) (bool, error) {
	_, ok := r.data[processingType]
	return ok, nil
}

func (r *memoryProcessingTypeRepo) Add(processingType string) {
	r.data[processingType] = struct{}{}
}

func (o *memoryOutbox) Add(ctx context.Context, events []catalog.Event) error {
	if len(events) == 0 {
		return nil
	}
	o.events = append(o.events, events...)
	return nil
}

func (o *memoryOutbox) Count() int {
	return len(o.events)
}

func (o *memoryOutbox) Last() catalog.Event {
	if len(o.events) == 0 {
		return nil
	}
	return o.events[len(o.events)-1]
}

type testDeps struct {
	svc                *CatalogService
	fishRepo           *memoryFishRepo
	unitRepo           *memoryUnitRepo
	processingTypeRepo *memoryProcessingTypeRepo
	productRepo        *memoryProductRepo
	lotRepo            *memoryLotRepo
	outbox             *memoryOutbox
}

func newTestDeps() *testDeps {
	fishRepo := newMemoryFishRepo()
	unitRepo := newMemoryUnitRepo()
	processingTypeRepo := newMemoryProcessingTypeRepo()
	productRepo := newMemoryProductRepo()
	lotRepo := newMemoryLotRepo()
	outbox := &memoryOutbox{}
	idGenerator := stubIDGenerator{
		fishID:    "fish-generated",
		productID: "prod-1",
		lotID:     "lot-generated",
	}
	svc := NewCatalogService(fishRepo, unitRepo, processingTypeRepo, productRepo, lotRepo, outbox, idGenerator, noopTx{})
	return &testDeps{
		svc:                svc,
		fishRepo:           fishRepo,
		unitRepo:           unitRepo,
		processingTypeRepo: processingTypeRepo,
		productRepo:        productRepo,
		lotRepo:            lotRepo,
		outbox:             outbox,
	}
}

func seedRefs(deps *testDeps, unit, processingType string) {
	deps.unitRepo.Add(unit)
	deps.processingTypeRepo.Add(processingType)
}

func newProductSnapshot() catalog.ProductSnapshot {
	return catalog.ProductSnapshot{
		FishID:         "fish-1",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	}
}

func TestCreateFishGeneratesID(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fishID, err := deps.svc.CreateFish(ctx, CreateFishCommand{
		Name:        "Cod",
		Description: "desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fishID != "fish-generated" {
		t.Fatalf("expected generated fish id, got %s", fishID)
	}

	stored, err := deps.fishRepo.Get(ctx, fishID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.ID() != fishID {
		t.Fatalf("expected stored fish id to match, got %s", stored.ID())
	}
}

func TestCreateProductRequiresFish(t *testing.T) {
	deps := newTestDeps()
	seedRefs(deps, "kg", "frozen")

	_, _, err := deps.svc.CreateProduct(context.Background(), CreateProductCommand{
		FishID:         "fish-1",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != ErrFishNotFound {
		t.Fatalf("expected ErrFishNotFound, got %v", err)
	}
	if deps.outbox.Count() != 0 {
		t.Fatalf("expected no outbox events, got %d", deps.outbox.Count())
	}
}

func TestCreateProductRequiresUnit(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-1", "Cod", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deps.processingTypeRepo.Add("frozen")

	_, _, err = deps.svc.CreateProduct(ctx, CreateProductCommand{
		FishID:         "fish-1",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != ErrUnitNotFound {
		t.Fatalf("expected ErrUnitNotFound, got %v", err)
	}
}

func TestCreateProductRequiresProcessingType(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-1", "Cod", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deps.unitRepo.Add("kg")

	_, _, err = deps.svc.CreateProduct(ctx, CreateProductCommand{
		FishID:         "fish-1",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != ErrProcessingTypeNotFound {
		t.Fatalf("expected ErrProcessingTypeNotFound, got %v", err)
	}
}

func TestCreateProductWritesOutbox(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-1", "Cod", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seedRefs(deps, "kg", "frozen")

	productID, events, err := deps.svc.CreateProduct(ctx, CreateProductCommand{
		FishID:         "fish-1",
		Weight:         10,
		Unit:           "kg",
		Size:           "M",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if productID != "prod-1" {
		t.Fatalf("expected product id to match, got %s", productID)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].(catalog.ProductCreated); !ok {
		t.Fatalf("expected ProductCreated event")
	}
	if deps.outbox.Count() != 1 {
		t.Fatalf("expected 1 outbox event, got %d", deps.outbox.Count())
	}
}

func TestCreateLotGeneratesID(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	product, _, err := catalog.NewProduct("prod-existing", "fish-1", 10, "kg", "M", catalog.ProcessingType("frozen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.productRepo.Save(ctx, product); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lotID, events, err := deps.svc.CreateLot(ctx, CreateLotCommand{
		ProductID:       "prod-existing",
		SellerCompanyID: "seller-1",
		Photo:           "",
		Quantity:        10,
		StartPrice:      100,
		AuctionStartsAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lotID != "lot-generated" {
		t.Fatalf("expected generated lot id, got %s", lotID)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if _, ok := events[0].(catalog.LotCreated); !ok {
		t.Fatalf("expected LotCreated event")
	}

	stored, err := deps.lotRepo.Get(ctx, lotID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.ID() != lotID {
		t.Fatalf("expected stored lot id to match, got %s", stored.ID())
	}
}

func TestUpdateProductRequiresFish(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	product, _, err := catalog.NewProduct("prod-1", "fish-1", 10, "kg", "M", catalog.ProcessingType("frozen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.productRepo.Save(ctx, product); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seedRefs(deps, "kg", "frozen")

	err = deps.svc.UpdateProduct(ctx, UpdateProductCommand{
		ProductID:      "prod-1",
		FishID:         "fish-1",
		Weight:         12,
		Unit:           "kg",
		Size:           "L",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != ErrFishNotFound {
		t.Fatalf("expected ErrFishNotFound, got %v", err)
	}
}

func TestUpdateProductRequiresUnit(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-1", "Cod", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deps.processingTypeRepo.Add("frozen")

	product, _, err := catalog.NewProduct("prod-1", "fish-1", 10, "kg", "M", catalog.ProcessingType("frozen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.productRepo.Save(ctx, product); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = deps.svc.UpdateProduct(ctx, UpdateProductCommand{
		ProductID:      "prod-1",
		FishID:         "fish-1",
		Weight:         12,
		Unit:           "kg",
		Size:           "L",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != ErrUnitNotFound {
		t.Fatalf("expected ErrUnitNotFound, got %v", err)
	}
}

func TestUpdateProductRequiresProcessingType(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-1", "Cod", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	deps.unitRepo.Add("kg")

	product, _, err := catalog.NewProduct("prod-1", "fish-1", 10, "kg", "M", catalog.ProcessingType("frozen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.productRepo.Save(ctx, product); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = deps.svc.UpdateProduct(ctx, UpdateProductCommand{
		ProductID:      "prod-1",
		FishID:         "fish-1",
		Weight:         12,
		Unit:           "kg",
		Size:           "L",
		ProcessingType: catalog.ProcessingType("frozen"),
	})
	if err != ErrProcessingTypeNotFound {
		t.Fatalf("expected ErrProcessingTypeNotFound, got %v", err)
	}
}

func TestUpdateProductWritesOutbox(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-1", "Cod", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	seedRefs(deps, "kg", "frozen")

	product, _, err := catalog.NewProduct("prod-1", "fish-1", 10, "kg", "M", catalog.ProcessingType("frozen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.productRepo.Save(ctx, product); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := deps.svc.UpdateProduct(ctx, UpdateProductCommand{
		ProductID:      "prod-1",
		FishID:         "fish-1",
		Weight:         12,
		Unit:           "kg",
		Size:           "L",
		ProcessingType: catalog.ProcessingType("frozen"),
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deps.outbox.Count() != 1 {
		t.Fatalf("expected 1 outbox event, got %d", deps.outbox.Count())
	}
	if _, ok := deps.outbox.Last().(catalog.ProductUpdated); !ok {
		t.Fatalf("expected ProductUpdated event")
	}
}

func TestPublishLotRequiresPublishedProduct(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-2", "Salmon", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	product, _, err := catalog.NewProduct("prod-2", "fish-2", 10, "kg", "M", catalog.ProcessingType("frozen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.productRepo.Save(ctx, product); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schedule := catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour))
	lot, _, err := catalog.NewLot("lot-2", "prod-2", "seller-2", "", 10.0, int64(100), schedule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = lot.AssignAuctionID("auc-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.lotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = deps.svc.PublishLot(ctx, "lot-2")
	if err != catalog.ErrPublishingRuleViolation {
		t.Fatalf("expected ErrPublishingRuleViolation, got %v", err)
	}
	stored, err := deps.lotRepo.Get(ctx, "lot-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Status() != catalog.LotStatusDraft {
		t.Fatalf("expected lot to remain draft, got %s", stored.Status())
	}
	if deps.outbox.Count() != 0 {
		t.Fatalf("expected no outbox events, got %d", deps.outbox.Count())
	}
}

func TestPublishLotRequiresAuctionID(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	fish, err := catalog.NewFish("fish-3", "Hake", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.fishRepo.Save(ctx, fish); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	product, _, err := catalog.NewProduct("prod-3", "fish-3", 10, "kg", "M", catalog.ProcessingType("frozen"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := product.Publish(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.productRepo.Save(ctx, product); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	schedule := catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour))
	lot, _, err := catalog.NewLot("lot-3", "prod-3", "seller-3", "", 10.0, int64(100), schedule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.lotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = deps.svc.PublishLot(ctx, "lot-3")
	if err != catalog.ErrAuctionIDRequired {
		t.Fatalf("expected ErrAuctionIDRequired, got %v", err)
	}
	stored, err := deps.lotRepo.Get(ctx, "lot-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Status() != catalog.LotStatusDraft {
		t.Fatalf("expected lot to remain draft, got %s", stored.Status())
	}
	if deps.outbox.Count() != 0 {
		t.Fatalf("expected no outbox events, got %d", deps.outbox.Count())
	}
}

func TestAssignAuctionIDSavesLot(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	schedule := catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour))
	lot, _, err := catalog.NewLot("lot-3", "prod-3", "seller-3", "", 10.0, int64(100), schedule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.lotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := deps.svc.AssignAuctionID(ctx, "lot-3", "auc-3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, err := deps.lotRepo.Get(ctx, "lot-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.AuctionID() != "auc-3" {
		t.Fatalf("expected auction id to be saved, got %s", stored.AuctionID())
	}
}

func TestCloseLotWritesOutbox(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	schedule := catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour))
	lot, _, err := catalog.NewLot("lot-4", "prod-4", "seller-4", "", 10.0, int64(100), schedule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = lot.AssignAuctionID("auc-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = lot.Publish(true, newProductSnapshot())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.lotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := deps.svc.CloseLot(ctx, "lot-4", int64(150)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, err := deps.lotRepo.Get(ctx, "lot-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Status() != catalog.LotStatusClosed {
		t.Fatalf("expected lot to be closed, got %s", stored.Status())
	}
	if stored.FinalPrice() != int64(150) {
		t.Fatalf("expected final price to be updated, got %d", stored.FinalPrice())
	}
	if deps.outbox.Count() != 1 {
		t.Fatalf("expected 1 outbox event, got %d", deps.outbox.Count())
	}
	if _, ok := deps.outbox.Last().(catalog.LotClosed); !ok {
		t.Fatalf("expected LotClosed event")
	}
}

func TestHandleAuctionWonClosesByAuctionID(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	schedule := catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour))
	lot, _, err := catalog.NewLot("lot-5", "prod-5", "seller-5", "", 10.0, int64(100), schedule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = lot.AssignAuctionID("auc-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = lot.Publish(true, newProductSnapshot())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.lotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := deps.svc.HandleAuctionWon(ctx, AuctionWonDTO{AuctionID: "auc-5", FinalPrice: int64(150)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stored, err := deps.lotRepo.Get(ctx, "lot-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Status() != catalog.LotStatusClosed {
		t.Fatalf("expected lot to be closed, got %s", stored.Status())
	}
	if stored.FinalPrice() != int64(150) {
		t.Fatalf("expected final price to be updated, got %d", stored.FinalPrice())
	}
	if deps.outbox.Count() != 1 {
		t.Fatalf("expected 1 outbox event, got %d", deps.outbox.Count())
	}
	if _, ok := deps.outbox.Last().(catalog.LotClosed); !ok {
		t.Fatalf("expected LotClosed event")
	}
}

func TestHandleBidPlacedWritesLotPriceUpdatedToOutbox(t *testing.T) {
	deps := newTestDeps()
	ctx := context.Background()

	schedule := catalog.NewAuctionScheduleAt(time.Now().Add(time.Hour))
	lot, _, err := catalog.NewLot("lot-6", "prod-6", "seller-6", "", 10.0, int64(100), schedule)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = lot.AssignAuctionID("auc-6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = lot.Publish(true, newProductSnapshot())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := deps.lotRepo.Save(ctx, lot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := deps.svc.HandleBidPlaced(ctx, BidPlacedDTO{AuctionID: "auc-6", Amount: int64(145)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, err := deps.lotRepo.Get(ctx, "lot-6")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.CurPrice() != int64(145) {
		t.Fatalf("expected current price to be updated, got %d", stored.CurPrice())
	}
	if deps.outbox.Count() != 1 {
		t.Fatalf("expected 1 outbox event, got %d", deps.outbox.Count())
	}

	updated, ok := deps.outbox.Last().(catalog.LotPriceUpdated)
	if !ok {
		t.Fatalf("expected LotPriceUpdated event")
	}
	if updated.LotID != "lot-6" {
		t.Fatalf("expected lot id to match, got %s", updated.LotID)
	}
	if updated.AuctionID != "auc-6" {
		t.Fatalf("expected auction id to match, got %s", updated.AuctionID)
	}
	if updated.CurrentPrice != int64(145) {
		t.Fatalf("expected current price to match, got %d", updated.CurrentPrice)
	}
}
