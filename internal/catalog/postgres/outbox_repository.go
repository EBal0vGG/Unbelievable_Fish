package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/domain"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/catalog/app"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

var _ app.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Add(ctx context.Context, events []catalog.Event) error {
	if len(events) == 0 {
		return nil
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

	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		id := newOutboxID()
		eventID := newOutboxID()
		eventType := "catalog." + reflect.TypeOf(event).Name()
		aggregateID := aggregateIDFor(event)

		if _, err := dbtx.ExecContext(
			ctx,
			query,
			id,
			eventID,
			eventType,
			aggregateID,
			payload,
			now,
			now,
			nil,
			nil,
			nil,
			nil,
			"catalog",
			nil,
		); err != nil {
			return err
		}
	}

	return nil
}

func aggregateIDFor(event catalog.Event) string {
	switch e := event.(type) {
	case catalog.ProductCreated:
		return e.ProductID
	case catalog.ProductUpdated:
		return e.ProductID
	case catalog.ProductPublished:
		return e.ProductID
	case catalog.ProductUnpublished:
		return e.ProductID
	case catalog.LotCreated:
		return e.LotID
	case catalog.LotPublished:
		return e.LotID
	case catalog.LotUnpublished:
		return e.LotID
	case catalog.LotClosed:
		return e.LotID
	case catalog.LotPriceUpdated:
		return e.LotID
	case catalog.LotAuctionLinked:
		return e.LotID
	default:
		return ""
	}
}

func newOutboxID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
