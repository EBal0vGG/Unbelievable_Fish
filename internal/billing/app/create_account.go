package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type CreateAccount struct {
	accounts AccountRepository
	ids      IDGenerator
}

func NewCreateAccount(accounts AccountRepository, ids IDGenerator) (*CreateAccount, error) {
	if accounts == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	return &CreateAccount{accounts: accounts, ids: ids}, nil
}

func (uc *CreateAccount) Execute(ctx context.Context, companyID string) error {
	if isBlank(companyID) {
		return wallet.ErrInvalidIdentifier
	}
	ok, err := uc.accounts.ExistsByCompany(ctx, companyID)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	acc, err := wallet.NewAccount(uc.ids.NewID(), companyID, wallet.CurrencyRUB)
	if err != nil {
		return err
	}
	return uc.accounts.Create(ctx, acc)
}
