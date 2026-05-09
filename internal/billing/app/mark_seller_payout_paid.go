package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/billing/wallet"
)

const ledgerRefTypeSellerPayout = "seller_payout"

const ledgerReasonSellerPayoutPaid = "SELLER_PAYOUT_PAID"

// MarkSellerPayoutPaid credits seller.available when payout becomes PAID (READY → PAID).
//
// Execute must run inside the same database transaction as other billing writes (e.g. billing UnitOfWork.WithinTx):
// it performs account mutation, payout save, ledger append, and optional outbox publish. Without a tx boundary,
// a partial failure could leave inconsistent state; Postgres also relies on the tx for FOR UPDATE locks.
//
// Idempotency: payout row status PAID, plus ledger ExistsByReference (seller_payout + SELLER_PAYOUT_CREDITED),
// and UNIQUE on ledger; concurrent admins are serialized by LoadByIDForUpdate on the payout row.
type MarkSellerPayoutPaid struct {
	payouts       SellerPayoutRepository
	accounts      AccountRepository
	ledger        LedgerRepository
	ensureAccount *CreateAccount
	ids           IDGenerator
	clock         Clock
	events        DomainEventPublisher
}

func NewMarkSellerPayoutPaid(
	payouts SellerPayoutRepository,
	accounts AccountRepository,
	ledger LedgerRepository,
	ensureAccount *CreateAccount,
	ids IDGenerator,
	clock Clock,
	events DomainEventPublisher,
) (*MarkSellerPayoutPaid, error) {
	if payouts == nil || accounts == nil || ledger == nil || ensureAccount == nil {
		return nil, ErrNilDependency
	}
	if ids == nil {
		ids = RandomHexID{}
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &MarkSellerPayoutPaid{
		payouts:       payouts,
		accounts:      accounts,
		ledger:        ledger,
		ensureAccount: ensureAccount,
		ids:           ids,
		clock:         clock,
		events:        events,
	}, nil
}

func (uc *MarkSellerPayoutPaid) Execute(ctx context.Context, payoutID string) (*wallet.SellerPayout, error) {
	if isBlank(payoutID) {
		return nil, wallet.ErrInvalidIdentifier
	}
	p, err := uc.payouts.LoadByIDForUpdate(ctx, payoutID)
	if err != nil {
		return nil, err
	}

	credited, err := uc.ledger.ExistsByReference(ctx, p.SellerCompanyID, ledgerRefTypeSellerPayout, p.ID, wallet.LedgerSellerPayoutCredited)
	if err != nil {
		return nil, err
	}

	if p.Status == wallet.SellerPayoutPaid {
		if credited {
			return p, nil
		}
		// Rare orphan: row PAID but no ledger line — append ledger only (no second Deposit).
		if err := uc.ensureAccount.Execute(ctx, p.SellerCompanyID); err != nil {
			return nil, err
		}
		acc, err := uc.accounts.LoadByCompanyForUpdate(ctx, p.SellerCompanyID)
		if err != nil {
			return nil, err
		}
		if acc.Currency() != p.Currency {
			return nil, wallet.ErrInvoiceCurrencyMismatch
		}
		now := uc.clock.Now().UTC()
		entry := wallet.LedgerEntry{
			ID:            uc.ids.NewID(),
			AccountID:     acc.ID(),
			CompanyID:     p.SellerCompanyID,
			Currency:      p.Currency,
			Amount:        p.Amount,
			EntryType:     wallet.LedgerSellerPayoutCredited,
			ReferenceType: ledgerRefTypeSellerPayout,
			ReferenceID:   p.ID,
			Reason:        ledgerReasonSellerPayoutPaid,
			CreatedAt:     now,
		}
		if err := uc.ledger.Append(ctx, entry); err != nil {
			return nil, err
		}
		return p, nil
	}

	if p.Status != wallet.SellerPayoutReady {
		return nil, wallet.ErrSellerPayoutWrongStatus
	}

	if credited {
		// Ledger already recorded the credit (e.g. retry after payout save); finish status + outbox only.
		now := uc.clock.Now().UTC()
		if err := p.MarkPaid(now); err != nil {
			return nil, err
		}
		if err := uc.payouts.Save(ctx, p); err != nil {
			return nil, err
		}
		if uc.events != nil {
			if err := uc.events.Publish(ctx, p.DealID, p.SellerCompanyID, wallet.SellerPayoutMarkedPaid{
				PayoutID:        p.ID,
				DealID:          p.DealID,
				InvoiceID:       p.InvoiceID,
				SellerCompanyID: p.SellerCompanyID,
				Amount:          p.Amount,
				Currency:        p.Currency,
				PaidAt:          now,
			}); err != nil {
				return nil, err
			}
		}
		return p, nil
	}

	if err := uc.ensureAccount.Execute(ctx, p.SellerCompanyID); err != nil {
		return nil, err
	}
	acc, err := uc.accounts.LoadByCompanyForUpdate(ctx, p.SellerCompanyID)
	if err != nil {
		return nil, err
	}
	if acc.Currency() != p.Currency {
		return nil, wallet.ErrInvoiceCurrencyMismatch
	}
	if err := acc.Deposit(p.Amount); err != nil {
		return nil, err
	}
	now := uc.clock.Now().UTC()
	if err := p.MarkPaid(now); err != nil {
		return nil, err
	}
	if err := uc.accounts.Save(ctx, acc); err != nil {
		return nil, err
	}
	if err := uc.payouts.Save(ctx, p); err != nil {
		return nil, err
	}
	entry := wallet.LedgerEntry{
		ID:            uc.ids.NewID(),
		AccountID:     acc.ID(),
		CompanyID:     p.SellerCompanyID,
		Currency:      p.Currency,
		Amount:        p.Amount,
		EntryType:     wallet.LedgerSellerPayoutCredited,
		ReferenceType: ledgerRefTypeSellerPayout,
		ReferenceID:   p.ID,
		Reason:        ledgerReasonSellerPayoutPaid,
		CreatedAt:     now,
	}
	if dup, err := uc.ledger.ExistsByReference(ctx, p.SellerCompanyID, ledgerRefTypeSellerPayout, p.ID, wallet.LedgerSellerPayoutCredited); err != nil {
		return nil, err
	} else if !dup {
		if err := uc.ledger.Append(ctx, entry); err != nil {
			return nil, err
		}
	}
	if uc.events != nil {
		if err := uc.events.Publish(ctx, p.DealID, p.SellerCompanyID, wallet.SellerPayoutMarkedPaid{
			PayoutID:        p.ID,
			DealID:          p.DealID,
			InvoiceID:       p.InvoiceID,
			SellerCompanyID: p.SellerCompanyID,
			Amount:          p.Amount,
			Currency:        p.Currency,
			PaidAt:          now,
		}); err != nil {
			return nil, err
		}
	}
	return p, nil
}
