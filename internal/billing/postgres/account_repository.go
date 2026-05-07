package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

var _ billingapp.AccountRepository = (*AccountRepository)(nil)

func (r *AccountRepository) Create(ctx context.Context, account *wallet.Account) error {
	const q = `
INSERT INTO billing_accounts (id, company_id, currency, available_amount, held_amount, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, q,
		account.ID(),
		account.CompanyID(),
		string(account.Currency()),
		account.Available(),
		account.Held(),
	)
	return err
}

func (r *AccountRepository) ExistsByCompany(ctx context.Context, companyID string) (bool, error) {
	const q = `SELECT 1 FROM billing_accounts WHERE company_id = $1`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, companyID)
	var one int
	if err := row.Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *AccountRepository) LoadByCompany(ctx context.Context, companyID string) (*wallet.Account, error) {
	return r.loadByCompany(ctx, companyID, false)
}

func (r *AccountRepository) LoadByCompanyForUpdate(ctx context.Context, companyID string) (*wallet.Account, error) {
	return r.loadByCompany(ctx, companyID, true)
}

func (r *AccountRepository) loadByCompany(ctx context.Context, companyID string, forUpdate bool) (*wallet.Account, error) {
	q := `
SELECT id, company_id, currency, available_amount, held_amount
FROM billing_accounts
WHERE company_id = $1
`
	if forUpdate {
		q += " FOR UPDATE"
	}
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, companyID)
	var (
		id, cid, cur string
		avail, held   int64
	)
	if err := row.Scan(&id, &cid, &cur, &avail, &held); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, billingapp.ErrAccountNotFound
		}
		return nil, err
	}
	return wallet.RehydrateAccount(id, cid, wallet.Currency(cur), avail, held)
}

func (r *AccountRepository) Save(ctx context.Context, account *wallet.Account) error {
	const q = `
UPDATE billing_accounts
SET available_amount = $2, held_amount = $3, updated_at = $4
WHERE id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	res, err := dbtx.ExecContext(ctx, q, account.ID(), account.Available(), account.Held(), time.Now().UTC())
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return billingapp.ErrAccountNotFound
	}
	return nil
}
