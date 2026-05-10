package app

import (
	"context"
	"time"
)

// PayoutQueueRow is a read model for admin payout queue (SQL join in postgres; no identity import).
type PayoutQueueRow struct {
	PayoutID          string     `json:"payout_id"`
	DealID            string     `json:"deal_id"`
	InvoiceID         string     `json:"invoice_id"`
	AuctionID         string     `json:"auction_id"`
	SellerCompanyID   string     `json:"seller_company_id"`
	BuyerCompanyID    string     `json:"buyer_company_id"`
	SellerCompanyName string     `json:"seller_company_name"`
	BuyerCompanyName  string     `json:"buyer_company_name"`
	Amount            int64      `json:"amount"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	InvoiceStatus     string     `json:"invoice_status"`
	CreatedAt         time.Time  `json:"created_at"`
	ReadyAt           *time.Time `json:"ready_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	FailedAt          *time.Time `json:"failed_at,omitempty"`
	CancelledAt       *time.Time `json:"cancelled_at,omitempty"`
}

// PayoutQueueLister returns the global payout queue for platform operators.
type PayoutQueueLister interface {
	ListPayoutQueue(ctx context.Context, limit int) ([]PayoutQueueRow, error)
}
