package app

import (
	"context"
	"errors"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type RequestDealConfirmationCommand struct {
	Stage                 deal.DealConfirmationStage
	VerificationMethod    deal.VerificationMethod
	VerificationTokenHash string
	SignatureRef          string
	Comment               string
	ExpiresAt             *time.Time
}

type RequestDealConfirmation struct {
	uow      UnitOfWork
	notifier ConfirmationNotifier
}

func NewRequestDealConfirmation(uow UnitOfWork, notifier ConfirmationNotifier) (*RequestDealConfirmation, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	if notifier == nil {
		notifier = NoopConfirmationNotifier{}
	}
	return &RequestDealConfirmation{uow: uow, notifier: notifier}, nil
}

func (uc *RequestDealConfirmation) Execute(
	ctx context.Context,
	meta CommandMeta,
	dealID string,
	cmd RequestDealConfirmationCommand,
) (*deal.DealConfirmation, error) {
	if dealID == "" {
		return nil, ErrDealIDRequired
	}

	var created *deal.DealConfirmation
	err := uc.uow.Do(ctx, func(tx Tx) error {
		item, err := tx.Deals().GetByID(ctx, dealID)
		if err != nil {
			return err
		}
		pending, err := tx.Confirmations().GetPendingByDealAndStage(ctx, dealID, cmd.Stage)
		switch {
		case err == nil && pending != nil:
			return deal.ErrConfirmationAlreadyPending
		case err != nil && !errors.Is(err, deal.ErrConfirmationNotFound):
			return err
		}

		confirmation, events, err := item.RequestConfirmation(
			cmd.Stage,
			meta.CompanyID,
			meta.UserID,
			cmd.VerificationMethod,
			cmd.VerificationTokenHash,
			cmd.SignatureRef,
			cmd.Comment,
			cmd.ExpiresAt,
		)
		if err != nil {
			return err
		}
		if err := tx.Confirmations().Save(ctx, confirmation); err != nil {
			return err
		}
		if err := tx.Outbox().Add(ctx, events); err != nil {
			return err
		}
		if err := uc.notifier.NotifyConfirmationRequested(ctx, item, confirmation); err != nil {
			return err
		}
		created = confirmation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

type ApproveDealConfirmation struct {
	uow UnitOfWork
}

func NewApproveDealConfirmation(uow UnitOfWork) (*ApproveDealConfirmation, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &ApproveDealConfirmation{uow: uow}, nil
}

func (uc *ApproveDealConfirmation) Execute(
	ctx context.Context,
	meta CommandMeta,
	dealID string,
	confirmationID string,
) (*deal.DealConfirmation, error) {
	if dealID == "" {
		return nil, ErrDealIDRequired
	}
	if confirmationID == "" {
		return nil, ErrConfirmationIDRequired
	}

	var updated *deal.DealConfirmation
	err := uc.uow.Do(ctx, func(tx Tx) error {
		item, err := tx.Deals().GetByID(ctx, dealID)
		if err != nil {
			return err
		}
		confirmation, err := tx.Confirmations().GetByID(ctx, confirmationID)
		if err != nil {
			return err
		}
		if confirmation.DealID() != item.ID() {
			return deal.ErrConfirmationDealMismatch
		}

		approvalEvents, err := confirmation.Approve(meta.CompanyID, meta.UserID, time.Now().UTC())
		if err != nil {
			return err
		}
		dealEvents, err := item.ApplyApprovedConfirmation(confirmation)
		if err != nil {
			return err
		}

		if err := tx.Confirmations().Save(ctx, confirmation); err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		events := append(approvalEvents, dealEvents...)
		if err := tx.Outbox().Add(ctx, events); err != nil {
			return err
		}
		updated = confirmation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

type RejectDealConfirmation struct {
	uow UnitOfWork
}

func NewRejectDealConfirmation(uow UnitOfWork) (*RejectDealConfirmation, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &RejectDealConfirmation{uow: uow}, nil
}

func (uc *RejectDealConfirmation) Execute(
	ctx context.Context,
	meta CommandMeta,
	dealID string,
	confirmationID string,
	reason string,
) (*deal.DealConfirmation, error) {
	if dealID == "" {
		return nil, ErrDealIDRequired
	}
	if confirmationID == "" {
		return nil, ErrConfirmationIDRequired
	}

	var updated *deal.DealConfirmation
	err := uc.uow.Do(ctx, func(tx Tx) error {
		item, err := tx.Deals().GetByID(ctx, dealID)
		if err != nil {
			return err
		}
		confirmation, err := tx.Confirmations().GetByID(ctx, confirmationID)
		if err != nil {
			return err
		}
		if confirmation.DealID() != item.ID() {
			return deal.ErrConfirmationDealMismatch
		}

		events, err := confirmation.Reject(meta.CompanyID, meta.UserID, reason, time.Now().UTC())
		if err != nil {
			return err
		}
		if err := tx.Confirmations().Save(ctx, confirmation); err != nil {
			return err
		}
		if err := tx.Outbox().Add(ctx, events); err != nil {
			return err
		}
		updated = confirmation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

type GetDealConfirmations struct {
	deals         DealRepository
	confirmations DealConfirmationRepository
}

func NewGetDealConfirmations(deals DealRepository, confirmations DealConfirmationRepository) *GetDealConfirmations {
	return &GetDealConfirmations{deals: deals, confirmations: confirmations}
}

func (uc *GetDealConfirmations) Execute(ctx context.Context, dealID string) ([]*deal.DealConfirmation, error) {
	if dealID == "" {
		return nil, ErrDealIDRequired
	}
	if _, err := uc.deals.GetByID(ctx, dealID); err != nil {
		return nil, err
	}
	return uc.confirmations.ListByDealID(ctx, dealID)
}
