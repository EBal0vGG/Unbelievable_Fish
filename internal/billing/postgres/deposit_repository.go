package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type AuctionDepositRepository struct {
	db *sql.DB
}

func NewAuctionDepositRepository(db *sql.DB) *AuctionDepositRepository {
	return &AuctionDepositRepository{db: db}
}

var _ billingapp.AuctionDepositRepository = (*AuctionDepositRepository)(nil)

func (r *AuctionDepositRepository) Find(ctx context.Context, auctionID, companyID string) (*wallet.AuctionDeposit, error) {
	const q = `
SELECT auction_id, company_id, account_id, amount, currency, status, created_at, released_at, captured_at
FROM billing_auction_deposits
WHERE auction_id = $1 AND company_id = $2
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, auctionID, companyID)
	var (
		aid, cid, accid, cur, st string
		amount                    int64
		createdAt                 time.Time
		releasedAt, capturedAt    sql.NullTime
	)
	if err := row.Scan(&aid, &cid, &accid, &amount, &cur, &st, &createdAt, &releasedAt, &capturedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d := &wallet.AuctionDeposit{
		AuctionID: aid,
		CompanyID: cid,
		AccountID: accid,
		Amount:    amount,
		Currency:  wallet.Currency(cur),
		Status:    wallet.DepositStatus(st),
		CreatedAt: createdAt.UTC(),
	}
	if releasedAt.Valid {
		t := releasedAt.Time.UTC()
		d.ReleasedAt = &t
	}
	if capturedAt.Valid {
		t := capturedAt.Time.UTC()
		d.CapturedAt = &t
	}
	return d, nil
}

func (r *AuctionDepositRepository) Create(ctx context.Context, deposit *wallet.AuctionDeposit) error {
	const q = `
INSERT INTO billing_auction_deposits (
    auction_id, company_id, account_id, amount, currency, status, created_at, released_at, captured_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, NULL, NULL)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, q,
		deposit.AuctionID,
		deposit.CompanyID,
		deposit.AccountID,
		deposit.Amount,
		string(deposit.Currency),
		string(deposit.Status),
		deposit.CreatedAt.UTC(),
	)
	return err
}

func (r *AuctionDepositRepository) Save(ctx context.Context, deposit *wallet.AuctionDeposit) error {
	const q = `
UPDATE billing_auction_deposits
SET status = $3, released_at = $4, captured_at = $5
WHERE auction_id = $1 AND company_id = $2
`
	dbtx := DBTXFromContext(ctx, r.db)
	var released, captured any
	if deposit.ReleasedAt != nil {
		released = deposit.ReleasedAt.UTC()
	}
	if deposit.CapturedAt != nil {
		captured = deposit.CapturedAt.UTC()
	}
	_, err := dbtx.ExecContext(ctx, q,
		deposit.AuctionID,
		deposit.CompanyID,
		string(deposit.Status),
		released,
		captured,
	)
	return err
}

func (r *AuctionDepositRepository) ListByAuction(ctx context.Context, auctionID string) ([]*wallet.AuctionDeposit, error) {
	const q = `
SELECT auction_id, company_id, account_id, amount, currency, status, created_at, released_at, captured_at
FROM billing_auction_deposits
WHERE auction_id = $1
ORDER BY company_id
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, q, auctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*wallet.AuctionDeposit
	for rows.Next() {
		var (
			aid, cid, accid, cur, st string
			amount                    int64
			createdAt                 time.Time
			releasedAt, capturedAt    sql.NullTime
		)
		if err := rows.Scan(&aid, &cid, &accid, &amount, &cur, &st, &createdAt, &releasedAt, &capturedAt); err != nil {
			return nil, err
		}
		d := &wallet.AuctionDeposit{
			AuctionID: aid,
			CompanyID: cid,
			AccountID: accid,
			Amount:    amount,
			Currency:  wallet.Currency(cur),
			Status:    wallet.DepositStatus(st),
			CreatedAt: createdAt.UTC(),
		}
		if releasedAt.Valid {
			t := releasedAt.Time.UTC()
			d.ReleasedAt = &t
		}
		if capturedAt.Valid {
			t := capturedAt.Time.UTC()
			d.CapturedAt = &t
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *AuctionDepositRepository) ListByCompany(ctx context.Context, companyID string, limit int) ([]*wallet.AuctionDeposit, error) {
	if limit <= 0 {
		limit = 100
	}
	const q = `
SELECT auction_id, company_id, account_id, amount, currency, status, created_at, released_at, captured_at
FROM billing_auction_deposits
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

	out := make([]*wallet.AuctionDeposit, 0, limit)
	for rows.Next() {
		var (
			aid, cid, accid, cur, st string
			amount                    int64
			createdAt                 time.Time
			releasedAt, capturedAt    sql.NullTime
		)
		if err := rows.Scan(&aid, &cid, &accid, &amount, &cur, &st, &createdAt, &releasedAt, &capturedAt); err != nil {
			return nil, err
		}
		d := &wallet.AuctionDeposit{
			AuctionID: aid,
			CompanyID: cid,
			AccountID: accid,
			Amount:    amount,
			Currency:  wallet.Currency(cur),
			Status:    wallet.DepositStatus(st),
			CreatedAt: createdAt.UTC(),
		}
		if releasedAt.Valid {
			t := releasedAt.Time.UTC()
			d.ReleasedAt = &t
		}
		if capturedAt.Valid {
			t := capturedAt.Time.UTC()
			d.CapturedAt = &t
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
