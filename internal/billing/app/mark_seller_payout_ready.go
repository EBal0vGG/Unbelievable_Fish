package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

// MarkSellerPayoutReady transitions PENDING → READY (admin / ops).
type MarkSellerPayoutReady struct {
	payouts SellerPayoutRepository
	clock   Clock
	events  DomainEventPublisher
}

func NewMarkSellerPayoutReady(payouts SellerPayoutRepository, clock Clock, events DomainEventPublisher) (*MarkSellerPayoutReady, error) {
	if payouts == nil {
		return nil, ErrNilDependency
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &MarkSellerPayoutReady{payouts: payouts, clock: clock, events: events}, nil
}

// Execute must run inside the same database transaction as other billing writes (e.g. billing UnitOfWork.WithinTx)
// so that LoadByIDForUpdate row locks and Save publish atomically with sibling mutations.
func (uc *MarkSellerPayoutReady) Execute(ctx context.Context, payoutID string) (*wallet.SellerPayout, error) {
	if isBlank(payoutID) {
		return nil, wallet.ErrInvalidIdentifier
	}
	p, err := uc.payouts.LoadByIDForUpdate(ctx, payoutID)
	if err != nil {
		return nil, err
	}
	switch p.Status {
	case wallet.SellerPayoutReady, wallet.SellerPayoutPaid:
		return p, nil
	case wallet.SellerPayoutPending:
	default:
		return nil, wallet.ErrSellerPayoutWrongStatus
	}
	now := uc.clock.Now().UTC()
	if err := p.MarkReady(now); err != nil {
		return nil, err
	}
	if err := uc.payouts.Save(ctx, p); err != nil {
		return nil, err
	}
	if uc.events != nil {
		if err := uc.events.Publish(ctx, p.DealID, p.SellerCompanyID, wallet.SellerPayoutMarkedReady{
			PayoutID:        p.ID,
			DealID:          p.DealID,
			InvoiceID:       p.InvoiceID,
			SellerCompanyID: p.SellerCompanyID,
			Amount:          p.Amount,
			Currency:        p.Currency,
			ReadyAt:         now,
		}); err != nil {
			return nil, err
		}
	}
	return p, nil
}
