package inmemory

import (
	"context"
	"errors"
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
}

func (b *Bus) Publish(ctx context.Context, envelope events.Envelope) error {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	handlers := append([]events.Handler(nil), b.handlers[envelope.Type]...)
	b.mu.RUnlock()
	if len(handlers) == 0 {
		return nil
	}

	var errs []error
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		if err := handler(ctx, envelope); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
