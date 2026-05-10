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
	SellerPayoutCancelled SellerPayoutStatus = "CANCELLED" // reserved for future ops / no Mark* yet
	SellerPayoutFailed    SellerPayoutStatus = "FAILED"    // reserved for future ops / no Mark* yet
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
	ErrInvalidSellerPayout      = errors.New("invalid seller payout")
	ErrSellerPayoutWrongStatus  = errors.New("seller payout: invalid status for this operation")
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

// MarkReady transitions PENDING → READY (admin / ops approval before crediting seller balance).
func (p *SellerPayout) MarkReady(now time.Time) error {
	if p == nil {
		return ErrInvalidSellerPayout
	}
	if p.Status == SellerPayoutReady || p.Status == SellerPayoutPaid {
		return nil // idempotent no-op
	}
	if p.Status != SellerPayoutPending {
		return ErrSellerPayoutWrongStatus
	}
	t := now.UTC()
	p.Status = SellerPayoutReady
	p.ReadyAt = &t
	return nil
}

// MarkFailed transitions PENDING or READY → FAILED (no balance movement).
func (p *SellerPayout) MarkFailed(now time.Time) error {
	if p == nil {
		return ErrInvalidSellerPayout
	}
	if p.Status == SellerPayoutFailed {
		return nil
	}
	if p.Status == SellerPayoutPaid {
		return ErrSellerPayoutWrongStatus
	}
	if p.Status != SellerPayoutPending && p.Status != SellerPayoutReady {
		return ErrSellerPayoutWrongStatus
	}
	t := now.UTC()
	p.Status = SellerPayoutFailed
	p.FailedAt = &t
	return nil
}

// MarkPaid transitions READY → PAID (credits seller.available in application layer, not here).
func (p *SellerPayout) MarkPaid(now time.Time) error {
	if p == nil {
		return ErrInvalidSellerPayout
	}
	if p.Status == SellerPayoutPaid {
		return nil
	}
	if p.Status != SellerPayoutReady {
		return ErrSellerPayoutWrongStatus
	}
	t := now.UTC()
	p.Status = SellerPayoutPaid
	p.PaidAt = &t
	return nil
}
