package postgres

import (
	"context"
	"database/sql"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
)

type ProcessedTopUpRepository struct {
	db *sql.DB
}

func NewProcessedTopUpRepository(db *sql.DB) *ProcessedTopUpRepository {
	return &ProcessedTopUpRepository{db: db}
}

var _ billingapp.ProcessedTopUpRepository = (*ProcessedTopUpRepository)(nil)

func (r *ProcessedTopUpRepository) InsertIfNew(ctx context.Context, externalPaymentID, companyID, accountID string, amount int64) (bool, error) {
	const q = `
INSERT INTO billing_processed_top_ups (external_payment_id, company_id, account_id, amount)
VALUES ($1, $2, $3, $4)
ON CONFLICT (external_payment_id) DO NOTHING
`
	dbtx := DBTXFromContext(ctx, r.db)
	res, err := dbtx.ExecContext(ctx, q, externalPaymentID, companyID, accountID, amount)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
