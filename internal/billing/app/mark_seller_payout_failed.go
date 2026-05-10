package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

// MarkSellerPayoutFailed sets payout to FAILED from PENDING or READY (operator / demo).
type MarkSellerPayoutFailed struct {
	payouts SellerPayoutRepository
	clock   Clock
	events  DomainEventPublisher
}

func NewMarkSellerPayoutFailed(payouts SellerPayoutRepository, clock Clock, events DomainEventPublisher) (*MarkSellerPayoutFailed, error) {
	if payouts == nil {
		return nil, ErrNilDependency
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &MarkSellerPayoutFailed{payouts: payouts, clock: clock, events: events}, nil
}

func (uc *MarkSellerPayoutFailed) Execute(ctx context.Context, payoutID string) (*wallet.SellerPayout, error) {
	if isBlank(payoutID) {
		return nil, wallet.ErrInvalidIdentifier
	}
	p, err := uc.payouts.LoadByIDForUpdate(ctx, payoutID)
	if err != nil {
		return nil, err
	}
	if err := p.MarkFailed(uc.clock.Now()); err != nil {
		return nil, err
	}
	if err := uc.payouts.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
