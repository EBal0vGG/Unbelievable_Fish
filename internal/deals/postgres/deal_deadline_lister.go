package postgres

import (
	"context"
	"database/sql"
	"time"
)

type DealDeadlineLister struct {
	db *sql.DB
}

func NewDealDeadlineLister(db *sql.DB) *DealDeadlineLister {
	return &DealDeadlineLister{db: db}
}

func (l *DealDeadlineLister) ListExpiredForFallback(ctx context.Context, now time.Time, limit int) ([]string, error) {
	const query = `
SELECT deal_id
FROM deals
WHERE status = 'pending'
  AND contract_signed_at IS NULL
  AND contract_sign_deadline IS NOT NULL
  AND contract_sign_deadline <= $1
ORDER BY created_at
LIMIT $2
`
	dbtx := DBTXFromContext(ctx, l.db)
	rows, err := dbtx.QueryContext(ctx, query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}
