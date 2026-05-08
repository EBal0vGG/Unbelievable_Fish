package fake

import (
	"context"

	billingapp "github.com/EBal0vGG/Unbelievable_Fish/internal/billing/app"
)

const ProviderName = "fake"

// Provider is a dev-only payment provider that builds deterministic payment ids and return URLs.
type Provider struct{}

func (Provider) CreateTopUp(ctx context.Context, req billingapp.CreateTopUpRequest) (billingapp.CreateTopUpResponse, error) {
	_ = ctx
	pid := "fake-pay-" + req.TopUpID
	confirmationURL := "/billing/top-ups/" + req.TopUpID + "/fake-confirm"
	return billingapp.CreateTopUpResponse{
		ProviderPaymentID: pid,
		ConfirmationURL:   confirmationURL,
	}, nil
}
