package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type TopUpRepository struct {
	db *sql.DB
}

func NewTopUpRepository(db *sql.DB) *TopUpRepository {
	return &TopUpRepository{db: db}
}

var _ billingapp.TopUpRepository = (*TopUpRepository)(nil)

func (r *TopUpRepository) Create(ctx context.Context, tu *wallet.TopUp) error {
	const q = `
INSERT INTO billing_top_ups (
	id, company_id, account_id, amount, currency, status, provider,
	provider_payment_id, confirmation_url, created_at, confirmed_at, failed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, q,
		tu.ID,
		tu.CompanyID,
		tu.AccountID,
		tu.Amount,
		string(tu.Currency),
		string(tu.Status),
		tu.Provider,
		tu.ProviderPaymentID,
		tu.ConfirmationURL,
		tu.CreatedAt,
		nullTime(tu.ConfirmedAt),
		nullTime(tu.FailedAt),
	)
	return err
}

func (r *TopUpRepository) Save(ctx context.Context, tu *wallet.TopUp) error {
	const q = `
UPDATE billing_top_ups SET
	status = $2,
	provider_payment_id = $3,
	confirmation_url = $4,
	confirmed_at = $5,
	failed_at = $6
WHERE id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	res, err := dbtx.ExecContext(ctx, q,
		tu.ID,
		string(tu.Status),
		tu.ProviderPaymentID,
		tu.ConfirmationURL,
		nullTime(tu.ConfirmedAt),
		nullTime(tu.FailedAt),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return billingapp.ErrTopUpNotFound
	}
	return nil
}

func (r *TopUpRepository) Load(ctx context.Context, id string) (*wallet.TopUp, error) {
	return r.load(ctx, id, false)
}

func (r *TopUpRepository) LoadForUpdate(ctx context.Context, id string) (*wallet.TopUp, error) {
	return r.load(ctx, id, true)
}

func (r *TopUpRepository) load(ctx context.Context, id string, forUpdate bool) (*wallet.TopUp, error) {
	q := `
SELECT id, company_id, account_id, amount, currency, status, provider,
	provider_payment_id, confirmation_url, created_at, confirmed_at, failed_at
FROM billing_top_ups
WHERE id = $1
`
	if forUpdate {
		q += " FOR UPDATE"
	}
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, id)
	tu, err := scanTopUp(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, billingapp.ErrTopUpNotFound
		}
		return nil, err
	}
	return tu, nil
}

func (r *TopUpRepository) LoadByProviderPayment(ctx context.Context, provider, providerPaymentID string) (*wallet.TopUp, error) {
	return r.loadByProviderPayment(ctx, provider, providerPaymentID, false)
}

func (r *TopUpRepository) LoadByProviderPaymentForUpdate(ctx context.Context, provider, providerPaymentID string) (*wallet.TopUp, error) {
	return r.loadByProviderPayment(ctx, provider, providerPaymentID, true)
}

func (r *TopUpRepository) loadByProviderPayment(ctx context.Context, provider, providerPaymentID string, forUpdate bool) (*wallet.TopUp, error) {
	q := `
SELECT id, company_id, account_id, amount, currency, status, provider,
	provider_payment_id, confirmation_url, created_at, confirmed_at, failed_at
FROM billing_top_ups
WHERE provider = $1 AND provider_payment_id = $2
`
	if forUpdate {
		q += " FOR UPDATE"
	}
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, provider, providerPaymentID)
	tu, err := scanTopUp(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, billingapp.ErrTopUpNotFound
		}
		return nil, err
	}
	return tu, nil
}

func (r *TopUpRepository) ListByCompany(ctx context.Context, companyID string, limit int) ([]*wallet.TopUp, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
SELECT id, company_id, account_id, amount, currency, status, provider,
	provider_payment_id, confirmation_url, created_at, confirmed_at, failed_at
FROM billing_top_ups
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
	var out []*wallet.TopUp
	for rows.Next() {
		tu, err := scanTopUp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, tu)
	}
	return out, rows.Err()
}

func scanTopUp(sc interface {
	Scan(dest ...any) error
}) (*wallet.TopUp, error) {
	var (
		tu                    wallet.TopUp
		cur                   string
		st                    string
		confirmedAt, failedAt sql.NullTime
	)
	if err := sc.Scan(
		&tu.ID,
		&tu.CompanyID,
		&tu.AccountID,
		&tu.Amount,
		&cur,
		&st,
		&tu.Provider,
		&tu.ProviderPaymentID,
		&tu.ConfirmationURL,
		&tu.CreatedAt,
		&confirmedAt,
		&failedAt,
	); err != nil {
		return nil, err
	}
	tu.Currency = wallet.Currency(cur)
	tu.Status = wallet.TopUpStatus(st)
	if confirmedAt.Valid {
		t := confirmedAt.Time
		tu.ConfirmedAt = &t
	}
	if failedAt.Valid {
		t := failedAt.Time
		tu.FailedAt = &t
	}
	return &tu, nil
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
