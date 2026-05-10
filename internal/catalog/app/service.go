package app

import (
	"context"
	"strings"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
)

type CatalogService struct {
	fishRepo           FishRepository
	unitRepo           UnitRepository
	processingTypeRepo ProcessingTypeRepository
	productRepo        ProductRepository
	lotRepo            LotRepository
	outbox             OutboxRepository
	idGenerator        IDGenerator
	tx                 TransactionManager
}

func NewCatalogService(
	fishRepo FishRepository,
	unitRepo UnitRepository,
	processingTypeRepo ProcessingTypeRepository,
	productRepo ProductRepository,
	lotRepo LotRepository,
	outbox OutboxRepository,
	idGenerator IDGenerator,
	tx TransactionManager,
) *CatalogService {
	if idGenerator == nil {
		idGenerator = NewRandomIDGenerator()
	}
	if tx == nil {
		tx = noopTransactionManager{}
	}

	return &CatalogService{
		fishRepo:           fishRepo,
		unitRepo:           unitRepo,
		processingTypeRepo: processingTypeRepo,
		productRepo:        productRepo,
		lotRepo:            lotRepo,
		outbox:             outbox,
		idGenerator:        idGenerator,
		tx:                 tx,
	}
}

func (s *CatalogService) CreateFish(ctx context.Context, cmd CreateFishCommand) (string, error) {
	var fishID string

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		fish, err := catalog.NewFish(s.idGenerator.NewFishID(), cmd.Name, cmd.Description)
		if err != nil {
			return err
		}
		if err := s.fishRepo.Save(ctx, fish); err != nil {
			return err
		}
		fishID = fish.ID()
		return nil
	})

	return fishID, err
}

func (s *CatalogService) ListFish(ctx context.Context) ([]*catalog.Fish, error) {
	return s.fishRepo.List(ctx)
}

func (s *CatalogService) UpdateFish(ctx context.Context, cmd UpdateFishCommand) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		fish, err := s.getFish(ctx, cmd.FishID)
		if err != nil {
			return err
		}
		if err := fish.Update(cmd.Name, cmd.Description); err != nil {
			return err
		}
		return s.fishRepo.Save(ctx, fish)
	})
}

func (s *CatalogService) CreateProduct(ctx context.Context, cmd CreateProductCommand) (string, []catalog.Event, error) {
	var (
		productID string
		events    []catalog.Event
	)

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		actor, ok := ActorFromContext(ctx)
		if !ok {
			return ErrMissingCompanyID
		}
		if err := s.validateProductRefs(ctx, cmd.FishID, cmd.Unit, cmd.ProcessingType); err != nil {
			return err
		}

		productID = s.idGenerator.NewProductID()
		product, evs, err := catalog.NewProduct(
			productID,
			cmd.FishID,
			actor.CompanyID,
			cmd.Weight,
			cmd.Unit,
			cmd.Size,
			cmd.ProcessingType,
		)
		if err != nil {
			return err
		}
		if err := s.productRepo.Save(ctx, product); err != nil {
			return err
		}
		if err := s.addEvents(ctx, evs); err != nil {
			return err
		}
		productID = product.ID()
		events = evs
		return nil
	})

	return productID, events, err
}

