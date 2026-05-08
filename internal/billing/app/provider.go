package app

import "context"

type PaymentProvider interface {
	CreateTopUp(ctx context.Context, req CreateTopUpRequest) (CreateTopUpResponse, error)
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
