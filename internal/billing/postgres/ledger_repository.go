package postgres

import (
	"context"
	"database/sql"
	"errors"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type LedgerRepository struct {
	db *sql.DB
}

func NewLedgerRepository(db *sql.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

var _ billingapp.LedgerRepository = (*LedgerRepository)(nil)

func (r *LedgerRepository) Append(ctx context.Context, entry wallet.LedgerEntry) error {
	const q = `
INSERT INTO billing_ledger_entries (id, account_id, company_id, type, amount, currency, reference_type, reference_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, q,
		entry.ID,
		entry.AccountID,
		entry.CompanyID,
		string(entry.EntryType),
		entry.Amount,
		string(entry.Currency),
		entry.ReferenceType,
		entry.ReferenceID,
		entry.CreatedAt.UTC(),
	)
	return err
}

func (r *LedgerRepository) ExistsByReference(ctx context.Context, companyID, referenceType, referenceID string, typ wallet.LedgerEntryType) (bool, error) {
	const q = `
SELECT 1 FROM billing_ledger_entries
WHERE company_id = $1 AND reference_type = $2 AND reference_id = $3 AND type = $4
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, companyID, referenceType, referenceID, string(typ))
	var one int
	if err := row.Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type LedgerLister struct {
	db *sql.DB
}

func NewLedgerLister(db *sql.DB) *LedgerLister {
	return &LedgerLister{db: db}
}

var _ billingapp.LedgerQuery = (*LedgerLister)(nil)

func (r *LedgerLister) ListByCompany(ctx context.Context, companyID string, limit int) ([]wallet.LedgerEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
SELECT id, account_id, company_id, type, amount, currency, reference_type, reference_id, created_at
FROM billing_ledger_entries
WHERE company_id = $1
ORDER BY created_at DESC
LIMIT $2
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, q, companyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []wallet.LedgerEntry
	for rows.Next() {
		var e wallet.LedgerEntry
		var typ, cur string
		if err := rows.Scan(&e.ID, &e.AccountID, &e.CompanyID, &typ, &e.Amount, &cur, &e.ReferenceType, &e.ReferenceID, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.EntryType = wallet.LedgerEntryType(typ)
		e.Currency = wallet.Currency(cur)
		out = append(out, e)
	}
	return out, rows.Err()
}
