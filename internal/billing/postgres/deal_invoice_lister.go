package postgres

import (
	"context"
	"database/sql"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
)

type DealInvoiceLister struct {
	db *sql.DB
}

func NewDealInvoiceLister(db *sql.DB) *DealInvoiceLister {
	return &DealInvoiceLister{db: db}
}

var _ billingapp.ExpiredDealInvoiceLister = (*DealInvoiceLister)(nil)

func (l *DealInvoiceLister) ListExpired(ctx context.Context, now time.Time, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
SELECT id
FROM billing_deal_invoices
WHERE status = 'PAYMENT_PENDING'
  AND due_at <= $1
ORDER BY due_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED
`
	dbtx := DBTXFromContext(ctx, l.db)
	rows, err := dbtx.QueryContext(ctx, q, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
