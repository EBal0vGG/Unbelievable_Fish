package inmemory

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
)

type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]events.Handler
}

func NewBus() *Bus {
	return &Bus{
		handlers: make(map[string][]events.Handler),
	}
}

func (b *Bus) Subscribe(eventType string, handler events.Handler) {
	if b == nil || handler == nil || eventType == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
	slog.Debug("event_bus_subscribed", "component", "eventbus.inmemory", "event_type", eventType, "handler_count", len(b.handlers[eventType]))
}

func (b *Bus) Publish(ctx context.Context, envelope events.Envelope) error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	handlers := append([]events.Handler(nil), b.handlers[envelope.Type]...)
	b.mu.RUnlock()
	if len(handlers) == 0 {
		slog.DebugContext(ctx, "event_bus_no_handlers", "component", "eventbus.inmemory", "event_type", envelope.Type)
		return nil
	}

	var errs []error
	for idx, handler := range handlers {
		if handler == nil {
			continue
		}
		if err := handler(ctx, envelope); err != nil {
			slog.WarnContext(
				ctx,
				"event_bus_handler_failed",
				"component", "eventbus.inmemory",
				"event_type", envelope.Type,
				"handler_index", idx,
				"correlation_id", envelope.Meta["correlation_id"],
				"causation_id", envelope.Meta["causation_id"],
				"error", err,
			)
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		slog.InfoContext(
			ctx,
			"event_bus_dispatched",
			"component", "eventbus.inmemory",
			"event_type", envelope.Type,
			"handler_count", len(handlers),
			"correlation_id", envelope.Meta["correlation_id"],
			"causation_id", envelope.Meta["causation_id"],
		)
		return nil
	}
	return errors.Join(errs...)
}
