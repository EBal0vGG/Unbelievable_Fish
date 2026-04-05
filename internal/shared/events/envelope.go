package events

import (
	"context"
	"time"
)

type Envelope struct {
	Type       string
	Payload    any
	OccurredAt time.Time
	Meta       map[string]string
}

type Handler func(ctx context.Context, envelope Envelope) error

type Publisher interface {
	Publish(ctx context.Context, envelope Envelope) error
}

type Subscriber interface {
	Subscribe(eventType string, handler Handler)
}