func (s *CatalogService) UpdateProduct(ctx context.Context, cmd UpdateProductCommand) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		product, err := s.getProduct(ctx, cmd.ProductID)
		if err != nil {
			return err
		}
		if err := s.validateProductRefs(ctx, cmd.FishID, cmd.Unit, cmd.ProcessingType); err != nil {
			return err
		}
		events, err := product.Update(
			cmd.FishID,
			cmd.Weight,
			cmd.Unit,
			cmd.Size,
			cmd.ProcessingType,
		)
		if err != nil {
			return err
		}
		if err := s.productRepo.Save(ctx, product); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) PublishProduct(ctx context.Context, productID string) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		actor, ok := ActorFromContext(ctx)
		if !ok {
			return ErrMissingCompanyID
		}
		product, err := s.getProduct(ctx, productID)
		if err != nil {
			return err
		}
		if !actorOwnsProduct(actor, product) {
			return ErrForbiddenOwner
		}
		events, err := product.Publish()
		if err != nil {
			return err
		}
		if err := s.productRepo.Save(ctx, product); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) UnpublishProduct(ctx context.Context, productID string) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		product, err := s.getProduct(ctx, productID)
		if err != nil {
			return err
		}
		events, err := product.Unpublish()
		if err != nil {
			return err
		}
		if err := s.productRepo.Save(ctx, product); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) CreateLot(ctx context.Context, cmd CreateLotCommand) (string, []catalog.Event, error) {
	var (
		lotID  string
		events []catalog.Event
	)

	err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
		actor, ok := ActorFromContext(ctx)
		if !ok {
			return ErrMissingCompanyID
		}
		product, err := s.getProduct(ctx, cmd.ProductID)
		if err != nil {
			return err
		}
		if !actorOwnsProduct(actor, product) {
			return ErrForbiddenOwner
		}
		companyID := actor.CompanyID

		duration := time.Hour
		if cmd.AuctionDurationMinutes > 0 {
			duration = time.Duration(cmd.AuctionDurationMinutes) * time.Minute
		}
		minBidStep := cmd.MinBidStep
		if minBidStep <= 0 {
			minBidStep = 1
		}
		schedule := catalog.NewAuctionScheduleAt(cmd.AuctionStartsAt, duration)
		lotID = s.idGenerator.NewLotID()
		lot, evs, err := catalog.NewLot(
			lotID,
			cmd.ProductID,
			companyID,
			cmd.Photo,
			cmd.Quantity,
			cmd.StartPrice,
			minBidStep,
			schedule,
		)
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		if err := s.addEvents(ctx, evs); err != nil {
			return err
		}
		lotID = lot.ID()
		events = evs
		return nil
	})

	return lotID, events, err
}

func (s *CatalogService) AssignAuctionID(ctx context.Context, lotID, auctionID string) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		lot, err := s.getLot(ctx, lotID)
		if err != nil {
			return err
		}
		events, err := lot.AssignAuctionID(auctionID)
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) GetLotAuctionID(ctx context.Context, lotID string) (string, error) {
	lot, err := s.getLot(ctx, lotID)
	if err != nil {
		return "", err
	}
	return lot.AuctionID(), nil
}

func (s *CatalogService) PublishLot(ctx context.Context, lotID string) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		actor, ok := ActorFromContext(ctx)
		if !ok {
			return ErrMissingCompanyID
		}
		lot, err := s.getLot(ctx, lotID)
		if err != nil {
			return err
		}
		if !actorOwnsLot(actor, lot) {
			return ErrForbiddenOwner
		}
		product, err := s.getProduct(ctx, lot.ProductID())
		if err != nil {
			return err
		}
		if !actorOwnsProduct(actor, product) {
			return ErrForbiddenOwner
		}
		if product.SellerCompanyID() != lot.SellerCompanyID() {
			return ErrForbiddenOwner
		}
		productIsPublished := product.Status() == catalog.ProductStatusPublished
		fish, err := s.getFish(ctx, product.FishID())
		if err != nil {
			return err
		}

		productSnapshot := catalog.ProductSnapshot{
			ProductID:      product.ID(),
			Name:           fish.Name(),
			Weight:         product.Weight(),
			Unit:           product.Unit(),
			Size:           product.Size(),
			ProcessingType: product.ProcessingType(),
		}

		events, err := lot.Publish(productIsPublished, productSnapshot)
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) UnpublishLot(ctx context.Context, lotID string) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		lot, err := s.getLot(ctx, lotID)
		if err != nil {
			return err
		}
		events, err := lot.Unpublish()
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) CloseLot(ctx context.Context, lotID string, finalPrice int64) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		lot, err := s.getLot(ctx, lotID)
		if err != nil {
			return err
		}
		events, err := lot.Close(finalPrice)
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) HandleAuctionWon(ctx context.Context, evt AuctionWonDTO) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		lot, err := s.getLotByAuctionID(ctx, evt.AuctionID)
		if err != nil {
			return err
		}
		if lot.Status() == catalog.LotStatusClosed {
			return nil
		}
		events, err := lot.Close(evt.FinalPrice)
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) HandleBidPlaced(ctx context.Context, evt BidPlacedDTO) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		lot, err := s.getLotByAuctionID(ctx, evt.AuctionID)
		if err != nil {
			return err
		}
		if lot.Status() != catalog.LotStatusPublished {
			return nil
		}
		events, err := lot.UpdateCurrentPrice(evt.Amount)
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) HandleAuctionClosed(ctx context.Context, evt AuctionClosedDTO) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		lot, err := s.getLotByAuctionID(ctx, evt.AuctionID)
		if err != nil {
			return err
		}
		if lot.Status() != catalog.LotStatusPublished {
			return nil
		}
		events, err := lot.Close(lot.CurPrice())
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) HandleAuctionCancelled(ctx context.Context, evt AuctionCancelledDTO) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		lot, err := s.getLotByAuctionID(ctx, evt.AuctionID)
		if err != nil {
			return err
		}
		if lot.Status() != catalog.LotStatusPublished {
			return nil
		}
		events, err := lot.Unpublish()
		if err != nil {
			return err
		}
		if err := s.lotRepo.Save(ctx, lot); err != nil {
			return err
		}
		return s.addEvents(ctx, events)
	})
}

