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

// PendingDealInvoiceAdminRow is a slim read model for operator tooling (GET /admin/invoices/pending).
type PendingDealInvoiceAdminRow struct {
	ID                string    `json:"id"`
	DealID            string    `json:"deal_id"`
	AuctionID         string    `json:"auction_id"`
	BuyerCompanyID    string    `json:"buyer_company_id"`
	SellerCompanyID   string    `json:"seller_company_id"`
	Status            string    `json:"status"`
	GoodsAmount       int64     `json:"goods_amount"`
	TotalAmount       int64     `json:"total_amount"`
	Currency          string    `json:"currency"`
	DueAt             time.Time `json:"due_at"`
	CreatedAt         time.Time `json:"created_at"`
	Provider          string    `json:"provider"`
	ProviderInvoiceID string    `json:"provider_invoice_id,omitempty"`
}

// ListPaymentPendingAdmin returns PAYMENT_PENDING invoices (newest first) for admin UI.
func (l *DealInvoiceLister) ListPaymentPendingAdmin(ctx context.Context, limit int) ([]PendingDealInvoiceAdminRow, error) {
	if limit <= 0 {
		limit = 200
	}
	const q = `
SELECT id, deal_id, auction_id, buyer_company_id, seller_company_id, status,
       goods_amount, total_amount, currency, due_at, created_at, provider, provider_invoice_id
FROM billing_deal_invoices
WHERE status = 'PAYMENT_PENDING'
ORDER BY created_at DESC
LIMIT $1
`
	dbtx := DBTXFromContext(ctx, l.db)
	rows, err := dbtx.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingDealInvoiceAdminRow
	for rows.Next() {
		var r PendingDealInvoiceAdminRow
		if err := rows.Scan(
			&r.ID, &r.DealID, &r.AuctionID, &r.BuyerCompanyID, &r.SellerCompanyID, &r.Status,
			&r.GoodsAmount, &r.TotalAmount, &r.Currency, &r.DueAt, &r.CreatedAt, &r.Provider, &r.ProviderInvoiceID,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
