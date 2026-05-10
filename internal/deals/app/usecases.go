package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type CreateProjection struct {
	repo ProjectionRepository
}

func NewCreateProjection(repo ProjectionRepository) *CreateProjection {
	return &CreateProjection{repo: repo}
}

func (uc *CreateProjection) Execute(
	ctx context.Context,
	meta CommandMeta,
	auctionID string,
	supplierID string,
	snapshot deal.ProductSnapshot,
	startPrice int64,
	publishedAt time.Time,
) error {
	_ = meta
	if auctionID == "" {
		return ErrAuctionIDRequired
	}
	if supplierID == "" {
		return ErrSupplierIDRequired
	}
	if startPrice <= 0 {
		return ErrStartPriceMustBePositive
	}
	if publishedAt.IsZero() {
		return ErrPublishedAtRequired
	}

	if existing, err := uc.repo.GetByAuctionID(ctx, auctionID); err == nil && existing != nil {
		return nil
	} else if err != nil && !errors.Is(err, deal.ErrProjectionNotFound) {
		return err
	}

	return uc.repo.Save(ctx, deal.NewDealProjection(auctionID, supplierID, snapshot, startPrice, publishedAt))
}

type GetProjectionByAuctionID struct {
	repo ProjectionRepository
}

func NewGetProjectionByAuctionID(repo ProjectionRepository) *GetProjectionByAuctionID {
	return &GetProjectionByAuctionID{repo: repo}
}

func (uc *GetProjectionByAuctionID) Execute(ctx context.Context, auctionID string) (*deal.DealProjection, error) {
	if auctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	return uc.repo.GetByAuctionID(ctx, auctionID)
}

type CreateDealFromAuctionWon struct {
	uow     UnitOfWork
	factory *deal.Factory
}

func NewCreateDealFromAuctionWon(
	uow UnitOfWork,
) (*CreateDealFromAuctionWon, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &CreateDealFromAuctionWon{
		uow:     uow,
		factory: deal.NewFactory(),
	}, nil
}

func (uc *CreateDealFromAuctionWon) Execute(
	ctx context.Context,
	meta CommandMeta,
	auctionID string,
	winnerCompanyID string,
	finalPrice int64,
	wonAt time.Time,
) error {
	_ = meta
	if auctionID == "" {
		return ErrAuctionIDRequired
	}
	if winnerCompanyID == "" {
		return ErrWinnerCompanyRequired
	}
	if finalPrice <= 0 {
		return ErrFinalPriceRequired
	}
	if wonAt.IsZero() {
		return ErrWonAtRequired
	}

	return uc.uow.Do(ctx, func(tx Tx) error {
		projection, err := tx.Projections().GetByAuctionID(ctx, auctionID)
		if err != nil {
			return err
		}

		item, events, err := uc.factory.CreateFromProjection(projection, winnerCompanyID, finalPrice, wonAt)
		if err != nil {
			return err
		}
		if err := item.Validate(); err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		if err := tx.Projections().Save(ctx, projection); err != nil {
			return err
		}
		return tx.Outbox().Add(ctx, events)
	})
}

type CreateDealSelectionFromAuctionWon struct {
	uow     UnitOfWork
	factory *deal.Factory
}

func NewCreateDealSelectionFromAuctionWon(
	uow UnitOfWork,
) (*CreateDealSelectionFromAuctionWon, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &CreateDealSelectionFromAuctionWon{
		uow:     uow,
		factory: deal.NewFactory(),
	}, nil
}

