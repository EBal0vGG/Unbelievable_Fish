package app

import (
	"context"
	"time"

	"unbelievable_fish/internal/deals/deal"
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
	deals       DealRepository
	projections ProjectionRepository
	publisher   EventPublisher
	factory     *deal.Factory
}

func NewCreateDealFromAuctionWon(
	deals DealRepository,
	projections ProjectionRepository,
	publisher EventPublisher,
) *CreateDealFromAuctionWon {
	return &CreateDealFromAuctionWon{
		deals:       deals,
		projections: projections,
		publisher:   publisher,
		factory:     deal.NewFactory(),
	}
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

	projection, err := uc.projections.GetByAuctionID(ctx, auctionID)
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
	if err := uc.deals.Save(ctx, item); err != nil {
		return err
	}
	if err := uc.projections.Save(ctx, projection); err != nil {
		return err
	}
	return publishEvents(ctx, uc.publisher, events)
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
	repo      DealRepository
	publisher EventPublisher
}

func NewConfirmDeal(repo DealRepository, publisher EventPublisher) *ConfirmDeal {
	return &ConfirmDeal{repo: repo, publisher: publisher}
}

func (uc *ConfirmDeal) Execute(ctx context.Context, meta CommandMeta, dealID string) error {
	_ = meta
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Confirm()
	})
}

type PrepareContract struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewPrepareContract(repo DealRepository, publisher EventPublisher) *PrepareContract {
	return &PrepareContract{repo: repo, publisher: publisher}
}

func (uc *PrepareContract) Execute(ctx context.Context, meta CommandMeta, dealID, contractNumber, documentURL string) error {
	_ = meta
	if contractNumber == "" {
		return ErrContractNumberRequired
	}
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.PrepareContract(contractNumber, documentURL)
	})
}

type SignContract struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewSignContract(repo DealRepository, publisher EventPublisher) *SignContract {
	return &SignContract{repo: repo, publisher: publisher}
}

func (uc *SignContract) Execute(ctx context.Context, meta CommandMeta, dealID, signatureRef string) error {
	signedBy := actorFromMeta(meta)
	if signedBy == "" {
		return ErrSignedByRequired
	}
	if signatureRef == "" {
		return ErrSignatureRefRequired
	}
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.SignContract(signedBy, signatureRef)
	})
}

type RequestPayment struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewRequestPayment(repo DealRepository, publisher EventPublisher) *RequestPayment {
	return &RequestPayment{repo: repo, publisher: publisher}
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
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.RequestPayment(invoiceNumber, dueDate)
	})
}

type MarkDealPaid struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewMarkDealPaid(repo DealRepository, publisher EventPublisher) *MarkDealPaid {
	return &MarkDealPaid{repo: repo, publisher: publisher}
}

func (uc *MarkDealPaid) Execute(ctx context.Context, meta CommandMeta, dealID, paymentID, paymentType string) error {
	_ = meta
	if paymentID == "" {
		return ErrPaymentIDRequired
	}
	if paymentType == "" {
		return ErrPaymentTypeRequired
	}
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.MarkAsPaid(paymentID, paymentType)
	})
}

type RequestShipment struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewRequestShipment(repo DealRepository, publisher EventPublisher) *RequestShipment {
	return &RequestShipment{repo: repo, publisher: publisher}
}

func (uc *RequestShipment) Execute(ctx context.Context, meta CommandMeta, dealID string) error {
	_ = meta
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.RequestShipment()
	})
}

type MarkDealShipped struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewMarkDealShipped(repo DealRepository, publisher EventPublisher) *MarkDealShipped {
	return &MarkDealShipped{repo: repo, publisher: publisher}
}

func (uc *MarkDealShipped) Execute(ctx context.Context, meta CommandMeta, dealID, trackingNumber, carrier string) error {
	_ = meta
	if trackingNumber == "" {
		return ErrTrackingNumberRequired
	}
	if carrier == "" {
		return ErrCarrierRequired
	}
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.MarkAsShipped(trackingNumber, carrier)
	})
}

type CompleteDeal struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewCompleteDeal(repo DealRepository, publisher EventPublisher) *CompleteDeal {
	return &CompleteDeal{repo: repo, publisher: publisher}
}

func (uc *CompleteDeal) Execute(ctx context.Context, meta CommandMeta, dealID string) error {
	_ = meta
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Complete()
	})
}

type CancelDeal struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewCancelDeal(repo DealRepository, publisher EventPublisher) *CancelDeal {
	return &CancelDeal{repo: repo, publisher: publisher}
}

func (uc *CancelDeal) Execute(ctx context.Context, meta CommandMeta, dealID, reason string) error {
	cancelledBy := actorFromMeta(meta)
	if reason == "" {
		return ErrReasonRequired
	}
	if cancelledBy == "" {
		return ErrCancelledByRequired
	}
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Cancel(reason, cancelledBy)
	})
}

type UpdateDealPrice struct {
	repo      DealRepository
	publisher EventPublisher
}

func NewUpdateDealPrice(repo DealRepository, publisher EventPublisher) *UpdateDealPrice {
	return &UpdateDealPrice{repo: repo, publisher: publisher}
}

func (uc *UpdateDealPrice) Execute(ctx context.Context, meta CommandMeta, dealID string, newPrice int64) error {
	updatedBy := actorFromMeta(meta)
	if newPrice <= 0 {
		return ErrFinalPriceRequired
	}
	if updatedBy == "" {
		return ErrUpdatedByRequired
	}
	return executeDealMutation(ctx, uc.repo, uc.publisher, dealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.UpdatePrice(newPrice, updatedBy)
	})
}

func executeDealMutation(
	ctx context.Context,
	repo DealRepository,
	publisher EventPublisher,
	dealID string,
	mutate func(item *deal.Deal) ([]deal.Event, error),
) error {
	if dealID == "" {
		return ErrDealIDRequired
	}
	item, err := repo.GetByID(ctx, dealID)
	if err != nil {
		return err
	}
	events, err := mutate(item)
	if err != nil {
		return err
	}
	if err := repo.Save(ctx, item); err != nil {
		return err
	}
	return publishEvents(ctx, publisher, events)
}

func publishEvents(ctx context.Context, publisher EventPublisher, events []deal.Event) error {
	if publisher == nil || len(events) == 0 {
		return nil
	}
	return publisher.Publish(ctx, events)
}

func actorFromMeta(meta CommandMeta) string {
	if meta.UserID != "" {
		return meta.UserID
	}
	return meta.CompanyID
}
