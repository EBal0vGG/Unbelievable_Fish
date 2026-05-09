package app

import (
	"context"
	"errors"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

// ErrPlatformFeeDueMismatch is returned when the invoice’s platform-fee portion (amount paid beyond goods)
// does not match max(0, fee(goods) − HELD deposit): the outstanding fee covered by the invoice must equal that expectation.
var ErrPlatformFeeDueMismatch = errors.New("billing: platform fee outstanding from invoice does not match deposit offset expectation")

type SettleWinnerDepositAfterInvoicePaid struct {
	accounts AccountRepository
	deposits AuctionDepositRepository
	ledger   LedgerRepository
	ids      IDGenerator
	clock    Clock
	events   DomainEventPublisher
}

func NewSettleWinnerDepositAfterInvoicePaid(
	accounts AccountRepository,
	deposits AuctionDepositRepository,
	ledger LedgerRepository,
	ids IDGenerator,
	clock Clock,
	events DomainEventPublisher,
) (*SettleWinnerDepositAfterInvoicePaid, error) {
	if accounts == nil || deposits == nil || ledger == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &SettleWinnerDepositAfterInvoicePaid{
		accounts: accounts,
		deposits: deposits,
		ledger:   ledger,
		ids:      ids,
		clock:    clock,
		events:   events,
	}, nil
}

// Execute settles the winner’s HELD deposit after invoice payment.
// platformFeeOutstandingPaid is the invoice line-item paying the platform-fee remainder (max(0, fee(goods) − deposit)),
// not the full platform fee. reason is stored on ledger rows (e.g. WINNER_FINALIZED) for audit; empty uses a neutral default prefix.
func (uc *SettleWinnerDepositAfterInvoicePaid) Execute(
	ctx context.Context,
	auctionID, companyID string,
	goodsAmount, platformFeeOutstandingPaid int64,
	reason string,
) error {
	if isBlank(auctionID) || isBlank(companyID) {
		return wallet.ErrInvalidIdentifier
	}
	if goodsAmount <= 0 {
		return wallet.ErrInvalidAmount
	}

	dep, err := uc.deposits.Find(ctx, auctionID, companyID)
	if err != nil {
		return err
	}
	if dep == nil {
		return ErrDepositNotFound
	}
	if dep.Status != wallet.DepositHeld {
		return nil
	}

	fee := platformFeeFromFinalPrice(goodsAmount)
	expectedDue := fee - dep.Amount
	if expectedDue < 0 {
		expectedDue = 0
	}
	if platformFeeOutstandingPaid != expectedDue {
		return ErrPlatformFeeDueMismatch
	}

	settlementRef := platformFeeSettlementRef(auctionID, companyID)
	done, err := uc.settlementAlreadyRecorded(ctx, companyID, settlementRef)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	now := uc.clock.Now().UTC()
	acc, err := uc.accounts.LoadByCompanyForUpdate(ctx, companyID)
	if err != nil {
		return err
	}

	if fee <= 0 {
		return uc.releaseFullDeposit(ctx, acc, dep, companyID, settlementRef, reason, now)
	}

	captureAmount := dep.Amount
	if fee < captureAmount {
		captureAmount = fee
	}
	releaseAmount := dep.Amount - captureAmount

	if captureAmount > 0 && releaseAmount > 0 {
		if err := acc.Capture(captureAmount); err != nil {
			return err
		}
		if err := acc.Release(releaseAmount); err != nil {
			return err
		}
		dep.MarkSettled(now)
		if err := uc.deposits.Save(ctx, dep); err != nil {
			return err
		}
		if err := uc.accounts.Save(ctx, acc); err != nil {
			return err
		}
		if err := uc.appendFeeLedger(ctx, acc, companyID, dep, captureAmount, settlementRef, reason, now); err != nil {
			return err
		}
		if err := uc.appendReleaseLedger(ctx, acc, companyID, dep, releaseAmount, settlementRef+":remainder", reason, now); err != nil {
			return err
		}
		return uc.publishCaptured(ctx, dep, companyID, captureAmount)
	}

	if captureAmount > 0 && releaseAmount == 0 {
		if err := acc.Capture(captureAmount); err != nil {
			return err
		}
		dep.MarkCaptured(now)
		if err := uc.deposits.Save(ctx, dep); err != nil {
			return err
		}
		if err := uc.accounts.Save(ctx, acc); err != nil {
			return err
		}
		if err := uc.appendFeeLedger(ctx, acc, companyID, dep, captureAmount, settlementRef, reason, now); err != nil {
			return err
		}
		return uc.publishCaptured(ctx, dep, companyID, captureAmount)
	}

	// captureAmount == 0, full release
	if err := acc.Release(releaseAmount); err != nil {
		return err
	}
	dep.MarkReleased(now)
	if err := uc.deposits.Save(ctx, dep); err != nil {
		return err
	}
	if err := uc.accounts.Save(ctx, acc); err != nil {
		return err
	}
	if err := uc.appendReleaseLedger(ctx, acc, companyID, dep, releaseAmount, settlementRef+":full_release", reason, now); err != nil {
		return err
	}
	return nil
}

func (uc *SettleWinnerDepositAfterInvoicePaid) releaseFullDeposit(
	ctx context.Context,
	acc *wallet.Account,
	dep *wallet.AuctionDeposit,
	companyID, settlementRef, reason string,
	now time.Time,
) error {
	amount := dep.Amount
	if err := acc.Release(amount); err != nil {
		return err
	}
	dep.MarkReleased(now)
	if err := uc.deposits.Save(ctx, dep); err != nil {
		return err
	}
	if err := uc.accounts.Save(ctx, acc); err != nil {
		return err
	}
	return uc.appendReleaseLedger(ctx, acc, companyID, dep, amount, settlementRef+":fee_zero", reason, now)
}

func settlementLedgerReason(scenario, entryKind string) string {
	if isBlank(scenario) {
		return entryKind
	}
	return scenario + ":" + entryKind
}

func (uc *SettleWinnerDepositAfterInvoicePaid) appendFeeLedger(
	ctx context.Context,
	acc *wallet.Account,
	companyID string,
	dep *wallet.AuctionDeposit,
	amount int64,
	settlementRef, reason string,
	now time.Time,
) error {
	return uc.ledger.Append(ctx, wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      dep.Currency,
		Amount:        amount,
		EntryType:     wallet.LedgerPlatformFeeCaptured,
		ReferenceType: "winner_deposit_after_invoice",
		ReferenceID:   settlementRef,
		Reason:        settlementLedgerReason(reason, "PLATFORM_FEE_AFTER_INVOICE"),
		CreatedAt:     now,
	})
}

