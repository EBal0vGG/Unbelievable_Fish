package postgres

import (
	"context"
	"database/sql"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type DealInvoiceRepository struct {
	db *sql.DB
}

func NewDealInvoiceRepository(db *sql.DB) *DealInvoiceRepository {
	return &DealInvoiceRepository{db: db}
}

var _ billingapp.DealInvoiceRepository = (*DealInvoiceRepository)(nil)

func (r *DealInvoiceRepository) Create(ctx context.Context, inv *wallet.DealInvoice) error {
	const q = `
INSERT INTO billing_deal_invoices (
	id, deal_id, auction_id, buyer_company_id, seller_company_id,
	goods_amount, platform_fee_due_amount, total_amount, currency, status,
	provider, provider_invoice_id, payment_url, due_at, created_at,
	paid_at, expired_at, cancelled_at, failed_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, q,
		inv.ID, inv.DealID, inv.AuctionID, inv.BuyerCompanyID, inv.SellerCompanyID,
		inv.GoodsAmount, inv.PlatformFeeDueAmount, inv.TotalAmount, string(inv.Currency), string(inv.Status),
		inv.Provider, inv.ProviderInvoiceID, inv.PaymentURL, inv.DueAt, inv.CreatedAt,
		nullTime(inv.PaidAt), nullTime(inv.ExpiredAt), nullTime(inv.CancelledAt), nullTime(inv.FailedAt),
	)
	return err
}

func (r *DealInvoiceRepository) Save(ctx context.Context, inv *wallet.DealInvoice) error {
	const q = `
UPDATE billing_deal_invoices SET
	auction_id = $2,
	buyer_company_id = $3,
	seller_company_id = $4,
	goods_amount = $5,
	platform_fee_due_amount = $6,
	total_amount = $7,
	currency = $8,
	status = $9,
	provider = $10,
	provider_invoice_id = $11,
	payment_url = $12,
	due_at = $13,
	paid_at = $14,
	expired_at = $15,
	cancelled_at = $16,
	failed_at = $17
WHERE id = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	res, err := dbtx.ExecContext(ctx, q,
		inv.ID,
		inv.AuctionID, inv.BuyerCompanyID, inv.SellerCompanyID,
		inv.GoodsAmount, inv.PlatformFeeDueAmount, inv.TotalAmount, string(inv.Currency), string(inv.Status),
		inv.Provider, inv.ProviderInvoiceID, inv.PaymentURL, inv.DueAt,
		nullTime(inv.PaidAt), nullTime(inv.ExpiredAt), nullTime(inv.CancelledAt), nullTime(inv.FailedAt),
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return billingapp.ErrDealInvoiceNotFound
	}
	return nil
}

func (r *DealInvoiceRepository) LoadByDealID(ctx context.Context, dealID string) (*wallet.DealInvoice, error) {
	return r.loadByDealID(ctx, dealID, false)
}

func (r *DealInvoiceRepository) LoadByDealIDForUpdate(ctx context.Context, dealID string) (*wallet.DealInvoice, error) {
	return r.loadByDealID(ctx, dealID, true)
}

func (r *DealInvoiceRepository) loadByDealID(ctx context.Context, dealID string, forUpdate bool) (*wallet.DealInvoice, error) {
	q := `
SELECT id, deal_id, auction_id, buyer_company_id, seller_company_id,
	goods_amount, platform_fee_due_amount, total_amount, currency, status,
	provider, provider_invoice_id, payment_url, due_at, created_at,
	paid_at, expired_at, cancelled_at, failed_at
FROM billing_deal_invoices
WHERE deal_id = $1
`
	if forUpdate {
		q += " FOR UPDATE"
	}
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, dealID)
	return scanDealInvoice(row)
}

func (r *DealInvoiceRepository) LoadByID(ctx context.Context, id string) (*wallet.DealInvoice, error) {
	return r.loadByID(ctx, id, false)
}

func (r *DealInvoiceRepository) LoadByIDForUpdate(ctx context.Context, id string) (*wallet.DealInvoice, error) {
	return r.loadByID(ctx, id, true)
}

func (r *DealInvoiceRepository) loadByID(ctx context.Context, id string, forUpdate bool) (*wallet.DealInvoice, error) {
	q := `
SELECT id, deal_id, auction_id, buyer_company_id, seller_company_id,
	goods_amount, platform_fee_due_amount, total_amount, currency, status,
	provider, provider_invoice_id, payment_url, due_at, created_at,
	paid_at, expired_at, cancelled_at, failed_at
FROM billing_deal_invoices
WHERE id = $1
`
	if forUpdate {
		q += " FOR UPDATE"
	}
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, q, id)
	return scanDealInvoice(row)
}

func (r *DealInvoiceRepository) ListByBuyerCompany(ctx context.Context, buyerCompanyID string, limit int) ([]*wallet.DealInvoice, error) {
	if limit <= 0 {
		limit = 50
	}
	const q = `
SELECT id, deal_id, auction_id, buyer_company_id, seller_company_id,
	goods_amount, platform_fee_due_amount, total_amount, currency, status,
	provider, provider_invoice_id, payment_url, due_at, created_at,
	paid_at, expired_at, cancelled_at, failed_at
FROM billing_deal_invoices
WHERE buyer_company_id = $1
ORDER BY created_at DESC
LIMIT $2
`
	dbtx := DBTXFromContext(ctx, r.db)
	rows, err := dbtx.QueryContext(ctx, q, buyerCompanyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*wallet.DealInvoice
	for rows.Next() {
		inv, err := scanDealInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

type dealInvoiceRow interface {
	Scan(dest ...any) error
}

func scanDealInvoice(row dealInvoiceRow) (*wallet.DealInvoice, error) {
	var (
		inv                                      wallet.DealInvoice
		auctionID                                string
		currencyStr, statusStr                   string
		paidAt, expiredAt, cancelledAt, failedAt sql.NullTime
	)
	if err := row.Scan(
		&inv.ID, &inv.DealID, &auctionID, &inv.BuyerCompanyID, &inv.SellerCompanyID,
		&inv.GoodsAmount, &inv.PlatformFeeDueAmount, &inv.TotalAmount, &currencyStr, &statusStr,
		&inv.Provider, &inv.ProviderInvoiceID, &inv.PaymentURL, &inv.DueAt, &inv.CreatedAt,
		&paidAt, &expiredAt, &cancelledAt, &failedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, billingapp.ErrDealInvoiceNotFound
		}
		return nil, err
	}
	inv.AuctionID = auctionID
	inv.Currency = wallet.Currency(currencyStr)
	inv.Status = wallet.InvoiceStatus(statusStr)
	if paidAt.Valid {
		t := paidAt.Time
		inv.PaidAt = &t
	}
	if expiredAt.Valid {
		t := expiredAt.Time
		inv.ExpiredAt = &t
	}
	if cancelledAt.Valid {
		t := cancelledAt.Time
		inv.CancelledAt = &t
	}
	if failedAt.Valid {
		t := failedAt.Time
		inv.FailedAt = &t
	}
	return &inv, nil
}