func (uc *CreateDealSelectionFromAuctionWon) Execute(
	ctx context.Context,
	meta CommandMeta,
	auctionID string,
	winnerCandidates []string,
	finalPrice int64,
	wonAt time.Time,
) error {
	_ = meta
	if auctionID == "" {
		return ErrAuctionIDRequired
	}
	if len(winnerCandidates) == 0 {
		return ErrWinnerCandidatesRequired
	}
	if finalPrice <= 0 {
		return ErrFinalPriceRequired
	}
	if wonAt.IsZero() {
		return ErrWonAtRequired
	}

	return uc.uow.Do(ctx, func(tx Tx) error {
		projection, err := tx.Projections().GetByAuctionID(ctx, auctionID)
		if err != nil {
			return err
		}

		selection, err := tx.Selections().GetByAuctionID(ctx, auctionID)
		if err != nil && !errors.Is(err, deal.ErrSelectionNotFound) {
			return err
		}
		if selection == nil {
			selection = deal.NewWinnerSelection(
				auctionID,
				winnerCandidates,
				finalPrice,
				wonAt,
				projection.SupplierID,
				projection.ProductSnapshot,
			)
		}
		if selection.DealID != "" {
			return nil
		}

		current, ok := selection.CurrentCandidate()
		if !ok {
			selection.MarkExhausted()
			if err := tx.Selections().Save(ctx, selection); err != nil {
				return err
			}
			return ErrNoAvailableWinner
		}

		item, events, err := uc.factory.CreateFromSelection(
			auctionID,
			selection.SupplierID,
			selection.ProductSnapshot,
			current,
			selection.FinalPrice,
			selection.WonAt,
		)
		if err != nil {
			return err
		}
		if err := item.Validate(); err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		projection.MarkAsConverted()
		if err := tx.Projections().Save(ctx, projection); err != nil {
			return err
		}
		selection.DealID = item.ID()
		if err := tx.Selections().Save(ctx, selection); err != nil {
			return err
		}
		return tx.Outbox().Add(ctx, events)
	})
}

type HandleDealDeclined struct {
	uow     UnitOfWork
	factory *deal.Factory
}

func NewHandleDealDeclined(
	uow UnitOfWork,
) (*HandleDealDeclined, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &HandleDealDeclined{
		uow:     uow,
		factory: deal.NewFactory(),
	}, nil
}

func (uc *HandleDealDeclined) Execute(ctx context.Context, meta CommandMeta, auctionID string, dealID string) error {
	_ = meta

	return uc.uow.Do(ctx, func(tx Tx) error {
		resolvedAuctionID := auctionID
		if resolvedAuctionID == "" && dealID != "" {
			item, err := tx.Deals().GetByID(ctx, dealID)
			if err != nil {
				return err
			}
			resolvedAuctionID = item.AuctionID()
		}
		if resolvedAuctionID == "" {
			return ErrAuctionIDRequired
		}
		selection, err := tx.Selections().GetByAuctionID(ctx, resolvedAuctionID)
		if err != nil {
			return err
		}
		if selection == nil || selection.Status == deal.WinnerSelectionExhausted {
			return ErrNoAvailableWinner
		}
		if selection.Status == deal.WinnerSelectionConfirmedPendingPayment {
			return deal.ErrWinnerFallbackOnlyWhileActive
		}

		effectiveDealID := dealID
		if effectiveDealID == "" {
			effectiveDealID = selection.DealID
		}
		if effectiveDealID != "" && selection.DealID != "" && selection.DealID != effectiveDealID {
			return deal.ErrStaleWinnerSelection
		}

		currentCand, ok := selection.CurrentCandidate()
		if !ok {
			return ErrNoAvailableWinner
		}

		if selection.DealID != "" {
			curDeal, err := tx.Deals().GetByID(ctx, selection.DealID)
			if err != nil {
				return err
			}
			if curDeal.Status() != deal.DealStatusCancelled {
				return deal.ErrDealNotCancelledForFallback
			}
			if curDeal.CustomerID() != currentCand {
				return deal.ErrWrongSelectedCandidate
			}
		}

		now := time.Now().UTC()
		if !selection.Advance() {
			if err := tx.Selections().Save(ctx, selection); err != nil {
				return err
			}
			if err := tx.Outbox().Add(ctx, []deal.Event{
				deal.WinnerSelectionFailed{
					SelectionID: resolvedAuctionID,
					AuctionID:   resolvedAuctionID,
					FailedAt:    now,
					Reason:      "NO_CANDIDATES_LEFT",
				},
			}); err != nil {
				return err
			}
			return nil
		}

		next, ok := selection.CurrentCandidate()
		if !ok {
			selection.MarkExhausted()
			if err := tx.Selections().Save(ctx, selection); err != nil {
				return err
			}
			if err := tx.Outbox().Add(ctx, []deal.Event{
				deal.WinnerSelectionFailed{
					SelectionID: resolvedAuctionID,
					AuctionID:   resolvedAuctionID,
					FailedAt:    now,
					Reason:      "NO_CANDIDATES_LEFT",
				},
			}); err != nil {
				return err
			}
			return nil
		}

		item, events, err := uc.factory.CreateFromSelection(
			selection.AuctionID,
			selection.SupplierID,
			selection.ProductSnapshot,
			next,
			selection.FinalPrice,
			selection.WonAt,
		)
		if err != nil {
			return err
		}
		if err := item.Validate(); err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		selection.DealID = item.ID()
		if err := tx.Selections().Save(ctx, selection); err != nil {
			return err
		}
		rank := selection.CurrentIndex + 1
		events = append(events, deal.NextWinnerSelected{
			SelectionID: selection.AuctionID,
			AuctionID:   selection.AuctionID,
			CompanyID:   next,
			Rank:        rank,
			DealID:      item.ID(),
			SelectedAt:  now,
		})
		return tx.Outbox().Add(ctx, events)
	})
}

