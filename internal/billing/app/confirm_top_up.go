package app

import (
	"context"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type ConfirmTopUp struct {
	accounts  AccountRepository
	ledger    LedgerRepository
	processed ProcessedTopUpRepository
	ids       IDGenerator
	clock     Clock
	events    DomainEventPublisher
}

func NewConfirmTopUp(accounts AccountRepository, ledger LedgerRepository, processed ProcessedTopUpRepository, ids IDGenerator, clock Clock, events DomainEventPublisher) (*ConfirmTopUp, error) {
	if accounts == nil || ledger == nil || processed == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &ConfirmTopUp{
		accounts:  accounts,
		ledger:    ledger,
		processed: processed,
		ids:       ids,
		clock:     clock,
		events:    events,
	}, nil
}

func (uc *ConfirmTopUp) Execute(ctx context.Context, companyID string, amount int64, externalPaymentID string) error {
	if isBlank(companyID) || isBlank(externalPaymentID) {
		return wallet.ErrInvalidIdentifier
	}
	if amount <= 0 {
		return wallet.ErrInvalidAmount
	}
	acc, err := uc.accounts.LoadByCompanyForUpdate(ctx, companyID)
	if err != nil {
		return err
	}
	inserted, err := uc.processed.InsertIfNew(ctx, externalPaymentID, companyID, acc.ID(), amount)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}
	if err := acc.Deposit(amount); err != nil {
		return err
	}
	if err := uc.accounts.Save(ctx, acc); err != nil {
		return err
	}
	now := uc.clock.Now().UTC()
	entry := wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      wallet.CurrencyRUB,
		Amount:        amount,
		EntryType:     wallet.LedgerTopUpConfirmed,
		ReferenceType: "external_payment",
		ReferenceID:   externalPaymentID,
		Reason:        "TOP_UP",
		CreatedAt:     now,
	}
	if err := uc.ledger.Append(ctx, entry); err != nil {
		return err
	}
	if uc.events != nil {
		return uc.events.Publish(ctx, acc.ID(), companyID, wallet.BalanceToppedUp{
			AccountID: acc.ID(),
			CompanyID: companyID,
			Amount:    amount,
			Currency:  wallet.CurrencyRUB,
		})
	}
	return nil
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }
