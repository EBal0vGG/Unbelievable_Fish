package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type CapturePlatformFeeFromDeposit struct {
	accounts AccountRepository
	deposits AuctionDepositRepository
	ledger   LedgerRepository
	ids      IDGenerator
	clock    Clock
	events   DomainEventPublisher
}

func NewCapturePlatformFeeFromDeposit(accounts AccountRepository, deposits AuctionDepositRepository, ledger LedgerRepository, ids IDGenerator, clock Clock, events DomainEventPublisher) (*CapturePlatformFeeFromDeposit, error) {
	if accounts == nil || deposits == nil || ledger == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &CapturePlatformFeeFromDeposit{accounts: accounts, deposits: deposits, ledger: ledger, ids: ids, clock: clock, events: events}, nil
}

func (uc *CapturePlatformFeeFromDeposit) Execute(ctx context.Context, companyID, auctionID string, finalPrice int64) error {
	if isBlank(companyID) || isBlank(auctionID) {
		return wallet.ErrInvalidIdentifier
	}
	if finalPrice < 0 {
		return wallet.ErrInvalidAmount
	}
	now := uc.clock.Now().UTC()
	settlementRef := platformFeeSettlementRef(auctionID, companyID)

	done, err := uc.settlementAlreadyRecorded(ctx, companyID, settlementRef)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

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
	if dep.Status != wallet.DepositHeld {
		return ErrDepositNotHeld
	}

	fee := platformFeeFromFinalPrice(finalPrice)
	depositAmount := dep.Amount

	if fee <= 0 {
		if err := acc.Release(depositAmount); err != nil {
			return err
		}
		dep.MarkReleased(now)
		if err := uc.deposits.Save(ctx, dep); err != nil {
			return err
		}
		if err := uc.accounts.Save(ctx, acc); err != nil {
			return err
		}
		return uc.ledger.Append(ctx, wallet.LedgerEntry{
			ID:            uc.ids.NewID(),
			AccountID:     acc.ID(),
			CompanyID:     companyID,
			Currency:      dep.Currency,
			Amount:        depositAmount,
			EntryType:     wallet.LedgerBidDepositReleased,
			ReferenceType: "platform_fee_from_deposit_zero",
			ReferenceID:   settlementRef,
			Reason:        "PLATFORM_FEE_ZERO",
			CreatedAt:     now,
		})
	}

	// deposit > fee: capture fee, release remainder, mark as settled.
	if depositAmount > fee {
		remainder := depositAmount - fee
		if err := acc.Capture(fee); err != nil {
			return err
		}
		if err := acc.Release(remainder); err != nil {
			return err
		}
		dep.MarkSettled(now)
		if err := uc.deposits.Save(ctx, dep); err != nil {
			return err
		}
		if err := uc.accounts.Save(ctx, acc); err != nil {
			return err
		}
		if err := uc.ledger.Append(ctx, wallet.LedgerEntry{
			ID:            uc.ids.NewID(),
			AccountID:     acc.ID(),
			CompanyID:     companyID,
			Currency:      dep.Currency,
			Amount:        fee,
			EntryType:     wallet.LedgerPlatformFeeCaptured,
			ReferenceType: "platform_fee_from_deposit",
			ReferenceID:   settlementRef,
			Reason:        "PLATFORM_FEE",
			CreatedAt:     now,
		}); err != nil {
			return err
		}
		if err := uc.ledger.Append(ctx, wallet.LedgerEntry{
			ID:            uc.ids.NewID(),
			AccountID:     acc.ID(),
			CompanyID:     companyID,
			Currency:      dep.Currency,
			Amount:        remainder,
			EntryType:     wallet.LedgerBidDepositReleased,
			ReferenceType: "platform_fee_deposit_remainder",
			ReferenceID:   settlementRef + ":remainder",
			Reason:        "PLATFORM_FEE_REMAINDER",
			CreatedAt:     now,
		}); err != nil {
			return err
		}
		if uc.events != nil {
			return uc.events.Publish(ctx, dep.AuctionID, companyID, wallet.PlatformFeeCaptured{
				AuctionID: dep.AuctionID,
				CompanyID: companyID,
				Amount:    fee,
				Currency:  dep.Currency,
			})
		}
		return nil
	}

	// deposit == fee: full capture, mark captured.
	if depositAmount == fee {
		if err := acc.Capture(depositAmount); err != nil {
			return err
		}
		dep.MarkCaptured(now)
		if err := uc.deposits.Save(ctx, dep); err != nil {
			return err
		}
		if err := uc.accounts.Save(ctx, acc); err != nil {
			return err
		}
		if err := uc.ledger.Append(ctx, wallet.LedgerEntry{
			ID:            uc.ids.NewID(),
			AccountID:     acc.ID(),
			CompanyID:     companyID,
			Currency:      dep.Currency,
			Amount:        depositAmount,
			EntryType:     wallet.LedgerPlatformFeeCaptured,
			ReferenceType: "platform_fee_from_deposit",
			ReferenceID:   settlementRef,
			Reason:        "PLATFORM_FEE",
			CreatedAt:     now,
		}); err != nil {
			return err
		}
		if uc.events != nil {
			return uc.events.Publish(ctx, dep.AuctionID, companyID, wallet.PlatformFeeCaptured{
				AuctionID: dep.AuctionID,
				CompanyID: companyID,
				Amount:    depositAmount,
				Currency:  dep.Currency,
			})
		}
		return nil
	}

	// deposit < fee: full deposit captured + due on remainder.
	if err := acc.Capture(depositAmount); err != nil {
		return err
	}
	dep.MarkCaptured(now)
	if err := uc.deposits.Save(ctx, dep); err != nil {
		return err
	}
	if err := uc.accounts.Save(ctx, acc); err != nil {
		return err
	}
	if err := uc.ledger.Append(ctx, wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      dep.Currency,
		Amount:        depositAmount,
		EntryType:     wallet.LedgerPlatformFeeCaptured,
		ReferenceType: "platform_fee_from_deposit",
		ReferenceID:   settlementRef,
		Reason:        "PLATFORM_FEE",
		CreatedAt:     now,
	}); err != nil {
		return err
	}
	due := fee - depositAmount
	if err := uc.ledger.Append(ctx, wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      dep.Currency,
		Amount:        due,
		EntryType:     wallet.LedgerPlatformFeeDue,
		ReferenceType: "platform_fee_due",
		ReferenceID:   settlementRef + ":due",
		Reason:        "PLATFORM_FEE_DUE",
		CreatedAt:     now,
	}); err != nil {
		return err
	}
	if uc.events != nil {
		if err := uc.events.Publish(ctx, dep.AuctionID, companyID, wallet.PlatformFeeCaptured{
			AuctionID: dep.AuctionID,
			CompanyID: companyID,
			Amount:    depositAmount,
			Currency:  dep.Currency,
		}); err != nil {
			return err
		}
		return uc.events.Publish(ctx, dep.AuctionID, companyID, wallet.PlatformFeePaymentRequired{
			AuctionID: dep.AuctionID,
			CompanyID: companyID,
			AmountDue: due,
			Currency:  dep.Currency,
		})
	}
	return nil
}

func (uc *CapturePlatformFeeFromDeposit) settlementAlreadyRecorded(ctx context.Context, companyID, settlementRef string) (bool, error) {
	ok, err := uc.ledger.ExistsByReference(ctx, companyID, "platform_fee_from_deposit", settlementRef, wallet.LedgerPlatformFeeCaptured)
	if err != nil || ok {
		return ok, err
	}
	return uc.ledger.ExistsByReference(ctx, companyID, "platform_fee_from_deposit_zero", settlementRef, wallet.LedgerBidDepositReleased)
}
