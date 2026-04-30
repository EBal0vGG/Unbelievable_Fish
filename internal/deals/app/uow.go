package app

import "context"

type simpleTx struct {
	deals         DealRepository
	confirmations DealConfirmationRepository
	projections   ProjectionRepository
	selections    WinnerSelectionRepository
	outbox        OutboxRepository
}

func (t *simpleTx) Deals() DealRepository                     { return t.deals }
func (t *simpleTx) Confirmations() DealConfirmationRepository { return t.confirmations }
func (t *simpleTx) Projections() ProjectionRepository         { return t.projections }
func (t *simpleTx) Selections() WinnerSelectionRepository     { return t.selections }
func (t *simpleTx) Outbox() OutboxRepository                  { return t.outbox }

type SimpleUnitOfWork struct {
	tx *simpleTx
}

func NewSimpleUnitOfWork(
	deals DealRepository,
	confirmations DealConfirmationRepository,
	projections ProjectionRepository,
	selections WinnerSelectionRepository,
	outbox OutboxRepository,
) *SimpleUnitOfWork {
	return &SimpleUnitOfWork{
		tx: &simpleTx{
			deals:         deals,
			confirmations: confirmations,
			projections:   projections,
			selections:    selections,
			outbox:        outbox,
		},
	}
}

func (u *SimpleUnitOfWork) Do(ctx context.Context, fn func(Tx) error) error {
	return fn(u.tx)
}
