package wallet

import (
	"errors"
	"time"
)

type InvoiceStatus string

const (
	InvoicePending         InvoiceStatus = "PENDING"
	InvoicePaymentPending  InvoiceStatus = "PAYMENT_PENDING"
	InvoicePaid            InvoiceStatus = "PAID"
	InvoiceExpired         InvoiceStatus = "EXPIRED"
	InvoiceCancelled       InvoiceStatus = "CANCELLED"
	InvoiceFailed          InvoiceStatus = "FAILED"
)

// DealInvoice — требование оплаты по сделке (товар + недостающая часть комиссии после зачёта депозита).
//
// Снимок обязательства на момент создания: GoodsAmount, PlatformFeeDueAmount и TotalAmount
// не пересчитываются при последующих изменениях депозитов, комиссии или сделки.
type DealInvoice struct {
	ID                    string
	DealID                string
	AuctionID             string
	BuyerCompanyID        string
	SellerCompanyID       string
	GoodsAmount           int64
	PlatformFeeDueAmount  int64
	TotalAmount           int64
	Currency              Currency
	Status                InvoiceStatus
	Provider              string
	ProviderInvoiceID     string
	PaymentURL            string
	DueAt                 time.Time
	CreatedAt             time.Time
	PaidAt                *time.Time
	ExpiredAt             *time.Time
	CancelledAt           *time.Time
	FailedAt              *time.Time
}

var (
	ErrInvalidDealInvoice      = errors.New("invalid deal invoice")
	ErrInvoiceNotPayable       = errors.New("invoice cannot be paid in current status")
	ErrInvoiceNotExpirable     = errors.New("invoice cannot be expired in current status")
	ErrInvoiceAmountMismatch   = errors.New("payment amount does not match invoice")
	ErrInvoiceCurrencyMismatch = errors.New("payment currency does not match invoice")
)

func NewDealInvoice(
	id, dealID, auctionID, buyerID, sellerID string,
	goodsAmount, platformFeeDue int64,
	currency Currency,
	provider string,
	dueAt, createdAt time.Time,
) (*DealInvoice, error) {
	if id == "" || dealID == "" || buyerID == "" || sellerID == "" || provider == "" {
		return nil, ErrInvalidIdentifier
	}
	if buyerID == sellerID {
		return nil, ErrInvalidDealInvoice
	}
	if goodsAmount <= 0 || platformFeeDue < 0 {
		return nil, ErrInvalidAmount
	}
	if currency != CurrencyRUB {
		return nil, ErrUnsupportedCurrency
	}
	if dueAt.Before(createdAt) {
		return nil, ErrInvalidDealInvoice
	}
	total := goodsAmount + platformFeeDue
	return &DealInvoice{
		ID:                   id,
		DealID:               dealID,
		AuctionID:            auctionID,
		BuyerCompanyID:       buyerID,
		SellerCompanyID:      sellerID,
		GoodsAmount:          goodsAmount,
		PlatformFeeDueAmount: platformFeeDue,
		TotalAmount:          total,
		Currency:             currency,
		Status:               InvoicePending,
		Provider:             provider,
		DueAt:                dueAt,
		CreatedAt:            createdAt,
	}, nil
}

func (inv *DealInvoice) AttachProvider(providerInvoiceID, paymentURL string) error {
	if inv == nil {
		return ErrInvalidDealInvoice
	}
	if inv.Status != InvoicePending {
		return ErrInvalidDealInvoice
	}
	if providerInvoiceID == "" || paymentURL == "" {
		return ErrInvalidDealInvoice
	}
	inv.ProviderInvoiceID = providerInvoiceID
	inv.PaymentURL = paymentURL
	inv.Status = InvoicePaymentPending
	return nil
}

func (inv *DealInvoice) MarkPaid(amount int64, currency Currency, paidAt time.Time) error {
	if inv == nil {
		return ErrInvalidDealInvoice
	}
	if inv.Status != InvoicePaymentPending {
		return ErrInvoiceNotPayable
	}
	if amount != inv.TotalAmount || currency != inv.Currency {
		if currency != inv.Currency {
			return ErrInvoiceCurrencyMismatch
		}
		return ErrInvoiceAmountMismatch
	}
	inv.Status = InvoicePaid
	inv.PaidAt = &paidAt
	return nil
}

// MarkExpired transitions PAYMENT_PENDING → EXPIRED. PAID and EXPIRED are idempotent no-ops.
func (inv *DealInvoice) MarkExpired(now time.Time) error {
	if inv == nil {
		return ErrInvalidDealInvoice
	}
	switch inv.Status {
	case InvoicePaid, InvoiceExpired:
		return nil
	case InvoicePaymentPending:
		inv.Status = InvoiceExpired
		t := now.UTC()
		inv.ExpiredAt = &t
		return nil
	default:
		return ErrInvoiceNotExpirable
	}
}

// MarkPaidIdempotent sets PAID or returns nil if already PAID with matching amount/currency.
func (inv *DealInvoice) MarkPaidIdempotent(amount int64, currency Currency, paidAt time.Time) error {
	if inv == nil {
		return ErrInvalidDealInvoice
	}
	if inv.Status == InvoicePaid {
		if amount != inv.TotalAmount {
			return ErrInvoiceAmountMismatch
		}
		if currency != inv.Currency {
			return ErrInvoiceCurrencyMismatch
		}
		return nil
	}
	return inv.MarkPaid(amount, currency, paidAt)
}
