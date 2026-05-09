package app

import (
	"context"
	"log/slog"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type HandleDealInvoiceExpired struct {
	uow     UnitOfWork
	factory *deal.Factory
	clock   Clock
}

func NewHandleDealInvoiceExpired(uow UnitOfWork, clock Clock) (*HandleDealInvoiceExpired, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &HandleDealInvoiceExpired{uow: uow, factory: deal.NewFactory(), clock: clock}, nil
}

func (uc *HandleDealInvoiceExpired) Execute(ctx context.Context, evt wallet.DealInvoiceExpired) error {
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
		if d.Status() == deal.DealStatusPaid || d.Status() == deal.DealStatusCancelled {
			return nil
		}
		if d.AuctionID() != evt.AuctionID || d.CustomerID() != evt.BuyerCompanyID {
			return deal.ErrStaleWinnerSelection
		}
		if !isDealPostConfirmPrePaid(d.Status()) {
			return deal.ErrCannotCancelForPaymentTimeout
		}

		sel, err := tx.Selections().GetByAuctionIDForUpdate(ctx, evt.AuctionID)
		if err != nil {
			return err
		}
		if sel == nil {
			return deal.ErrSelectionNotFound
		}

		if sel.Status == deal.WinnerSelectionStatusFinalized || sel.Status == deal.WinnerSelectionExhausted {
			return nil
		}
		if sel.Status == deal.WinnerSelectionActive {
			slog.WarnContext(ctx, "deal_invoice_expired_no_op_selection_active",
				"component", "deals.app",
				"operation", "HandleDealInvoiceExpired",
				"reason", "selection_already_active_replay_or_advanced",
				"auction_id", evt.AuctionID,
				"deal_id", evt.DealID,
				"invoice_id", evt.InvoiceID,
				"selection_status", string(sel.Status),
				"selection_deal_id", sel.DealID,
			)
			return nil
		}
		if sel.Status != deal.WinnerSelectionConfirmedPendingPayment {
			return nil
		}
		if sel.DealID != d.ID() {
			slog.WarnContext(ctx, "deal_invoice_expired_no_op_selection_deal_mismatch",
				"component", "deals.app",
				"operation", "HandleDealInvoiceExpired",
				"reason", "replay_or_state_mismatch",
				"auction_id", evt.AuctionID,
				"deal_id", evt.DealID,
				"invoice_id", evt.InvoiceID,
				"selection_status", string(sel.Status),
				"selection_deal_id", sel.DealID,
			)
			return nil
		}

		cancelEvents, err := d.Cancel(deal.DealCancelReasonPaymentTimeout, "system")
		if err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, d); err != nil {
			return err
		}
		if err := sel.ReopenAfterPaymentTimeout(d.ID()); err != nil {
			return err
		}

		now := uc.clock.Now()
		if !sel.Advance() {
			if err := tx.Selections().Save(ctx, sel); err != nil {
				return err
			}
			batch := append(append([]deal.Event(nil), cancelEvents...), deal.WinnerSelectionFailed{
				SelectionID: evt.AuctionID,
				AuctionID:   evt.AuctionID,
				FailedAt:    now,
				Reason:      "NO_CANDIDATES_LEFT",
			})
			return tx.Outbox().Add(ctx, batch)
		}

		next, ok := sel.CurrentCandidate()
		if !ok {
			sel.MarkExhausted()
			if err := tx.Selections().Save(ctx, sel); err != nil {
				return err
			}
			batch := append(append([]deal.Event(nil), cancelEvents...), deal.WinnerSelectionFailed{
				SelectionID: evt.AuctionID,
				AuctionID:   evt.AuctionID,
				FailedAt:    now,
				Reason:      "NO_CANDIDATES_LEFT",
			})
			return tx.Outbox().Add(ctx, batch)
		}

		item, createEvents, err := uc.factory.CreateFromSelection(
			sel.AuctionID,
			sel.SupplierID,
			sel.ProductSnapshot,
			next,
			sel.FinalPrice,
			sel.WonAt,
		)
		if err != nil {
			return err
		}
		if err := item.Validate(); err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		sel.DealID = item.ID()
		if err := tx.Selections().Save(ctx, sel); err != nil {
			return err
		}
		rank := sel.CurrentIndex + 1
		nextWinner := deal.NextWinnerSelected{
			SelectionID: sel.AuctionID,
			AuctionID:   sel.AuctionID,
			CompanyID:   next,
			Rank:        rank,
			DealID:      item.ID(),
			SelectedAt:  now,
		}
		batch := append(append([]deal.Event(nil), cancelEvents...), append(createEvents, nextWinner)...)
		return tx.Outbox().Add(ctx, batch)
	})
}

func isDealPostConfirmPrePaid(s deal.DealStatus) bool {
	switch s {
	case deal.DealStatusConfirmed, deal.DealStatusContractPrepared, deal.DealStatusContractSigned, deal.DealStatusPaymentRequested:
		return true
	default:
		return false
	}
}
