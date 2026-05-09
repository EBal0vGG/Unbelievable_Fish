package wallet

import "time"

// Domain events for future billing outbox (stage 2+).

type AccountCreated struct {
	AccountID string
	CompanyID string
	Currency  Currency
}

type BalanceToppedUp struct {
	AccountID string
	CompanyID string
	Amount    int64
	Currency  Currency
}

type AuctionDepositReserved struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
}

type AuctionDepositReleased struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
	Reason    string
}

type AuctionDepositCaptured struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
	Reason    string
}

type PlatformFeeCaptured struct {
	AuctionID string
	CompanyID string
	Amount    int64
	Currency  Currency
}

type PlatformFeePaymentRequired struct {
	AuctionID string
	CompanyID string
	AmountDue int64
	Currency  Currency
}

// DealInvoiceCreated — инвойс создан, есть payment_url (slim integration event).
type DealInvoiceCreated struct {
	InvoiceID            string
	DealID               string
	AuctionID            string
	BuyerCompanyID       string
	SellerCompanyID      string
	GoodsAmount          int64
	PlatformFeeDueAmount int64
	TotalAmount          int64
	Currency             Currency
	PaymentURL           string
	DueAt                time.Time
	CreatedAt            time.Time
}

// DealInvoiceExpired — срок оплаты инвойса истёк; deals обрабатывают fallback победителя.
type DealInvoiceExpired struct {
	InvoiceID      string
	DealID         string
	AuctionID      string
	BuyerCompanyID string
	Amount         int64
	Currency       Currency
	ExpiredAt      time.Time
}

// DealInvoicePaid — оплата подтверждена (без движения wallet balance на Stage 9).
type DealInvoicePaid struct {
	InvoiceID            string
	DealID               string
	AuctionID            string
	BuyerCompanyID       string
	GoodsAmount          int64
	PlatformFeeDueAmount int64
	Amount               int64 // total paid (= GoodsAmount + PlatformFeeDueAmount)
	Currency             Currency
	PaidAt               time.Time
}

// SellerPayoutCreated — создана запись обязательства выплаты продавцу (goods_amount, статус PENDING на Stage 12).
type SellerPayoutCreated struct {
	PayoutID        string
	DealID          string
	InvoiceID       string
	AuctionID       string
	SellerCompanyID string
	BuyerCompanyID  string
	Amount          int64
	Currency        Currency
	CreatedAt       time.Time
}
