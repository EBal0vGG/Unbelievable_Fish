package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/shared/events"
)

type testRepo struct {
	msgs     []Message
	marked   []string
	failures []failureRecord
}

type failureRecord struct {
	id       string
	attempts int
	errMsg   string
	failedAt *time.Time
}

func (r *testRepo) ListUnpublished(ctx context.Context, limit int) ([]Message, error) {
	_ = ctx
	if limit > 0 && len(r.msgs) > limit {
		return r.msgs[:limit], nil
	}
	return r.msgs, nil
}

func (r *testRepo) MarkPublished(ctx context.Context, ids []string) error {
	_ = ctx
	r.marked = append(r.marked, ids...)
	return nil
}

func (r *testRepo) RecordFailure(ctx context.Context, id string, attempts int, errMsg string, failedAt *time.Time) error {
	_ = ctx
	r.failures = append(r.failures, failureRecord{id: id, attempts: attempts, errMsg: errMsg, failedAt: failedAt})
	return nil
}

type testBus struct {
	envelopes []events.Envelope
}

func (b *testBus) Publish(ctx context.Context, envelope events.Envelope) error {
	_ = ctx
	b.envelopes = append(b.envelopes, envelope)
	return nil
}

type payloadEvent struct {
	ID string `json:"id"`
}

func TestRelayPublishesMeta(t *testing.T) {
	msg := Message{
		ID:            "1",
		EventType:     "test.Event",
		SourceContext: "test",
		Payload:       []byte(`{"id":"evt-1"}`),
		OccurredAt:    time.Now().UTC().UnixNano(),
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
		CompanyID:     "company-1",
		UserID:        "user-1",
	}
	repo := &testRepo{msgs: []Message{msg}}
	bus := &testBus{}
	relay := NewRelay(repo, map[string]Decoder{
		"test.Event": JSONDecoder[payloadEvent](),
	})

	if err := relay.RunOnce(context.Background(), bus, 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bus.envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(bus.envelopes))
	}
	envelope := bus.envelopes[0]
	if envelope.Meta["source_context"] != "test" {
		t.Fatalf("expected source_context meta")
	}
	if envelope.Meta["correlation_id"] != "corr-1" {
		t.Fatalf("expected correlation_id meta")
	}
	if envelope.Meta["company_id"] != "company-1" {
		t.Fatalf("expected company_id meta")
	}
}

func TestRelayMissingDecoderFails(t *testing.T) {
	repo := &testRepo{msgs: []Message{{ID: "1", EventType: "test.Event"}}}
	bus := &testBus{}
	relay := NewRelay(repo, nil)

	if err := relay.RunOnce(context.Background(), bus, 10); err == nil {
		t.Fatal("expected error")
	}
	if len(repo.failures) != 1 || repo.failures[0].id != "1" {
		t.Fatalf("expected failure to be recorded")
	}
}

func TestJSONDecoder(t *testing.T) {
	decoder := JSONDecoder[payloadEvent]()
	payload, err := decoder([]byte(`{"id":"evt-2"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	event := payload.(payloadEvent)
	if event.ID != "evt-2" {
		t.Fatalf("expected evt-2, got %s", event.ID)
	}
}

func TestRelaySkipsPublishOnDecoderError(t *testing.T) {
	repo := &testRepo{msgs: []Message{{ID: "1", EventType: "test.Event", Payload: []byte("{")}}}
	bus := &testBus{}
	relay := NewRelay(repo, map[string]Decoder{
		"test.Event": func([]byte) (any, error) { return nil, errors.New("decode failed") },
	})

	if err := relay.RunOnce(context.Background(), bus, 10); err == nil {
		t.Fatal("expected error")
	}
	if len(bus.envelopes) != 0 {
		t.Fatalf("expected no envelopes, got %d", len(bus.envelopes))
	}
	if len(repo.failures) != 1 || repo.failures[0].id != "1" {
		t.Fatalf("expected failure to be recorded")
	}
}

func TestRelayHandlesRawJSON(t *testing.T) {
	repo := &testRepo{msgs: []Message{{ID: "1", EventType: "test.Event", Payload: []byte(`{"id":"evt-3"}`)}}}
	bus := &testBus{}
	relay := NewRelay(repo, map[string]Decoder{
		"test.Event": func(data []byte) (any, error) {
			var out payloadEvent
			if err := json.Unmarshal(data, &out); err != nil {
				return nil, err
			}
			return out, nil
		},
	})

	if err := relay.RunOnce(context.Background(), bus, 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bus.envelopes) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(bus.envelopes))
	}
}
