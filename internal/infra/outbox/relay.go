package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
)

type Message struct {
	ID            string
	EventType     string
	SourceContext string
	Payload       []byte
	OccurredAt    int64
	Attempts      int
	CorrelationID string
	CausationID   string
	CompanyID     string
	UserID        string
}

type Repository interface {
	ListUnpublished(ctx context.Context, limit int) ([]Message, error)
	MarkPublished(ctx context.Context, ids []string) error
	RecordFailure(ctx context.Context, id string, attempts int, errMsg string, failedAt *time.Time) error
}

type Bus interface {
	Publish(ctx context.Context, envelope events.Envelope) error
}

type Decoder func([]byte) (any, error)

type Relay struct {
	repo        Repository
	decoders    map[string]Decoder
	maxAttempts int
}

func NewRelay(repo Repository, decoders map[string]Decoder) *Relay {
	if decoders == nil {
		decoders = map[string]Decoder{}
	}
	return &Relay{
		repo:        repo,
		decoders:    decoders,
		maxAttempts: 5,
	}
}

func (r *Relay) RunOnce(ctx context.Context, bus Bus, limit int) error {
	if r == nil || bus == nil {
		return nil
	}
	msgs, err := r.repo.ListUnpublished(ctx, limit)
	if err != nil {
		return err
	}
	if len(msgs) == 0 {
		return nil
	}
	slog.DebugContext(ctx, "outbox_relay_batch_loaded", "component", "outbox.relay", "message_count", len(msgs), "limit", limit)
	var published []string
	var firstErr error
	for _, msg := range msgs {
		decoder := r.decoders[msg.EventType]
		var payload any
		if decoder == nil {
			if err := r.recordFailure(ctx, msg, errors.New("missing decoder for event type: "+msg.EventType)); err != nil {
				return err
			}
			if firstErr == nil {
				firstErr = errors.New("missing decoder for event type: " + msg.EventType)
			}
			continue
		}
		payload, err = decoder(msg.Payload)
		if err != nil {
			if err := r.recordFailure(ctx, msg, err); err != nil {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		envelope := events.Envelope{
			Type:       msg.EventType,
			Payload:    payload,
			OccurredAt: time.Unix(0, msg.OccurredAt),
			Meta:       metaFromMessage(msg),
		}
		slog.InfoContext(
			ctx,
			"outbox_relay_publish_attempt",
			"component", "outbox.relay",
			"message_id", msg.ID,
			"event_type", msg.EventType,
			"source_context", msg.SourceContext,
			"attempts", msg.Attempts,
			"correlation_id", msg.CorrelationID,
			"causation_id", msg.CausationID,
		)
		if err := bus.Publish(ctx, envelope); err != nil {
			if err := r.recordFailure(ctx, msg, err); err != nil {
				return err
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		slog.InfoContext(
			ctx,
			"outbox_relay_publish_success",
			"component", "outbox.relay",
			"message_id", msg.ID,
			"event_type", msg.EventType,
			"source_context", msg.SourceContext,
			"correlation_id", msg.CorrelationID,
			"causation_id", msg.CausationID,
		)
		published = append(published, msg.ID)
	}
	if err := r.repo.MarkPublished(ctx, published); err != nil {
		return err
	}
	slog.DebugContext(ctx, "outbox_relay_marked_published", "component", "outbox.relay", "message_count", len(published))
	return firstErr
}

func (r *Relay) recordFailure(ctx context.Context, msg Message, err error) error {
	if r == nil || r.repo == nil || err == nil {
		return nil
	}
	attempts := msg.Attempts + 1
	var failedAt *time.Time
	if attempts >= r.maxAttempts {
		now := time.Now().UTC()
		failedAt = &now
	}
	slog.WarnContext(
		ctx,
		"outbox_relay_failure",
		"component", "outbox.relay",
		"message_id", msg.ID,
		"event_type", msg.EventType,
		"attempts", attempts,
		"failed", failedAt != nil,
		"correlation_id", msg.CorrelationID,
		"causation_id", msg.CausationID,
		"error", err,
	)
	return r.repo.RecordFailure(ctx, msg.ID, attempts, err.Error(), failedAt)
}

func metaFromMessage(msg Message) map[string]string {
	meta := map[string]string{
		"source_context": msg.SourceContext,
	}
	if msg.CorrelationID != "" {
		meta["correlation_id"] = msg.CorrelationID
	}
	if msg.CausationID != "" {
		meta["causation_id"] = msg.CausationID
	}
	if msg.CompanyID != "" {
		meta["company_id"] = msg.CompanyID
	}
	if msg.UserID != "" {
		meta["user_id"] = msg.UserID
	}
	return meta
}

func JSONDecoder[T any]() Decoder {
	return func(data []byte) (any, error) {
		var out T
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	}
}
