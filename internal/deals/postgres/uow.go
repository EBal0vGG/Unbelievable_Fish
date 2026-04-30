package postgres

import (
	"context"
	"database/sql"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
)

type UnitOfWork struct {
	db            *sql.DB
	tx            *TransactionManager
	deals         *DealRepository
	confirmations *DealConfirmationRepository
	projections   *ProjectionRepository
	selections    *SelectionRepository
	outbox        *OutboxRepository
}

func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{
		db:            db,
		tx:            NewTransactionManager(db, nil),
		deals:         NewDealRepository(db),
		confirmations: NewDealConfirmationRepository(db),
		projections:   NewProjectionRepository(db),
		selections:    NewSelectionRepository(db),
		outbox:        NewOutboxRepository(db),
	}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(app.Tx) error) error {
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		return fn(&tx{
			deals:         u.deals,
			confirmations: u.confirmations,
			projections:   u.projections,
			selections:    u.selections,
			outbox:        u.outbox,
		})
	})
}

type tx struct {
	deals         *DealRepository
	confirmations *DealConfirmationRepository
	projections   *ProjectionRepository
	selections    *SelectionRepository
	outbox        *OutboxRepository
}

func (t *tx) Deals() app.DealRepository                     { return t.deals }
func (t *tx) Confirmations() app.DealConfirmationRepository { return t.confirmations }
func (t *tx) Projections() app.ProjectionRepository         { return t.projections }
func (t *tx) Selections() app.WinnerSelectionRepository     { return t.selections }
func (t *tx) Outbox() app.OutboxRepository                  { return t.outbox }
