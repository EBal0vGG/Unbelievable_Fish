package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) Publish(ctx context.Context, aggregateID, companyID string, event any) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	eventType := "billing." + reflect.TypeOf(event).Name()
	const q = `
INSERT INTO outbox_messages (
    id,
    event_id,
    event_type,
    aggregate_id,
    payload,
    occurred_at,
    created_at,
    correlation_id,
    causation_id,
    company_id,
    user_id,
    source_context,
    published_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err = dbtx.ExecContext(
		ctx,
		q,
		newOutboxID(),
		newOutboxID(),
		eventType,
		aggregateID,
		payload,
		now,
		now,
		nil,
		nil,
		companyID,
		nil,
		"billing",
		nil,
	)
	return err
}

func newOutboxID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
