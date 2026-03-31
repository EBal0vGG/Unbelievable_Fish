package app

import (
	"context"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/domain/catalog"
)

type FishRepository interface {
	Get(ctx context.Context, fishID string) (*catalog.Fish, error)
	Exists(ctx context.Context, fishID string) (bool, error)
	Save(ctx context.Context, fish *catalog.Fish) error
}

type ProductRepository interface {
	Get(ctx context.Context, productID string) (*catalog.Product, error)
	Save(ctx context.Context, product *catalog.Product) error
}

type LotRepository interface {
	Get(ctx context.Context, lotID string) (*catalog.Lot, error)
	GetByAuctionID(ctx context.Context, auctionID string) (*catalog.Lot, error)
	Save(ctx context.Context, lot *catalog.Lot) error
}

type OutboxRepository interface {
	Add(ctx context.Context, events []catalog.Event) error
}

type UnitRepository interface {
	Exists(ctx context.Context, unit string) (bool, error)
}

type ProcessingTypeRepository interface {
	Exists(ctx context.Context, processingType string) (bool, error)
}

type TransactionManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type CreateFishCommand struct {
	Name        string
	Description string
}

type UpdateFishCommand struct {
	FishID      string
	Name        string
	Description string
}

type CreateProductCommand struct {
	FishID         string
	Weight         float64
	Unit           string
	Size           string
	ProcessingType catalog.ProcessingType
}

type UpdateProductCommand struct {
	ProductID      string
	FishID         string
	Weight         float64
	Unit           string
	Size           string
	ProcessingType catalog.ProcessingType
}

type CreateLotCommand struct {
	ProductID       string
	SellerCompanyID string
	Photo           string
	Quantity        float64
	StartPrice      int64
	AuctionStartsAt time.Time
}

type AuctionWonDTO struct {
	AuctionID       string
	FinalPrice      int64
	WinnerCompanyID string
}

type BidPlacedDTO struct {
	AuctionID string
	Amount    int64
}

type AuctionClosedDTO struct {
	AuctionID string
}

type AuctionCancelledDTO struct {
	AuctionID string
}
