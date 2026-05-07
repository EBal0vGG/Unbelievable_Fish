package app

import (
	"context"
	"errors"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type ReserveAuctionDeposit struct {
	accounts  AccountRepository
	deposits  AuctionDepositRepository
	ledger    LedgerRepository
	ids       IDGenerator
	clock     Clock
}

func NewReserveAuctionDeposit(accounts AccountRepository, deposits AuctionDepositRepository, ledger LedgerRepository, ids IDGenerator, clock Clock) (*ReserveAuctionDeposit, error) {
	if accounts == nil || deposits == nil || ledger == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &ReserveAuctionDeposit{
		accounts: accounts,
		deposits: deposits,
		ledger:   ledger,
		ids:      ids,
		clock:    clock,
	}, nil
}

func (uc *ReserveAuctionDeposit) Execute(
	ctx context.Context,
	companyID string,
	auctionID string,
	startPrice int64,
	currency wallet.Currency,
) error {
	if isBlank(companyID) || isBlank(auctionID) {
		return wallet.ErrInvalidIdentifier
	}
	if currency != wallet.CurrencyRUB {
		return wallet.ErrUnsupportedCurrency
	}
	now := uc.clock.Now().UTC()

	acc, err := uc.accounts.LoadByCompanyForUpdate(ctx, companyID)
	if err != nil {
		return err
	}
	if acc.Currency() != currency {
		return wallet.ErrUnsupportedCurrency
	}

	existing, err := uc.deposits.Find(ctx, auctionID, companyID)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.Status == wallet.DepositHeld {
			return nil
		}
		return ErrDepositNotHeld
	}

	amount := depositFromStartPrice(startPrice)
	if amount <= 0 {
		return wallet.ErrInvalidAmount
	}
	if err := acc.Reserve(amount); err != nil {
		if errors.Is(err, wallet.ErrInsufficientFunds) {
			return ErrInsufficientFundsForDeposit
		}
		return err
	}

	dep, err := wallet.NewAuctionDeposit(auctionID, companyID, acc.ID(), amount, currency, now)
	if err != nil {
		return err
	}
	if err := uc.deposits.Create(ctx, dep); err != nil {
		return err
	}
	if err := uc.accounts.Save(ctx, acc); err != nil {
		return err
	}

	ref := auctionDepositLedgerRef(auctionID, companyID)
	exists, err := uc.ledger.ExistsByReference(ctx, companyID, "auction_deposit", ref, wallet.LedgerBidDepositReserved)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	entry := wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      currency,
		Amount:        amount,
		EntryType:     wallet.LedgerBidDepositReserved,
		ReferenceType: "auction_deposit",
		ReferenceID:   ref,
		CreatedAt:     now,
	}
	return uc.ledger.Append(ctx, entry)
}
