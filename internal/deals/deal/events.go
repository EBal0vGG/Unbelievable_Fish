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

type DealConfirmationRequested struct {
	ConfirmationID        string
	DealID                string
	Stage                 DealConfirmationStage
	RequestedByUserID     string
	RequestedByCompanyID  string
	CounterpartyCompanyID string
	VerificationMethod    VerificationMethod
	RequestedAt           time.Time
	ExpiresAt             *time.Time
	Comment               string
}

func (DealConfirmationRequested) isDealEvent() {}

type DealConfirmationApproved struct {
	ConfirmationID      string
	DealID              string
	Stage               DealConfirmationStage
	ApprovedByUserID    string
	ApprovedByCompanyID string
	ApprovedAt          time.Time
}

func (DealConfirmationApproved) isDealEvent() {}

type DealConfirmationRejected struct {
	ConfirmationID      string
	DealID              string
	Stage               DealConfirmationStage
	RejectedByUserID    string
	RejectedByCompanyID string
	RejectedAt          time.Time
	Reason              string
}

func (DealConfirmationRejected) isDealEvent() {}

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

// PaymentRequested - событие запроса оплаты (интеграция → Billing создаёт DealInvoice).
// InvoiceNumber не источник истины; номер задаёт провайдер/Billing.
type PaymentRequested struct {
	DealID          string
	AuctionID       string
	BuyerCompanyID  string
	SellerCompanyID string
	Currency        string // e.g. "RUB"
	GoodsAmount     int64  // сумма товара (лота); для аукциона при quantity=1 совпадает с финальной ценой
	InvoiceNumber   string // legacy / опционально
	DueDate         *time.Time
	RequestedAt     time.Time
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

// WinnerRejected — покупатель-участник топ-N отказался или просрочил подтверждение; billing удерживает депозит.
// SelectionID совпадает с AuctionID (одна winner selection на аукцион).
type WinnerRejected struct {
	SelectionID string
	DealID      string
	AuctionID   string
	CompanyID   string
	RejectedAt  time.Time
	Reason      string
}

func (WinnerRejected) isDealEvent() {}

// WinnerConfirmed — текущий кандидат подтвердил намерение заключить сделку; депозиты и комиссия не меняются до финализации оплаты.
// Keep this event small (ids, price, time only)—do not turn it into a snapshot/contract/invoice DTO.
type WinnerConfirmed struct {
	SelectionID string
	DealID      string
	AuctionID   string
	CompanyID   string
	FinalPrice  int64
	ConfirmedAt time.Time
}

func (WinnerConfirmed) isDealEvent() {}

// NextWinnerSelected — право покупки перешло следующему кандидату (новая сделка будет создана в том же или следующем шаге).
type NextWinnerSelected struct {
	SelectionID string
	AuctionID   string
	CompanyID   string
	Rank        int
	DealID      string
	SelectedAt  time.Time
}

func (NextWinnerSelected) isDealEvent() {}

// WinnerSelectionFailed — кандидаты исчерпаны, активной сделки по аукциону больше не будет.
type WinnerSelectionFailed struct {
	SelectionID string
	AuctionID   string
	FailedAt    time.Time
	Reason      string
}

func (WinnerSelectionFailed) isDealEvent() {}

// WinnerSelectionFinalized — оплата по инвойсу завершена, цепочка fallback закрыта; billing делает settlement депозитов.
// SelectionID совпадает с AuctionID.
type WinnerSelectionFinalized struct {
	SelectionID          string
	DealID               string
	AuctionID            string
	CompanyID            string
	FinalPrice           int64
	GoodsAmount          int64
	PlatformFeeDueAmount int64
	FinalizedAt          time.Time
}

func (WinnerSelectionFinalized) isDealEvent() {}

// PriceUpdated - событие обновления цены
type PriceUpdated struct {
	DealID    string
	OldPrice  int64
	NewPrice  int64
	UpdatedBy string
	UpdatedAt time.Time
}

func (PriceUpdated) isDealEvent() {}