func (uc *SettleWinnerDepositAfterInvoicePaid) appendReleaseLedger(
	ctx context.Context,
	acc *wallet.Account,
	companyID string,
	dep *wallet.AuctionDeposit,
	amount int64,
	referenceID, reason string,
	now time.Time,
) error {
	return uc.ledger.Append(ctx, wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      dep.Currency,
		Amount:        amount,
		EntryType:     wallet.LedgerBidDepositReleased,
		ReferenceType: "winner_deposit_after_invoice",
		ReferenceID:   referenceID,
		Reason:        settlementLedgerReason(reason, "DEPOSIT_RELEASE_AFTER_INVOICE"),
		CreatedAt:     now,
	})
}

func (uc *SettleWinnerDepositAfterInvoicePaid) publishCaptured(ctx context.Context, dep *wallet.AuctionDeposit, companyID string, amount int64) error {
	if uc.events == nil {
		return nil
	}
	return uc.events.Publish(ctx, dep.AuctionID, companyID, wallet.PlatformFeeCaptured{
		AuctionID: dep.AuctionID,
		CompanyID: companyID,
		Amount:    amount,
		Currency:  dep.Currency,
	})
}

func (uc *SettleWinnerDepositAfterInvoicePaid) settlementAlreadyRecorded(ctx context.Context, companyID, settlementRef string) (bool, error) {
	checks := []struct {
		ref string
		typ wallet.LedgerEntryType
	}{
		{settlementRef, wallet.LedgerPlatformFeeCaptured},
		{settlementRef + ":remainder", wallet.LedgerBidDepositReleased},
		{settlementRef + ":full_release", wallet.LedgerBidDepositReleased},
		{settlementRef + ":fee_zero", wallet.LedgerBidDepositReleased},
	}
	for _, c := range checks {
		ok, err := uc.ledger.ExistsByReference(ctx, companyID, "winner_deposit_after_invoice", c.ref, c.typ)
		if err != nil || ok {
			return ok, err
		}
	}
	return false, nil
}
