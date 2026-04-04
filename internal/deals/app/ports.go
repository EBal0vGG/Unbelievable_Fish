package app

import (
	"context"

	"unbelievable_fish/internal/deals/deal"
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

// EventPublisher delivers domain events to outer layers.
type EventPublisher interface {
	Publish(ctx context.Context, events []deal.Event) error
}
