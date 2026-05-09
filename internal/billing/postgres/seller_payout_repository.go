package postgres

import (
	"context"
	"database/sql"
	"errors"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type SellerPayoutRepository struct {
	db *sql.DB
}

func NewSellerPayoutRepository(db *sql.DB) *SellerPayoutRepository {
	return &SellerPayoutRepository{db: db}
}

var _ billingapp.SellerPayoutRepository = (*SellerPayoutRepository)(nil)

func (r *SellerPayoutRepository) Create(ctx context.Context, p *wallet.SellerPayout) error {
	const q = `
INSERT INTO billing_seller_payouts (
	id, deal_id, invoice_id, auction_id, seller_company_id, buyer_company_id,
	amount, currency, status, created_at,
	ready_at, paid_at, cancelled_at, failed_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, q,
		p.ID, p.DealID, p.InvoiceID, p.AuctionID, p.SellerCompanyID, p.BuyerCompanyID,
		p.Amount, string(p.Currency), string(p.Status), p.CreatedAt,
		nullTime(p.ReadyAt), nullTime(p.PaidAt), nullTime(p.CancelledAt), nullTime(p.FailedAt),
	)
	return err
}

func (r *SellerPayoutRepository) Save(ctx context.Context, p *wallet.SellerPayout) error {
	// Immutable snapshot: deal_id, invoice_id, auction_id, parties, amount, currency, created_at must not change here.
	const q = `
UPDATE billing_seller_payouts SET
	status = $2,
	ready_at = $3,
	paid_at = $4,
	cancelled_at = $5,
	failed_at = $6
WHERE id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	res, err := dbtx.ExecContext(ctx, q,
		p.ID,
		string(p.Status),
		nullTime(p.ReadyAt), nullTime(p.PaidAt), nullTime(p.CancelledAt), nullTime(p.FailedAt),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return billingapp.ErrSellerPayoutNotFound
	}
	return nil
}

func (r *SellerPayoutRepository) LoadByID(ctx context.Context, id string) (*wallet.SellerPayout, error) {
	return r.loadByID(ctx, id, false)
}

func (r *SellerPayoutRepository) LoadByDealID(ctx context.Context, dealID string) (*wallet.SellerPayout, error) {
	return r.loadByDealID(ctx, dealID, false)
}

func (r *SellerPayoutRepository) LoadByDealIDForUpdate(ctx context.Context, dealID string) (*wallet.SellerPayout, error) {
	return r.loadByDealID(ctx, dealID, true)
}

func (r *SellerPayoutRepository) loadByDealID(ctx context.Context, dealID string, forUpdate bool) (*wallet.SellerPayout, error) {
	q := `
SELECT id, deal_id, invoice_id, auction_id, seller_company_id, buyer_company_id,
	amount, currency, status, created_at,
	ready_at, paid_at, cancelled_at, failed_at
FROM billing_seller_payouts
WHERE deal_id = $1
`
	if forUpdate {
		q += " FOR UPDATE"
	}
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, dealID)
	return scanSellerPayout(row)
}

func (r *SellerPayoutRepository) loadByID(ctx context.Context, id string, forUpdate bool) (*wallet.SellerPayout, error) {
	q := `
SELECT id, deal_id, invoice_id, auction_id, seller_company_id, buyer_company_id,
	amount, currency, status, created_at,
	ready_at, paid_at, cancelled_at, failed_at
FROM billing_seller_payouts
WHERE id = $1
`
	if forUpdate {
		q += " FOR UPDATE"
	}
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, id)
	return scanSellerPayout(row)
}

func (r *SellerPayoutRepository) ListBySellerCompany(ctx context.Context, sellerCompanyID string, limit int) ([]*wallet.SellerPayout, error) {
	if limit <= 0 {
		limit = 50
	}
	const q = `
SELECT id, deal_id, invoice_id, auction_id, seller_company_id, buyer_company_id,
	amount, currency, status, created_at,
	ready_at, paid_at, cancelled_at, failed_at
FROM billing_seller_payouts
WHERE seller_company_id = $1
ORDER BY created_at DESC
LIMIT $2
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, q, sellerCompanyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*wallet.SellerPayout
	for rows.Next() {
		p, err := scanSellerPayout(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

type sellerPayoutRow interface {
	Scan(dest ...any) error
}

func scanSellerPayout(row sellerPayoutRow) (*wallet.SellerPayout, error) {
	var (
		p                                        wallet.SellerPayout
		cur                                      string
		st                                       string
		readyAt, paidAt, cancelledAt, failedAt sql.NullTime
	)
	err := row.Scan(
		&p.ID, &p.DealID, &p.InvoiceID, &p.AuctionID, &p.SellerCompanyID, &p.BuyerCompanyID,
		&p.Amount, &cur, &st, &p.CreatedAt,
		&readyAt, &paidAt, &cancelledAt, &failedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, billingapp.ErrSellerPayoutNotFound
		}
		return nil, err
	}
	p.Currency = wallet.Currency(cur)
	p.Status = wallet.SellerPayoutStatus(st)
	if readyAt.Valid {
		t := readyAt.Time
		p.ReadyAt = &t
	}
	if paidAt.Valid {
		t := paidAt.Time
		p.PaidAt = &t
	}
	if cancelledAt.Valid {
		t := cancelledAt.Time
		p.CancelledAt = &t
	}
	if failedAt.Valid {
		t := failedAt.Time
		p.FailedAt = &t
	}
	return &p, nil
}
