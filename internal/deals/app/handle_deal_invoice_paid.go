package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type HandleDealInvoicePaid struct {
	uow UnitOfWork
}

func NewHandleDealInvoicePaid(uow UnitOfWork) (*HandleDealInvoicePaid, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &HandleDealInvoicePaid{uow: uow}, nil
}

func (uc *HandleDealInvoicePaid) Execute(ctx context.Context, evt wallet.DealInvoicePaid) error {
	if evt.DealID == "" {
		return deal.ErrDealIDRequired
	}
	if evt.AuctionID == "" {
		return deal.ErrAuctionIDRequired
	}
	if evt.BuyerCompanyID == "" {
		return deal.ErrCustomerIDRequired
	}
	return uc.uow.Do(ctx, func(tx Tx) error {
		d, err := tx.Deals().GetByIDForUpdate(ctx, evt.DealID)
		if err != nil {
			return err
		}
		if d.Status() == deal.DealStatusCancelled {
			return deal.ErrDealCancelledPayment
		}
		if d.Status() == deal.DealStatusPaid {
			return nil
		}
		if d.CustomerID() != evt.BuyerCompanyID {
			return deal.ErrNotDealParticipant
		}
		if d.AuctionID() != evt.AuctionID {
			return deal.ErrStaleWinnerSelection
		}
		if err := validateDealInvoicePaidPayload(d, evt); err != nil {
			return err
		}
		if d.Status() != deal.DealStatusPaymentRequested {
			return deal.ErrCannotMarkAsPaid
		}
		paidEvents, err := d.MarkAsPaid(evt.InvoiceID, "invoice")
		if err != nil {
			return err
		}
		sel, err := tx.Selections().GetByAuctionIDForUpdate(ctx, evt.AuctionID)
		if err != nil {
			return err
		}
		if sel.DealID != d.ID() {
			return deal.ErrWinnerSelectionDealMismatch
		}
		if sel.Status == deal.WinnerSelectionStatusFinalized {
			if err := tx.Deals().Save(ctx, d); err != nil {
				return err
			}
			return tx.Outbox().Add(ctx, paidEvents)
		}
		if sel.Status != deal.WinnerSelectionConfirmedPendingPayment {
			return deal.ErrWinnerSelectionNotAwaitingPayment
		}
		if err := sel.MarkFinalized(d.ID()); err != nil {
			return err
		}
		cand, ok := sel.CurrentCandidate()
		if !ok {
			return deal.ErrNoAvailableWinnerCandidate
		}
		finalEvt := deal.WinnerSelectionFinalized{
			SelectionID:          evt.AuctionID,
			DealID:               d.ID(),
			AuctionID:            evt.AuctionID,
			CompanyID:            cand,
			FinalPrice:           sel.FinalPrice,
			GoodsAmount:          evt.GoodsAmount,
			PlatformFeeDueAmount: evt.PlatformFeeDueAmount,
			FinalizedAt:          evt.PaidAt,
		}
		events := append(append([]deal.Event(nil), paidEvents...), finalEvt)
		if err := tx.Deals().Save(ctx, d); err != nil {
			return err
		}
		if err := tx.Selections().Save(ctx, sel); err != nil {
			return err
		}
		return tx.Outbox().Add(ctx, events)
	})
}

func validateDealInvoicePaidPayload(d *deal.Deal, evt wallet.DealInvoicePaid) error {
	goods := d.CalculateTotal()
	if evt.GoodsAmount > 0 || evt.PlatformFeeDueAmount > 0 {
		if evt.GoodsAmount != goods {
			return deal.ErrDealInvoicePaidInvariant
		}
		if evt.Amount != evt.GoodsAmount+evt.PlatformFeeDueAmount {
			return deal.ErrDealInvoicePaidInvariant
		}
		return nil
	}
	if evt.Amount < goods {
		return deal.ErrDealInvoicePaidInvariant
	}
	return nil
}
