package app

import (
	"context"
	"errors"
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
	factory     *deal.Factory
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
	if auctionID == "" {
		return ErrAuctionIDRequired
	}

	return uc.uow.Do(ctx, func(tx Tx) error {
		selection, err := tx.Selections().GetByAuctionID(ctx, auctionID)
		if err != nil {
			return err
		}
		if dealID != "" && selection.DealID != "" && selection.DealID != dealID {
			return nil
		}
		if selection == nil || selection.Status == deal.WinnerSelectionExhausted {
			return ErrNoAvailableWinner
		}
		if !selection.Advance() {
			if err := tx.Selections().Save(ctx, selection); err != nil {
				return err
			}
			return ErrNoAvailableWinner
		}

		next, ok := selection.CurrentCandidate()
		if !ok {
			selection.MarkExhausted()
			if err := tx.Selections().Save(ctx, selection); err != nil {
				return err
			}
			return ErrNoAvailableWinner
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
	repo DealRepository
}

func NewGetDealByAuctionID(repo DealRepository) *GetDealByAuctionID {
	return &GetDealByAuctionID{repo: repo}
}

func (uc *GetDealByAuctionID) Execute(ctx context.Context, auctionID string) (*deal.Deal, error) {
	if auctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	return uc.repo.GetByAuctionID(ctx, auctionID)
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
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Confirm()
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
	if contractNumber == "" {
		return ErrContractNumberRequired
	}
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.PrepareContract(contractNumber, documentURL)
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
	if invoiceNumber == "" {
		return ErrInvoiceNumberRequired
	}
	return executeDealMutation(ctx, uc.uow, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.RequestPayment(invoiceNumber, dueDate)
	})
}

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
	cancelledBy := actorFromMeta(meta)
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
