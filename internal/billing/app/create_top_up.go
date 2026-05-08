package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type CreateTopUp struct {
	createAccount *CreateAccount
	accounts      AccountRepository
	topUps        TopUpRepository
	provider      PaymentProvider
	providerName  string
	ids           IDGenerator
	clock         Clock
	returnURL     string
}

func NewCreateTopUp(createAccount *CreateAccount, accounts AccountRepository, topUps TopUpRepository, provider PaymentProvider, providerName string, ids IDGenerator, clock Clock, returnURL string) (*CreateTopUp, error) {
	if createAccount == nil || accounts == nil || topUps == nil || provider == nil {
		return nil, ErrNilDependency
	}
	if isBlank(providerName) {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &CreateTopUp{
		createAccount: createAccount,
		accounts:      accounts,
		topUps:        topUps,
		provider:      provider,
		providerName:  providerName,
		ids:           ids,
		clock:         clock,
		returnURL:     returnURL,
	}, nil
}

func (uc *CreateTopUp) Execute(ctx context.Context, companyID string, amount int64, currency wallet.Currency) (*wallet.TopUp, error) {
	if isBlank(companyID) {
		return nil, wallet.ErrInvalidIdentifier
	}
	if amount <= 0 {
		return nil, wallet.ErrInvalidAmount
	}
	if currency != wallet.CurrencyRUB {
		return nil, wallet.ErrUnsupportedCurrency
	}
	if err := uc.createAccount.Execute(ctx, companyID); err != nil {
		return nil, err
	}
	acc, err := uc.accounts.LoadByCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	topUpID := uc.ids.NewID()
	tu, err := wallet.NewTopUp(topUpID, companyID, acc.ID(), amount, currency, uc.providerName, uc.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := uc.topUps.Create(ctx, tu); err != nil {
		return nil, err
	}
	resp, err := uc.provider.CreateTopUp(ctx, CreateTopUpRequest{
		TopUpID:   topUpID,
		CompanyID: companyID,
		Amount:    amount,
		Currency:  string(currency),
		ReturnURL: uc.returnURL,
	})
	if err != nil {
		return nil, err
	}
	if err := tu.AttachProviderPayment(resp.ProviderPaymentID, resp.ConfirmationURL); err != nil {
		return nil, err
	}
	if err := uc.topUps.Save(ctx, tu); err != nil {
		return nil, err
	}
	return tu, nil
}
