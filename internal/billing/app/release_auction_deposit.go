package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type ReleaseAuctionDeposit struct {
	accounts AccountRepository
	deposits AuctionDepositRepository
	ledger   LedgerRepository
	ids      IDGenerator
	clock    Clock
	events   DomainEventPublisher
}

func NewReleaseAuctionDeposit(accounts AccountRepository, deposits AuctionDepositRepository, ledger LedgerRepository, ids IDGenerator, clock Clock, events DomainEventPublisher) (*ReleaseAuctionDeposit, error) {
	if accounts == nil || deposits == nil || ledger == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &ReleaseAuctionDeposit{accounts: accounts, deposits: deposits, ledger: ledger, ids: ids, clock: clock, events: events}, nil
}

func (uc *ReleaseAuctionDeposit) Execute(ctx context.Context, companyID, auctionID, reason string) error {
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
	if dep.Status == wallet.DepositReleased {
		return nil
	}
	if dep.Status != wallet.DepositHeld {
		return ErrDepositNotHeld
	}
	exists, err := uc.ledger.ExistsByReference(ctx, companyID, "auction_deposit_release", refLedger, wallet.LedgerBidDepositReleased)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	if err := acc.Release(dep.Amount); err != nil {
		return err
	}
	dep.MarkReleased(now)
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
		EntryType:     wallet.LedgerBidDepositReleased,
		ReferenceType: "auction_deposit_release",
		ReferenceID:   refLedger,
		Reason:        reason,
		CreatedAt:     now,
	}
	if err := uc.ledger.Append(ctx, entry); err != nil {
		return err
	}
	if uc.events != nil {
		return uc.events.Publish(ctx, dep.AuctionID, companyID, wallet.AuctionDepositReleased{
			AuctionID: dep.AuctionID,
			CompanyID: companyID,
			Amount:    dep.Amount,
			Currency:  dep.Currency,
			Reason:    reason,
		})
	}
	return nil
}
