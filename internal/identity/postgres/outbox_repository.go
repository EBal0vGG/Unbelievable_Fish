package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	identityapp "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
	identity "github.com/EBal0vGG/Unbelievable_Fish/internal/identity/domain"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

var _ identityapp.CompanyCreatedPublisher = (*OutboxRepository)(nil)

func (r *OutboxRepository) PublishCompanyCreated(ctx context.Context, companyID string) error {
	evt := identity.CompanyCreated{CompanyID: companyID}
	payload, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	const query = `
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
	now := time.Now().UTC()
	id := newOutboxHexID()
	eventID := newOutboxHexID()
	_, err = dbtx.ExecContext(
		ctx,
		query,
		id,
		eventID,
		"identity.CompanyCreated",
		companyID,
		payload,
		now,
		now,
		nil,
		nil,
		companyID,
		nil,
		"identity",
		nil,
	)
	if err != nil {
		return err
	}
	slog.InfoContext(
		ctx,
		"outbox_message_enqueued",
		"component", "identity.outbox.repository",
		"source_context", "identity",
		"message_id", id,
		"event_type", "identity.CompanyCreated",
		"aggregate_id", companyID,
	)
	return nil
}

func newOutboxHexID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
