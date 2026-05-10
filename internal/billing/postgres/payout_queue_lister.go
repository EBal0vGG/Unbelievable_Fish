package postgres

import (
	"context"
	"database/sql"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
)

type PayoutQueueLister struct {
	db *sql.DB
}

func NewPayoutQueueLister(db *sql.DB) *PayoutQueueLister {
	return &PayoutQueueLister{db: db}
}

var _ billingapp.PayoutQueueLister = (*PayoutQueueLister)(nil)

func (l *PayoutQueueLister) ListPayoutQueue(ctx context.Context, limit int) ([]billingapp.PayoutQueueRow, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	const q = `
SELECT
	sp.id,
	sp.deal_id,
	sp.invoice_id,
	sp.auction_id,
	sp.seller_company_id,
	sp.buyer_company_id,
	COALESCE(sc.name, '') AS seller_company_name,
	COALESCE(bc.name, '') AS buyer_company_name,
	sp.amount,
	sp.currency,
	sp.status,
	COALESCE(di.status, '') AS invoice_status,
	sp.created_at,
	sp.ready_at,
	sp.paid_at,
	sp.failed_at,
	sp.cancelled_at
FROM billing_seller_payouts sp
LEFT JOIN identity_companies sc ON sc.company_id = sp.seller_company_id
LEFT JOIN identity_companies bc ON bc.company_id = sp.buyer_company_id
LEFT JOIN billing_deal_invoices di ON di.id = sp.invoice_id
ORDER BY sp.created_at DESC
LIMIT $1
`
	dbtx := DBTXFromContext(ctx, l.db)
	rows, err := dbtx.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []billingapp.PayoutQueueRow
	for rows.Next() {
		var r billingapp.PayoutQueueRow
		var readyAt, paidAt, failedAt, cancelledAt sql.NullTime
		if err := rows.Scan(
			&r.PayoutID,
			&r.DealID,
			&r.InvoiceID,
			&r.AuctionID,
			&r.SellerCompanyID,
			&r.BuyerCompanyID,
			&r.SellerCompanyName,
			&r.BuyerCompanyName,
			&r.Amount,
			&r.Currency,
			&r.Status,
			&r.InvoiceStatus,
			&r.CreatedAt,
			&readyAt,
			&paidAt,
			&failedAt,
			&cancelledAt,
		); err != nil {
			return nil, err
		}
		if readyAt.Valid {
			t := readyAt.Time
			r.ReadyAt = &t
		}
		if paidAt.Valid {
			t := paidAt.Time
			r.PaidAt = &t
		}
		if failedAt.Valid {
			t := failedAt.Time
			r.FailedAt = &t
		}
		if cancelledAt.Valid {
			t := cancelledAt.Time
			r.CancelledAt = &t
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []billingapp.PayoutQueueRow{}
	}
	return out, nil
}
