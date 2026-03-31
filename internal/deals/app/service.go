package app

import (
	"context"

	"unbelievable_fish/internal/deals/deal"
)

type Service struct {
	deals       DealRepository
	projections ProjectionRepository
	publisher   EventPublisher
	factory     *deal.Factory
}

func NewService(deals DealRepository, projections ProjectionRepository, publisher EventPublisher) (*Service, error) {
	if deals == nil {
		return nil, ErrDealRepositoryRequired
	}
	if projections == nil {
		return nil, ErrProjectionRepositoryRequired
	}
	if publisher == nil {
		return nil, ErrEventPublisherRequired
	}

	return &Service{
		deals:       deals,
		projections: projections,
		publisher:   publisher,
		factory:     deal.NewFactory(),
	}, nil
}

func (s *Service) CreateProjectionFromLotPublished(ctx context.Context, cmd CreateProjectionCommand) (*deal.DealProjection, error) {
	if cmd.AuctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	if cmd.SupplierID == "" {
		return nil, ErrSupplierIDRequired
	}
	if cmd.StartPrice <= 0 {
		return nil, ErrStartPriceMustBePositive
	}
	if cmd.PublishedAt.IsZero() {
		return nil, ErrPublishedAtRequired
	}

	projection := deal.NewDealProjection(
		cmd.AuctionID,
		cmd.SupplierID,
		cmd.ProductSnapshot,
		cmd.StartPrice,
		cmd.PublishedAt,
	)

	if err := s.projections.Save(ctx, projection); err != nil {
		return nil, err
	}

	return projection, nil
}

func (s *Service) CreateDealFromAuctionWon(ctx context.Context, cmd CreateDealFromAuctionWonCommand) (*deal.Deal, error) {
	if cmd.AuctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	if cmd.WinnerCompanyID == "" {
		return nil, ErrWinnerCompanyRequired
	}
	if cmd.FinalPrice <= 0 {
		return nil, ErrFinalPriceRequired
	}
	if cmd.WonAt.IsZero() {
		return nil, ErrWonAtRequired
	}

	projection, err := s.projections.GetByAuctionID(ctx, cmd.AuctionID)
	if err != nil {
		return nil, err
	}

	item, events, err := s.factory.CreateFromProjection(projection, cmd.WinnerCompanyID, cmd.FinalPrice, cmd.WonAt)
	if err != nil {
		return nil, err
	}
	if err := item.Validate(); err != nil {
		return nil, err
	}
	if err := s.deals.Save(ctx, item); err != nil {
		return nil, err
	}
	if err := s.projections.Save(ctx, projection); err != nil {
		return nil, err
	}
	if err := s.publisher.Publish(ctx, events); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *Service) GetDealByID(ctx context.Context, query GetDealByIDQuery) (*deal.Deal, error) {
	if query.DealID == "" {
		return nil, ErrDealIDRequired
	}
	return s.deals.GetByID(ctx, query.DealID)
}

func (s *Service) GetDealByAuctionID(ctx context.Context, query GetDealByAuctionIDQuery) (*deal.Deal, error) {
	if query.AuctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	return s.deals.GetByAuctionID(ctx, query.AuctionID)
}

func (s *Service) GetProjectionByAuctionID(ctx context.Context, query GetProjectionByAuctionIDQuery) (*deal.DealProjection, error) {
	if query.AuctionID == "" {
		return nil, ErrAuctionIDRequired
	}
	return s.projections.GetByAuctionID(ctx, query.AuctionID)
}

func (s *Service) ConfirmDeal(ctx context.Context, cmd ConfirmDealCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Confirm()
	})
}

func (s *Service) PrepareContract(ctx context.Context, cmd PrepareContractCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	if cmd.ContractNumber == "" {
		return nil, ErrContractNumberRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.PrepareContract(cmd.ContractNumber, cmd.DocumentURL)
	})
}

func (s *Service) SignContract(ctx context.Context, cmd SignContractCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	if cmd.SignedBy == "" {
		return nil, ErrSignedByRequired
	}
	if cmd.SignatureRef == "" {
		return nil, ErrSignatureRefRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.SignContract(cmd.SignedBy, cmd.SignatureRef)
	})
}

func (s *Service) RequestPayment(ctx context.Context, cmd RequestPaymentCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	if cmd.InvoiceNumber == "" {
		return nil, ErrInvoiceNumberRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.RequestPayment(cmd.InvoiceNumber, cmd.DueDate)
	})
}

func (s *Service) MarkDealPaid(ctx context.Context, cmd MarkDealPaidCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	if cmd.PaymentID == "" {
		return nil, ErrPaymentIDRequired
	}
	if cmd.PaymentType == "" {
		return nil, ErrPaymentTypeRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.MarkAsPaid(cmd.PaymentID, cmd.PaymentType)
	})
}

func (s *Service) RequestShipment(ctx context.Context, cmd RequestShipmentCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.RequestShipment()
	})
}

func (s *Service) MarkDealShipped(ctx context.Context, cmd MarkDealShippedCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	if cmd.TrackingNumber == "" {
		return nil, ErrTrackingNumberRequired
	}
	if cmd.Carrier == "" {
		return nil, ErrCarrierRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.MarkAsShipped(cmd.TrackingNumber, cmd.Carrier)
	})
}

func (s *Service) CompleteDeal(ctx context.Context, cmd CompleteDealCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Complete()
	})
}

func (s *Service) CancelDeal(ctx context.Context, cmd CancelDealCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	if cmd.Reason == "" {
		return nil, ErrReasonRequired
	}
	if cmd.CancelledBy == "" {
		return nil, ErrCancelledByRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.Cancel(cmd.Reason, cmd.CancelledBy)
	})
}

func (s *Service) UpdateDealPrice(ctx context.Context, cmd UpdateDealPriceCommand) (*deal.Deal, error) {
	if cmd.DealID == "" {
		return nil, ErrDealIDRequired
	}
	if cmd.NewPrice <= 0 {
		return nil, ErrFinalPriceRequired
	}
	if cmd.UpdatedBy == "" {
		return nil, ErrUpdatedByRequired
	}
	return s.mutateDeal(ctx, cmd.DealID, func(item *deal.Deal) ([]deal.Event, error) {
		return item.UpdatePrice(cmd.NewPrice, cmd.UpdatedBy)
	})
}

func (s *Service) mutateDeal(ctx context.Context, dealID string, mutate func(item *deal.Deal) ([]deal.Event, error)) (*deal.Deal, error) {
	item, err := s.deals.GetByID(ctx, dealID)
	if err != nil {
		return nil, err
	}

	events, err := mutate(item)
	if err != nil {
		return nil, err
	}
	if err := s.deals.Save(ctx, item); err != nil {
		return nil, err
	}
	if err := s.publisher.Publish(ctx, events); err != nil {
		return nil, err
	}

	return item, nil
}
