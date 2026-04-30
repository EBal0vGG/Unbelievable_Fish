package app

import (
	"context"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type NoopConfirmationNotifier struct{}

func (NoopConfirmationNotifier) NotifyConfirmationRequested(
	_ context.Context,
	_ *deal.Deal,
	_ *deal.DealConfirmation,
) error {
	return nil
}
