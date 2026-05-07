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
}

func NewCapturePlatformFeeFromDeposit(accounts AccountRepository, deposits AuctionDepositRepository, ledger LedgerRepository, ids IDGenerator, clock Clock) (*CapturePlatformFeeFromDeposit, error) {
	if accounts == nil || deposits == nil || ledger == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &CapturePlatformFeeFromDeposit{accounts: accounts, deposits: deposits, ledger: ledger, ids: ids, clock: clock}, nil
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
	d := dep.Amount

	if fee <= 0 {
		if err := acc.Release(d); err != nil {
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
			Amount:        d,
			EntryType:     wallet.LedgerBidDepositReleased,
			ReferenceType: "platform_fee_from_deposit_zero",
			ReferenceID:   settlementRef,
			CreatedAt:     now,
		})
	}

	if d >= fee {
		if err := acc.Capture(fee); err != nil {
			return err
		}
		remainder := d - fee
		if remainder > 0 {
			if err := acc.Release(remainder); err != nil {
				return err
			}
		}
		dep.MarkReleased(now)
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
			CreatedAt:     now,
		}); err != nil {
			return err
		}
		if remainder > 0 {
			return uc.ledger.Append(ctx, wallet.LedgerEntry{
				ID:            uc.ids.NewID(),
				AccountID:     acc.ID(),
				CompanyID:     companyID,
				Currency:      dep.Currency,
				Amount:        remainder,
				EntryType:     wallet.LedgerBidDepositReleased,
				ReferenceType: "platform_fee_deposit_remainder",
				ReferenceID:   settlementRef + ":remainder",
				CreatedAt:     now,
			})
		}
		return nil
	}

	if err := acc.Capture(d); err != nil {
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
		Amount:        d,
		EntryType:     wallet.LedgerPlatformFeeCaptured,
		ReferenceType: "platform_fee_from_deposit",
		ReferenceID:   settlementRef,
		CreatedAt:     now,
	}); err != nil {
		return err
	}
	due := fee - d
	if due <= 0 {
		return nil
	}
	return uc.ledger.Append(ctx, wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     companyID,
		Currency:      dep.Currency,
		Amount:        due,
		EntryType:     wallet.LedgerPlatformFeeDue,
		ReferenceType: "platform_fee_due",
		ReferenceID:   settlementRef + ":due",
		CreatedAt:     now,
	})
}

func (uc *CapturePlatformFeeFromDeposit) settlementAlreadyRecorded(ctx context.Context, companyID, settlementRef string) (bool, error) {
	ok, err := uc.ledger.ExistsByReference(ctx, companyID, "platform_fee_from_deposit", settlementRef, wallet.LedgerPlatformFeeCaptured)
	if err != nil || ok {
		return ok, err
	}
	return uc.ledger.ExistsByReference(ctx, companyID, "platform_fee_from_deposit_zero", settlementRef, wallet.LedgerBidDepositReleased)
}
