package catalog

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
	Weight         int64
	Unit           string
	Size           string
	ProcessingType ProcessingType
	Status         ProductStatus
}

type ProductUpdated struct {
	ProductID      string
	FishID         string
	Weight         int64
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
	Quantity        int64
	Unit            Unit
	DeliveryTerms   string
	StorageTerms    string
	Status          LotStatus
}

type LotPublished struct {
	LotID     string
	ProductID string
	Status    LotStatus
}

type LotUnpublished struct {
	LotID  string
	Status LotStatus
}

type LotSold struct {
	LotID  string
	DealID string
	Status LotStatus
}

func (ProductCreated) isCatalogEvent()     {}
func (ProductUpdated) isCatalogEvent()     {}
func (ProductPublished) isCatalogEvent()   {}
func (ProductUnpublished) isCatalogEvent() {}
func (LotCreated) isCatalogEvent()         {}
func (LotPublished) isCatalogEvent()       {}
func (LotUnpublished) isCatalogEvent()     {}
func (LotSold) isCatalogEvent()            {}
