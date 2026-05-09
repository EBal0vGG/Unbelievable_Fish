package app

import (
	"context"
	"errors"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

// CreateSellerPayoutCommand mirrors deals.WinnerSelectionFinalized fields needed to cross-check the paid invoice.
type CreateSellerPayoutCommand struct {
	DealID               string
	AuctionID            string
	BuyerCompanyID       string // winner / buyer from selection finalized (deals event CompanyID)
	GoodsAmount          int64
	PlatformFeeDueAmount int64
}

type CreateSellerPayout struct {
	payouts  SellerPayoutRepository
	invoices DealInvoiceRepository
	ids      IDGenerator
	clock    Clock
	events   DomainEventPublisher
}

func NewCreateSellerPayout(
	payouts SellerPayoutRepository,
	invoices DealInvoiceRepository,
	ids IDGenerator,
	clock Clock,
	events DomainEventPublisher,
) (*CreateSellerPayout, error) {
	if payouts == nil || invoices == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &CreateSellerPayout{payouts: payouts, invoices: invoices, ids: ids, clock: clock, events: events}, nil
}

func (uc *CreateSellerPayout) Execute(ctx context.Context, cmd CreateSellerPayoutCommand) (*wallet.SellerPayout, error) {
	if isBlank(cmd.DealID) || isBlank(cmd.AuctionID) || isBlank(cmd.BuyerCompanyID) {
		return nil, wallet.ErrInvalidIdentifier
	}
	if cmd.GoodsAmount <= 0 {
		return nil, wallet.ErrInvalidAmount
	}

	existing, err := uc.payouts.LoadByDealIDForUpdate(ctx, cmd.DealID)
	if err == nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, ErrSellerPayoutNotFound) {
		return nil, err
	}

	inv, err := uc.invoices.LoadByDealIDForUpdate(ctx, cmd.DealID)
	if err != nil {
		return nil, err
	}
	if inv.Status != wallet.InvoicePaid {
		return nil, wallet.ErrInvoiceNotPayable
	}
	if inv.AuctionID != cmd.AuctionID {
		return nil, ErrSellerPayoutInvoiceMismatch
	}
	if inv.BuyerCompanyID != cmd.BuyerCompanyID {
		return nil, ErrSellerPayoutInvoiceMismatch
	}
	if inv.GoodsAmount != cmd.GoodsAmount {
		return nil, ErrSellerPayoutInvoiceMismatch
	}
	// TODO(Stage 12+): seller payout is driven by invoice snapshot (goods, parties, deal).
	// Do not tie payout to selection fee once partial fee adjustments / disputes exist — fee is platform concern.
	if inv.PlatformFeeDueAmount != cmd.PlatformFeeDueAmount {
		return nil, ErrSellerPayoutInvoiceMismatch
	}
	if inv.BuyerCompanyID == inv.SellerCompanyID {
		return nil, wallet.ErrInvalidSellerPayout
	}

	now := uc.clock.Now().UTC()
	payout, err := wallet.NewSellerPayout(
		uc.ids.NewID(),
		inv.DealID,
		inv.ID,
		inv.AuctionID,
		inv.SellerCompanyID,
		inv.BuyerCompanyID,
		inv.GoodsAmount,
		inv.Currency,
		wallet.SellerPayoutPending,
		now,
	)
	if err != nil {
		return nil, err
	}
	if err := uc.payouts.Create(ctx, payout); err != nil {
		if IsPostgresUniqueViolation(err) {
			// Concurrent handler: other tx inserted first (uq deal_id / invoice_id). Idempotent return, no second outbox event.
			existing, loadErr := uc.payouts.LoadByDealIDForUpdate(ctx, cmd.DealID)
			if loadErr != nil {
				return nil, loadErr
			}
			return existing, nil
		}
		return nil, err
	}
	if uc.events != nil {
		if err := uc.events.Publish(ctx, inv.DealID, inv.SellerCompanyID, wallet.SellerPayoutCreated{
			PayoutID:        payout.ID,
			DealID:        payout.DealID,
			InvoiceID:     payout.InvoiceID,
			AuctionID:     payout.AuctionID,
			SellerCompanyID: payout.SellerCompanyID,
			BuyerCompanyID:  payout.BuyerCompanyID,
			Amount:        payout.Amount,
			Currency:      payout.Currency,
			CreatedAt:     payout.CreatedAt,
		}); err != nil {
			return nil, err
		}
	}
	return payout, nil
}
