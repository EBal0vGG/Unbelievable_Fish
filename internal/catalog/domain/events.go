package catalog

import "time"

type Event interface {
	isCatalogEvent()
}

type LotAuctionLinked struct {
	LotID     string
	AuctionID string
}

func (LotAuctionLinked) isCatalogEvent() {}

type ProductCreated struct {
	ProductID      string
	FishID         string
	Weight         float64
	Unit           string
	Size           string
	ProcessingType ProcessingType
	Status         ProductStatus
}

type ProductUpdated struct {
	ProductID      string
	FishID         string
	Weight         float64
	Unit           string
	Size           string
	ProcessingType ProcessingType
	Status         ProductStatus
}

type ProductPublished struct {
	ProductID string
	Status    ProductStatus
}

type ProductUnpublished struct {
	ProductID string
	Status    ProductStatus
}

type LotCreated struct {
	LotID           string
	ProductID       string
	SellerCompanyID string
	Photo           string
	Quantity        float64
	Status          LotStatus
}

type LotPublished struct {
	LotID           string
	AuctionID       string
	SellerCompanyID string
	ProductID       string
	Product         ProductSnapshot
	StartPrice      int64
	MinBidStep      int64
	AuctionStartsAt time.Time
	AuctionEndsAt   time.Time
	Status          LotStatus
}

type LotUnpublished struct {
	LotID  string
	Status LotStatus
}

type LotClosed struct {
	LotID      string
	FinalPrice int64
	Status     LotStatus
}

type LotPriceUpdated struct {
	LotID        string
	AuctionID    string
	CurrentPrice int64
	Status       LotStatus
}

func (ProductCreated) isCatalogEvent()     {}
func (ProductUpdated) isCatalogEvent()     {}
func (ProductPublished) isCatalogEvent()   {}
func (ProductUnpublished) isCatalogEvent() {}
func (LotCreated) isCatalogEvent()         {}
func (LotPublished) isCatalogEvent()       {}
func (LotUnpublished) isCatalogEvent()     {}
func (LotClosed) isCatalogEvent()          {}
func (LotPriceUpdated) isCatalogEvent()    {}

type ProductSnapshot struct {
	ProductID      string
	Name           string
	Weight         float64
	Unit           string
	Size           string
	ProcessingType ProcessingType
}
