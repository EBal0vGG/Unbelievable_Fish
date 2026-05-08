package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type ConfirmTopUpByProvider struct {
	topUps  TopUpRepository
	confirm *ConfirmTopUp
	clock   Clock
}

func NewConfirmTopUpByProvider(topUps TopUpRepository, confirm *ConfirmTopUp, clock Clock) (*ConfirmTopUpByProvider, error) {
	if topUps == nil || confirm == nil {
		return nil, ErrNilDependency
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &ConfirmTopUpByProvider{topUps: topUps, confirm: confirm, clock: clock}, nil
}

// Execute applies provider confirmation for a payment reference. Amount and currency must match the pending TopUp row.
func (uc *ConfirmTopUpByProvider) Execute(ctx context.Context, provider, providerPaymentID string, amount int64, currency wallet.Currency) error {
	if isBlank(provider) || isBlank(providerPaymentID) {
		return wallet.ErrInvalidIdentifier
	}
	tu, err := uc.topUps.LoadByProviderPaymentForUpdate(ctx, provider, providerPaymentID)
	if err != nil {
		return err
	}
	return uc.executeLoaded(ctx, tu, amount, currency)
}

// ExecuteByTopUpID locks the row by primary key and confirms using stored amount and currency (for dev fake-confirm route).
func (uc *ConfirmTopUpByProvider) ExecuteByTopUpID(ctx context.Context, topUpID string) error {
	if isBlank(topUpID) {
		return wallet.ErrInvalidIdentifier
	}
	tu, err := uc.topUps.LoadForUpdate(ctx, topUpID)
	if err != nil {
		return err
	}
	return uc.executeLoaded(ctx, tu, tu.Amount, tu.Currency)
}

func (uc *ConfirmTopUpByProvider) executeLoaded(ctx context.Context, tu *wallet.TopUp, amount int64, currency wallet.Currency) error {
	if tu.Status == wallet.TopUpSucceeded {
		return nil
	}
	if tu.Status != wallet.TopUpPending {
		return wallet.ErrInvalidTopUpStatus
	}
	if amount != tu.Amount || currency != tu.Currency {
		return ErrTopUpAmountMismatch
	}
	if err := uc.confirm.Execute(ctx, tu.CompanyID, amount, tu.ProviderPaymentID); err != nil {
		return err
	}
	if err := tu.MarkSucceeded(uc.clock.Now()); err != nil {
		return err
	}
	return uc.topUps.Save(ctx, tu)
}
