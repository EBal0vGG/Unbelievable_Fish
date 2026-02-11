package deal

import (
	"time"
)

// Event - маркерный интерфейс для всех событий сделки
type Event interface {
	isDealEvent()
}

// DealCreated - событие создания сделки при выигрыше аукциона
type DealCreated struct {
	DealID          string
	AuctionID       string
	CustomerID      string
	SupplierID      string
	ProductSnapshot ProductSnapshot
	FinalPrice      int64
	CreatedAt       time.Time
}

func (DealCreated) isDealEvent() {}

// DealConfirmed - событие подтверждения сделки
type DealConfirmed struct {
	DealID      string
	ConfirmedAt time.Time
}

func (DealConfirmed) isDealEvent() {}

// ContractPrepared - событие подготовки контракта
type ContractPrepared struct {
	DealID         string
	ContractNumber string
	PreparedAt     time.Time
	DocumentURL    string
}

func (ContractPrepared) isDealEvent() {}

// ContractSigned - событие подписания контракта
type ContractSigned struct {
	DealID         string
	ContractNumber string
	SignedAt       time.Time
	SignedBy       string
	SignatureRef   string
}

func (ContractSigned) isDealEvent() {}

// PaymentRequested - событие запроса оплаты
type PaymentRequested struct {
	DealID        string
	TotalAmount   int64
	InvoiceNumber string
	DueDate       *time.Time
	RequestedAt   time.Time
}

func (PaymentRequested) isDealEvent() {}

// DealPaid - событие оплаты сделки
type DealPaid struct {
	DealID      string
	PaymentID   string
	PaymentType string
	PaidAt      time.Time
}

func (DealPaid) isDealEvent() {}

// ShipmentRequested - событие запроса доставки
type ShipmentRequested struct {
	DealID      string
	RequestedAt time.Time
}

func (ShipmentRequested) isDealEvent() {}

// DealShipped - событие отправки сделки
type DealShipped struct {
	DealID         string
	TrackingNumber string
	Carrier        string
	ShippedAt      time.Time
}

func (DealShipped) isDealEvent() {}

// DealCompleted - событие завершения сделки
type DealCompleted struct {
	DealID      string
	CompletedAt time.Time
}

func (DealCompleted) isDealEvent() {}

// DealCancelled - событие отмены сделки
type DealCancelled struct {
	DealID      string
	Reason      string
	CancelledBy string
	CancelledAt time.Time
}

func (DealCancelled) isDealEvent() {}

// PriceUpdated - событие обновления цены
type PriceUpdated struct {
	DealID    string
	OldPrice  int64
	NewPrice  int64
	UpdatedBy string
	UpdatedAt time.Time
}

func (PriceUpdated) isDealEvent() {}
