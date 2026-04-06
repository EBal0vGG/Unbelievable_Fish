package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

// DealRepository persists deal aggregates.
type DealRepository interface {
	Save(ctx context.Context, item *deal.Deal) error
	GetByID(ctx context.Context, dealID string) (*deal.Deal, error)
	GetByAuctionID(ctx context.Context, auctionID string) (*deal.Deal, error)
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
}

// OutboxRepository persists deal events for publishing.
type OutboxRepository interface {
	Add(ctx context.Context, events []deal.Event) error
}

// Tx provides transaction-scoped repositories.
type Tx interface {
	Deals() DealRepository
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
