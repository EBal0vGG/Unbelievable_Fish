package app

import (
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type EventEnvelope struct {
	Meta       CommandMeta
	Events     []auction.Event
	OccurredAt time.Time
}

func NewEnvelope(meta CommandMeta, events []auction.Event) EventEnvelope {
	return EventEnvelope{
		Meta:       meta,
		Events:     events,
		OccurredAt: time.Now().UTC(),
	}
}
