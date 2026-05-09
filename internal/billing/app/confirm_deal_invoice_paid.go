package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type ConfirmDealInvoicePaid struct {
	invoices DealInvoiceRepository
	events   DomainEventPublisher
	clock    Clock
}

func NewConfirmDealInvoicePaid(invoices DealInvoiceRepository, events DomainEventPublisher, clock Clock) (*ConfirmDealInvoicePaid, error) {
	if invoices == nil {
		return nil, ErrNilDependency
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &ConfirmDealInvoicePaid{invoices: invoices, events: events, clock: clock}, nil
}

func (uc *ConfirmDealInvoicePaid) Execute(ctx context.Context, invoiceID string) error {
	if isBlank(invoiceID) {
		return wallet.ErrInvalidIdentifier
	}
	paidAt := uc.clock.Now()
	inv, err := uc.invoices.LoadByIDForUpdate(ctx, invoiceID)
	if err != nil {
		return err
	}
	alreadyPaid := inv.Status == wallet.InvoicePaid
	if err := inv.MarkPaidIdempotent(inv.TotalAmount, inv.Currency, paidAt); err != nil {
		return err
	}
	if alreadyPaid {
		return nil
	}
	if err := uc.invoices.Save(ctx, inv); err != nil {
		return err
	}
	if uc.events == nil {
		return nil
	}
	return uc.events.Publish(ctx, inv.DealID, inv.BuyerCompanyID, wallet.DealInvoicePaid{
		InvoiceID:            inv.ID,
		DealID:               inv.DealID,
		AuctionID:            inv.AuctionID,
		BuyerCompanyID:       inv.BuyerCompanyID,
		GoodsAmount:          inv.GoodsAmount,
		PlatformFeeDueAmount: inv.PlatformFeeDueAmount,
		Amount:               inv.TotalAmount,
		Currency:             inv.Currency,
		PaidAt:               paidAt,
	})
}
