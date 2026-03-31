package app

import (
	"time"

	"unbelievable_fish/internal/deals/deal"
)

type CreateProjectionCommand struct {
	AuctionID       string
	SupplierID      string
	ProductSnapshot deal.ProductSnapshot
	StartPrice      int64
	PublishedAt     time.Time
}

type CreateDealFromAuctionWonCommand struct {
	AuctionID       string
	WinnerCompanyID string
	FinalPrice      int64
	WonAt           time.Time
}

type GetDealByIDQuery struct {
	DealID string
}

type GetDealByAuctionIDQuery struct {
	AuctionID string
}

type GetProjectionByAuctionIDQuery struct {
	AuctionID string
}

type ConfirmDealCommand struct {
	DealID string
}

type PrepareContractCommand struct {
	DealID         string
	ContractNumber string
	DocumentURL    string
}

type SignContractCommand struct {
	DealID       string
	SignedBy     string
	SignatureRef string
}

type RequestPaymentCommand struct {
	DealID        string
	InvoiceNumber string
	DueDate       *time.Time
}

type MarkDealPaidCommand struct {
	DealID      string
	PaymentID   string
	PaymentType string
}

type RequestShipmentCommand struct {
	DealID string
}

type MarkDealShippedCommand struct {
	DealID         string
	TrackingNumber string
	Carrier        string
}

type CompleteDealCommand struct {
	DealID string
}

type CancelDealCommand struct {
	DealID      string
	Reason      string
	CancelledBy string
}

type UpdateDealPriceCommand struct {
	DealID    string
	NewPrice  int64
	UpdatedBy string
}