type GetDealByID struct {
	repo DealRepository
}

func NewGetDealByID(repo DealRepository) *GetDealByID {
	return &GetDealByID{repo: repo}
}

func (uc *GetDealByID) Execute(ctx context.Context, dealID string) (*deal.Deal, error) {
	if dealID == "" {
		return nil, ErrDealIDRequired
	}
	return uc.repo.GetByID(ctx, dealID)
}

type GetDealByAuctionID struct {
	uow UnitOfWork
}

func NewGetDealByAuctionID(uow UnitOfWork) *GetDealByAuctionID {
	return &GetDealByAuctionID{uow: uow}
}

// Execute resolves the current deal for an auction: WinnerSelection.DealID is authoritative when a selection exists;
// otherwise falls back to GetActiveDealByAuctionID (e.g. legacy or no selection row yet).
func (uc *GetDealByAuctionID) Execute(ctx context.Context, auctionID string) (*deal.Deal, error) {
	if auctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	var out *deal.Deal
	err := uc.uow.Do(ctx, func(tx Tx) error {
		sel, err := tx.Selections().GetByAuctionID(ctx, auctionID)
		if err != nil && !errors.Is(err, deal.ErrSelectionNotFound) {
			return err
		}
		if err == nil {
			if sel.Status == deal.WinnerSelectionExhausted {
				return ErrDealNotFound
			}
			if sel.DealID != "" {
				d, err := tx.Deals().GetByID(ctx, sel.DealID)
				if err != nil {
					return err
				}
				if d.Status() == deal.DealStatusCancelled {
					return deal.ErrStaleWinnerSelection
				}
				out = d
				return nil
			}
		}
		d, err := tx.Deals().GetActiveDealByAuctionID(ctx, auctionID)
		if err != nil {
			return err
		}
		out = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type ConfirmDeal struct {
	uow UnitOfWork
}

func NewConfirmDeal(uow UnitOfWork) (*ConfirmDeal, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &ConfirmDeal{uow: uow}, nil
}

func (uc *ConfirmDeal) Execute(ctx context.Context, meta CommandMeta, dealID string) error {
	_ = meta
	if dealID == "" {
		return ErrDealIDRequired
	}
	return uc.uow.Do(ctx, func(tx Tx) error {
		item, err := tx.Deals().GetByIDForUpdate(ctx, dealID)
		if err != nil {
			return err
		}
		dealEvents, err := item.Confirm()
		if err != nil {
			return err
		}
		extra, err := appendAuctionWinnerConfirmedIfNeeded(ctx, tx, item)
		if err != nil {
			return err
		}
		dealEvents = append(dealEvents, extra...)
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		return tx.Outbox().Add(ctx, dealEvents)
	})
}

type PrepareContract struct {
	uow UnitOfWork
}

func NewPrepareContract(uow UnitOfWork) (*PrepareContract, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &PrepareContract{uow: uow}, nil
}

func (uc *PrepareContract) Execute(ctx context.Context, meta CommandMeta, dealID, contractNumber, documentURL string) error {
	_ = meta
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		nextContractNumber := contractNumber
		if nextContractNumber == "" {
			nextContractNumber = fmt.Sprintf("CNT-%s-%s", item.ID(), time.Now().UTC().Format("20060102150405"))
		}
		nextDocumentURL := documentURL
		if nextDocumentURL == "" {
			nextDocumentURL = fmt.Sprintf("generated://contracts/%s.pdf", nextContractNumber)
		}
		return item.PrepareContract(nextContractNumber, nextDocumentURL)
	})
}

type SignContract struct {
	uow UnitOfWork
}

func NewSignContract(uow UnitOfWork) (*SignContract, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &SignContract{uow: uow}, nil
}

func (uc *SignContract) Execute(ctx context.Context, meta CommandMeta, dealID, signatureRef string) error {
	signedBy := actorFromMeta(meta)
	if signedBy == "" {
		return ErrSignedByRequired
	}
	if signatureRef == "" {
		return ErrSignatureRefRequired
	}
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.SignContract(signedBy, signatureRef)
	})
}

