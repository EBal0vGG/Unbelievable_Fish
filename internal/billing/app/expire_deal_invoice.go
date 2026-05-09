package app

import (
	"context"
	"errors"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

var ErrInvoiceNotExpired = errors.New("billing: invoice due date has not passed")

type ExpireDealInvoice struct {
	invoices DealInvoiceRepository
	events   DomainEventPublisher
	clock    Clock
}

func NewExpireDealInvoice(invoices DealInvoiceRepository, events DomainEventPublisher, clock Clock) (*ExpireDealInvoice, error) {
	if invoices == nil {
		return nil, ErrNilDependency
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &ExpireDealInvoice{invoices: invoices, events: events, clock: clock}, nil
}

func (uc *ExpireDealInvoice) Execute(ctx context.Context, invoiceID string) error {
	if isBlank(invoiceID) {
		return wallet.ErrInvalidIdentifier
	}
	now := uc.clock.Now().UTC()
	inv, err := uc.invoices.LoadByIDForUpdate(ctx, invoiceID)
	if err != nil {
		return err
	}
	switch inv.Status {
	case wallet.InvoicePaid, wallet.InvoiceExpired:
		return nil
	}
	if inv.Status != wallet.InvoicePaymentPending {
		return wallet.ErrInvoiceNotExpirable
	}
	if inv.DueAt.After(now) {
		return ErrInvoiceNotExpired
	}
	if err := inv.MarkExpired(now); err != nil {
		return err
	}
	if err := uc.invoices.Save(ctx, inv); err != nil {
		return err
	}
	if uc.events == nil {
		return nil
	}
	return uc.events.Publish(ctx, inv.DealID, inv.BuyerCompanyID, wallet.DealInvoiceExpired{
		InvoiceID:      inv.ID,
		DealID:         inv.DealID,
		AuctionID:      inv.AuctionID,
		BuyerCompanyID: inv.BuyerCompanyID,
		Amount:         inv.TotalAmount,
		Currency:       inv.Currency,
		ExpiredAt:      now,
	})
}
