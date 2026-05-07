package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type CaptureAuctionDeposit struct {
	accounts AccountRepository
	deposits AuctionDepositRepository
	ledger   LedgerRepository
	ids      IDGenerator
	clock    Clock
}

func NewCaptureAuctionDeposit(accounts AccountRepository, deposits AuctionDepositRepository, ledger LedgerRepository, ids IDGenerator, clock Clock) (*CaptureAuctionDeposit, error) {
	if accounts == nil || deposits == nil || ledger == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &CaptureAuctionDeposit{accounts: accounts, deposits: deposits, ledger: ledger, ids: ids, clock: clock}, nil
}

func (uc *CaptureAuctionDeposit) Execute(ctx context.Context, companyID, auctionID, reason string) error {
	if isBlank(companyID) || isBlank(auctionID) {
		return wallet.ErrInvalidIdentifier
	}
	now := uc.clock.Now().UTC()
	refLedger := auctionDepositLedgerRef(auctionID, companyID)

	acc, err := uc.accounts.LoadByCompanyForUpdate(ctx, companyID)
	if err != nil {
		return err
	}
	dep, err := uc.deposits.Find(ctx, auctionID, companyID)
	if err != nil {
		return err
	}
	if dep == nil {
		return ErrDepositNotFound
	}
	if dep.Status == wallet.DepositCaptured {
		return nil
	}
	if dep.Status != wallet.DepositHeld {
		return ErrDepositNotHeld
	}
	exists, err := uc.ledger.ExistsByReference(ctx, companyID, "auction_deposit_capture", refLedger, wallet.LedgerBidDepositCaptured)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if err := acc.Capture(dep.Amount); err != nil {
		return err
	}
	dep.MarkCaptured(now)
	if err := uc.deposits.Save(ctx, dep); err != nil {
		return err
	}
	if err := uc.accounts.Save(ctx, acc); err != nil {
		return err
	}
	entry := wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      dep.Currency,
		Amount:        dep.Amount,
		EntryType:     wallet.LedgerBidDepositCaptured,
		ReferenceType: "auction_deposit_capture",
		ReferenceID:   refLedger,
		CreatedAt:     now,
	}
	_ = reason
	return uc.ledger.Append(ctx, entry)
}