type RequestPayment struct {
	uow UnitOfWork
}

func NewRequestPayment(uow UnitOfWork) (*RequestPayment, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &RequestPayment{uow: uow}, nil
}

func (uc *RequestPayment) Execute(
	ctx context.Context,
	meta CommandMeta,
	dealID string,
	invoiceNumber string,
	dueDate *time.Time,
) error {
	_ = meta
	if dealID == "" {
		return ErrDealIDRequired
	}
	return uc.uow.Do(ctx, func(tx Tx) error {
		item, err := tx.Deals().GetByIDForUpdate(ctx, dealID)
		if err != nil {
			return err
		}
		if item.Type() == deal.DealTypeAuction && item.AuctionID() != "" {
			sel, err := tx.Selections().GetByAuctionIDForUpdate(ctx, item.AuctionID())
			if err != nil {
				if errors.Is(err, deal.ErrSelectionNotFound) {
					return deal.ErrWinnerSelectionMissingForAuctionDeal
				}
				return err
			}
			if sel.DealID != item.ID() {
				return deal.ErrWinnerSelectionDealMismatch
			}
			if sel.Status != deal.WinnerSelectionConfirmedPendingPayment {
				return deal.ErrWinnerSelectionNotAwaitingPayment
			}
		}
		events, err := item.RequestPayment(invoiceNumber, dueDate)
		if err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		return tx.Outbox().Add(ctx, events)
	})
}

// MarkDealPaid applies MarkAsPaid on the deal aggregate. It must not be exposed on the public HTTP API:
// paid transitions are driven by billing (invoice paid → HandleDealInvoicePaid → MarkAsPaid with invoice id).
// The constructor and handler exist only for tests or future strictly internal tooling.
type MarkDealPaid struct {
	uow UnitOfWork
}

func NewMarkDealPaid(uow UnitOfWork) (*MarkDealPaid, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &MarkDealPaid{uow: uow}, nil
}

func (uc *MarkDealPaid) Execute(ctx context.Context, meta CommandMeta, dealID, paymentID, paymentType string) error {
	_ = meta
	if paymentID == "" {
		return ErrPaymentIDRequired
	}
	if paymentType == "" {
		return ErrPaymentTypeRequired
	}
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.MarkAsPaid(paymentID, paymentType)
	})
}

type RequestShipment struct {
	uow UnitOfWork
}

func NewRequestShipment(uow UnitOfWork) (*RequestShipment, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &RequestShipment{uow: uow}, nil
}

func (uc *RequestShipment) Execute(ctx context.Context, meta CommandMeta, dealID string) error {
	_ = meta
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.RequestShipment()
	})
}

type MarkDealShipped struct {
	uow UnitOfWork
}

func NewMarkDealShipped(uow UnitOfWork) (*MarkDealShipped, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &MarkDealShipped{uow: uow}, nil
}

func (uc *MarkDealShipped) Execute(ctx context.Context, meta CommandMeta, dealID, trackingNumber, carrier string) error {
	_ = meta
	if trackingNumber == "" {
		return ErrTrackingNumberRequired
	}
	if carrier == "" {
		return ErrCarrierRequired
	}
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.MarkAsShipped(trackingNumber, carrier)
	})
}

