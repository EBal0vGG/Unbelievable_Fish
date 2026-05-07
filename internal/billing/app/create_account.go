package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type CreateAccount struct {
	accounts AccountRepository
	ids      IDGenerator
	events   DomainEventPublisher
}

func NewCreateAccount(accounts AccountRepository, ids IDGenerator, events DomainEventPublisher) (*CreateAccount, error) {
	if accounts == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	return &CreateAccount{accounts: accounts, ids: ids, events: events}, nil
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
	if err := uc.accounts.Create(ctx, acc); err != nil {
		return err
	}
	if uc.events != nil {
		return uc.events.Publish(ctx, acc.ID(), companyID, wallet.AccountCreated{
			AccountID: acc.ID(),
			CompanyID: companyID,
			Currency:  acc.Currency(),
		})
	}
	return nil
}
