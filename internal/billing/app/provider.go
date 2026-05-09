package app

import (
	"context"
	"time"
)

type PaymentProvider interface {
	CreateTopUp(ctx context.Context, req CreateTopUpRequest) (CreateTopUpResponse, error)
	CreateDealInvoice(ctx context.Context, req CreateDealInvoiceRequest) (CreateDealInvoiceResponse, error)
}

type CreateTopUpRequest struct {
	TopUpID   string
	CompanyID string
	Amount    int64
	Currency  string
	ReturnURL string
}

type CreateTopUpResponse struct {
	ProviderPaymentID string
	ConfirmationURL   string
}

type CreateDealInvoiceRequest struct {
	InvoiceID       string
	DealID          string
	BuyerCompanyID  string
	SellerCompanyID string
	Amount          int64
	Currency        string
	DueAt           time.Time
	ReturnURL       string
}

type CreateDealInvoiceResponse struct {
	ProviderInvoiceID string
	PaymentURL        string
}