type CompleteDeal struct {
	uow UnitOfWork
}

func NewCompleteDeal(uow UnitOfWork) (*CompleteDeal, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &CompleteDeal{uow: uow}, nil
}

func (uc *CompleteDeal) Execute(ctx context.Context, meta CommandMeta, dealID string) error {
	_ = meta
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Complete()
	})
}

type CancelDeal struct {
	uow UnitOfWork
}

func NewCancelDeal(uow UnitOfWork) (*CancelDeal, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &CancelDeal{uow: uow}, nil
}

func (uc *CancelDeal) Execute(ctx context.Context, meta CommandMeta, dealID, reason string) error {
	// Prefer company id so Cancel() can match buyer/seller; user id alone breaks auction forfeit rules.
	cancelledBy := meta.CompanyID
	if cancelledBy == "" {
		cancelledBy = actorFromMeta(meta)
	}
	if reason == "" {
		return ErrReasonRequired
	}
	if cancelledBy == "" {
		return ErrCancelledByRequired
	}
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Cancel(reason, cancelledBy)
	})
}

type UpdateDealPrice struct {
	uow UnitOfWork
}

func NewUpdateDealPrice(uow UnitOfWork) (*UpdateDealPrice, error) {
	if uow == nil {
		return nil, ErrNilUnitOfWork
	}
	return &UpdateDealPrice{uow: uow}, nil
}

func (uc *UpdateDealPrice) Execute(ctx context.Context, meta CommandMeta, dealID string, newPrice int64) error {
	updatedBy := actorFromMeta(meta)
	if newPrice <= 0 {
		return ErrFinalPriceRequired
	}
	if updatedBy == "" {
		return ErrUpdatedByRequired
	}
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.UpdatePrice(newPrice, updatedBy)
	})
}

func appendAuctionWinnerConfirmedIfNeeded(ctx context.Context, tx Tx, d *deal.Deal) ([]deal.Event, error) {
	if d.Type() != deal.DealTypeAuction || d.AuctionID() == "" {
		return nil, nil
	}
	if d.Status() != deal.DealStatusConfirmed {
		return nil, nil
	}
	// Row-lock selection together with locked deal so two confirms cannot both pass MarkCurrentConfirmed.
	sel, err := tx.Selections().GetByAuctionIDForUpdate(ctx, d.AuctionID())
	if err != nil {
		if errors.Is(err, deal.ErrSelectionNotFound) {
			return nil, deal.ErrWinnerSelectionMissingForAuctionDeal
		}
		return nil, err
	}
	if cand, ok := sel.CurrentCandidate(); !ok || cand != d.CustomerID() {
		return nil, deal.ErrWrongSelectedCandidate
	}
	if err := sel.MarkCurrentConfirmed(d.ID()); err != nil {
		return nil, err
	}
	if err := tx.Selections().Save(ctx, sel); err != nil {
		return nil, err
	}
	at := d.ConfirmedAt()
	if at == nil {
		return nil, deal.ErrCannotConfirmDeal
	}
	return []deal.Event{
		deal.WinnerConfirmed{
			SelectionID: d.AuctionID(),
			DealID:      d.ID(),
			AuctionID:   d.AuctionID(),
			CompanyID:   d.CustomerID(),
			FinalPrice:  d.UnitPrice(),
			ConfirmedAt: *at,
		},
	}, nil
}

func executeDealMutation(
	ctx context.Context,
	uow UnitOfWork,
	dealID string,
	mutate func(item *deal.Deal) ([]deal.Event, error),
) error {
	if dealID == "" {
		return ErrDealIDRequired
	}
	return uow.Do(ctx, func(tx Tx) error {
		item, err := tx.Deals().GetByID(ctx, dealID)
		if err != nil {
			return err
		}
		events, err := mutate(item)
		if err != nil {
			return err
		}
		if err := tx.Deals().Save(ctx, item); err != nil {
			return err
		}
		return tx.Outbox().Add(ctx, events)
	})
}

func actorFromMeta(meta CommandMeta) string {
	if meta.UserID != "" {
		return meta.UserID
	}
	return meta.CompanyID
}