func (s *CatalogService) addEvents(ctx context.Context, events []catalog.Event) error {
	if len(events) == 0 {
		return nil
	}
	return s.outbox.Add(ctx, events)
}

func (s *CatalogService) getFish(ctx context.Context, fishID string) (*catalog.Fish, error) {
	fish, err := s.fishRepo.Get(ctx, fishID)
	if err != nil {
		return nil, err
	}
	if fish == nil {
		return nil, ErrNotFound
	}
	return fish, nil
}

func (s *CatalogService) validateProductRefs(ctx context.Context, fishID, unit string, processingType catalog.ProcessingType) error {
	exists, err := s.fishRepo.Exists(ctx, fishID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrFishNotFound
	}

	normalizedUnit := strings.TrimSpace(unit)
	exists, err = s.unitRepo.Exists(ctx, normalizedUnit)
	if err != nil {
		return err
	}
	if !exists {
		return ErrUnitNotFound
	}

	normalizedProcessingType := strings.TrimSpace(string(processingType))
	exists, err = s.processingTypeRepo.Exists(ctx, normalizedProcessingType)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProcessingTypeNotFound
	}

	return nil
}

func (s *CatalogService) getProduct(ctx context.Context, productID string) (*catalog.Product, error) {
	product, err := s.productRepo.Get(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrNotFound
	}
	return product, nil
}

func (s *CatalogService) getLot(ctx context.Context, lotID string) (*catalog.Lot, error) {
	lot, err := s.lotRepo.Get(ctx, lotID)
	if err != nil {
		return nil, err
	}
	if lot == nil {
		return nil, ErrNotFound
	}
	return lot, nil
}

func (s *CatalogService) getLotByAuctionID(ctx context.Context, auctionID string) (*catalog.Lot, error) {
	lot, err := s.lotRepo.GetByAuctionID(ctx, auctionID)
	if err != nil {
		return nil, err
	}
	if lot == nil {
		return nil, ErrNotFound
	}
	return lot, nil
}

func (s *CatalogService) ListProducts(ctx context.Context) ([]*catalog.Product, error) {
	actor, ok := ActorFromContext(ctx)
	if !ok {
		return nil, ErrMissingCompanyID
	}
	if actor.isPlatformAdmin() {
		return s.productRepo.List(ctx)
	}
	if !actor.SellerCatalogAccess {
		return nil, ErrCatalogListForbidden
	}
	return s.productRepo.ListBySellerCompany(ctx, actor.CompanyID)
}

func (s *CatalogService) ListLots(ctx context.Context) ([]*catalog.Lot, error) {
	actor, ok := ActorFromContext(ctx)
	if !ok {
		return nil, ErrMissingCompanyID
	}
	if actor.isPlatformAdmin() {
		return s.lotRepo.List(ctx)
	}
	if !actor.SellerCatalogAccess {
		return nil, ErrCatalogListForbidden
	}
	return s.lotRepo.ListBySellerCompany(ctx, actor.CompanyID)
}

func actorOwnsProduct(actor Actor, p *catalog.Product) bool {
	if actor.isPlatformAdmin() {
		return true
	}
	return p.SellerCompanyID() == actor.CompanyID
}

func actorOwnsLot(actor Actor, l *catalog.Lot) bool {
	if actor.isPlatformAdmin() {
		return true
	}
	return l.SellerCompanyID() == actor.CompanyID
}
