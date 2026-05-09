package app

import (
	"context"
	"log/slog"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

type ReleaseAuctionDepositsExceptCandidates struct {
	accounts AccountRepository
	deposits AuctionDepositRepository
	ledger   LedgerRepository
	ids      IDGenerator
	clock    Clock
	events   DomainEventPublisher
}

func NewReleaseAuctionDepositsExceptCandidates(
	accounts AccountRepository,
	deposits AuctionDepositRepository,
	ledger LedgerRepository,
	ids IDGenerator,
	clock Clock,
	events DomainEventPublisher,
) (*ReleaseAuctionDepositsExceptCandidates, error) {
	if accounts == nil || deposits == nil || ledger == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &ReleaseAuctionDepositsExceptCandidates{
		accounts: accounts,
		deposits: deposits,
		ledger:   ledger,
		ids:      ids,
		clock:    clock,
		events:   events,
	}, nil
}

func (uc *ReleaseAuctionDepositsExceptCandidates) Execute(
	ctx context.Context,
	auctionID string,
	candidateCompanyIDs []string,
	reason string,
) error {
	if isBlank(auctionID) {
		return wallet.ErrInvalidIdentifier
	}
	for _, id := range candidateCompanyIDs {
		if isBlank(id) {
			return wallet.ErrInvalidIdentifier
		}
	}

	now := uc.clock.Now().UTC()

	candidates := make(map[string]struct{}, len(candidateCompanyIDs))
	for _, id := range candidateCompanyIDs {
		candidates[id] = struct{}{}
	}

	deposits, err := uc.deposits.ListByAuction(ctx, auctionID)
	if err != nil {
		return err
	}

	depositCompanies := make(map[string]struct{}, len(deposits))
	for _, dep := range deposits {
		depositCompanies[dep.CompanyID] = struct{}{}
	}
	for _, companyID := range candidateCompanyIDs {
		if _, ok := depositCompanies[companyID]; !ok {
			slog.WarnContext(ctx, "billing_auction_won_candidate_missing_deposit",
				"auction_id", auctionID,
				"company_id", companyID,
			)
		}
	}

	for _, dep := range deposits {
		if _, ok := candidates[dep.CompanyID]; ok {
			continue
		}
		if dep.Status != wallet.DepositHeld {
			continue
		}

		ref := auctionDepositLedgerRef(auctionID, dep.CompanyID)
		exists, err := uc.ledger.ExistsByReference(ctx, dep.CompanyID, "auction_deposit_release", ref, wallet.LedgerBidDepositReleased)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		acc, err := uc.accounts.LoadByCompanyForUpdate(ctx, dep.CompanyID)
		if err != nil {
			return err
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
			CompanyID:     dep.CompanyID,
			Currency:      dep.Currency,
			Amount:        dep.Amount,
			EntryType:     wallet.LedgerBidDepositReleased,
			ReferenceType: "auction_deposit_release",
			ReferenceID:   ref,
			Reason:        reason,
			CreatedAt:     now,
		}
		if err := uc.ledger.Append(ctx, entry); err != nil {
			return err
		}
		if uc.events != nil {
			if err := uc.events.Publish(ctx, dep.AuctionID, dep.CompanyID, wallet.AuctionDepositReleased{
				AuctionID: dep.AuctionID,
				CompanyID: dep.CompanyID,
				Amount:    dep.Amount,
				Currency:  dep.Currency,
				Reason:    reason,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}
