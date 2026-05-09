package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

// DealRepository persists deal aggregates.
type DealRepository interface {
	Save(ctx context.Context, item *deal.Deal) error
	GetByID(ctx context.Context, dealID string) (*deal.Deal, error)
	GetByIDForUpdate(ctx context.Context, dealID string) (*deal.Deal, error)
	// GetActiveDealByAuctionID returns the single non-cancelled deal for the auction.
	// More than one such row is a consistency violation (ErrMultipleActiveDealsForAuction).
	GetActiveDealByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error)
}

type DealConfirmationRepository interface {
	Save(ctx context.Context, item *deal.DealConfirmation) error
	GetByID(ctx context.Context, confirmationID string) (*deal.DealConfirmation, error)
	GetPendingByDealAndStage(ctx context.Context, dealID string, stage deal.DealConfirmationStage) (*deal.DealConfirmation, error)
	ListByDealID(ctx context.Context, dealID string) ([]*deal.DealConfirmation, error)
}

// ProjectionRepository persists auction-to-deal projections.
type ProjectionRepository interface {
	Save(ctx context.Context, item *deal.DealProjection) error
	GetByAuctionID(ctx context.Context, auctionID string) (*deal.DealProjection, error)
}

// WinnerSelectionRepository persists winner selection process state.
type WinnerSelectionRepository interface {
	Save(ctx context.Context, item *deal.WinnerSelection) error
	GetByAuctionID(ctx context.Context, auctionID string) (*deal.WinnerSelection, error)
	GetByAuctionIDForUpdate(ctx context.Context, auctionID string) (*deal.WinnerSelection, error)
}

// OutboxRepository persists deal events for publishing.
type OutboxRepository interface {
	Add(ctx context.Context, events []deal.Event) error
}

// Tx provides transaction-scoped repositories.
type Tx interface {
	Deals() DealRepository
	Confirmations() DealConfirmationRepository
	Projections() ProjectionRepository
	Selections() WinnerSelectionRepository
	Outbox() OutboxRepository
}

// UnitOfWork executes operations in a transaction.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(Tx) error) error
}

// EventPublisher delivers domain events to outer layers.
type EventPublisher interface {
	Publish(ctx context.Context, events []deal.Event) error
}

type ConfirmationNotifier interface {
	NotifyConfirmationRequested(ctx context.Context, item *deal.Deal, confirmation *deal.DealConfirmation) error
}
