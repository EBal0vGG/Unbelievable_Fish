package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type EventPublisher interface {
	Publish(ctx context.Context, events []auction.Event) error
}
