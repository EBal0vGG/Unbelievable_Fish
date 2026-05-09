package wallet

import (
	"errors"
	"time"
)

type SellerPayoutStatus string

const (
	SellerPayoutPending   SellerPayoutStatus = "PENDING"
	SellerPayoutReady     SellerPayoutStatus = "READY"
	SellerPayoutPaid      SellerPayoutStatus = "PAID"
	SellerPayoutCancelled SellerPayoutStatus = "CANCELLED"
	SellerPayoutFailed    SellerPayoutStatus = "FAILED"
)

// SellerPayout — обязательство платформы выплатить продавцу сумму товара по оплаченной сделке (не wallet credit на Stage 12).
type SellerPayout struct {
	ID      string
	DealID  string
	// InvoiceID — снимок оплаченного счёта: один PAID invoice → один payout (UNIQUE deal_id и UNIQUE invoice_id в БД).
	// Сумма/стороны фиксируются при создании; дальше меняется только статус выплаты и метки времени жизненного цикла.
	InvoiceID string
	AuctionID        string
	SellerCompanyID  string
	BuyerCompanyID   string
	Amount           int64
	Currency         Currency
	Status           SellerPayoutStatus
	CreatedAt        time.Time
	ReadyAt          *time.Time
	PaidAt           *time.Time
	CancelledAt      *time.Time
	FailedAt         *time.Time
}

var (
	ErrInvalidSellerPayout = errors.New("invalid seller payout")
)

func NewSellerPayout(
	id, dealID, invoiceID, auctionID, sellerCompanyID, buyerCompanyID string,
	amount int64,
	currency Currency,
	status SellerPayoutStatus,
	createdAt time.Time,
) (*SellerPayout, error) {
	if id == "" || dealID == "" || invoiceID == "" || auctionID == "" || sellerCompanyID == "" || buyerCompanyID == "" {
		return nil, ErrInvalidIdentifier
	}
	if sellerCompanyID == buyerCompanyID {
		return nil, ErrInvalidSellerPayout
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if currency != CurrencyRUB {
		return nil, ErrUnsupportedCurrency
	}
	if createdAt.IsZero() {
		return nil, ErrInvalidSellerPayout
	}
	switch status {
	case SellerPayoutPending, SellerPayoutReady, SellerPayoutPaid, SellerPayoutCancelled, SellerPayoutFailed:
	default:
		return nil, ErrInvalidSellerPayout
	}
	return &SellerPayout{
		ID:              id,
		DealID:          dealID,
		InvoiceID:       invoiceID,
		AuctionID:       auctionID,
		SellerCompanyID: sellerCompanyID,
		BuyerCompanyID:  buyerCompanyID,
		Amount:          amount,
		Currency:        currency,
		Status:          status,
		CreatedAt:       createdAt.UTC(),
	}, nil
}
