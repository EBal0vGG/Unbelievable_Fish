package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"reflect"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/deals/deal"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

var _ app.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Add(ctx context.Context, events []deal.Event) error {
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
		eventType := "deals." + reflect.TypeOf(event).Name()
		aggregateID := dealIDFor(event)
		messageID := newOutboxID()
		eventID := newOutboxID()

		if _, err := dbtx.ExecContext(
			ctx,
			query,
			messageID,
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
			"deals",
			nil,
		); err != nil {
			return err
		}
		slog.InfoContext(
			ctx,
			"outbox_message_enqueued",
			"component", "outbox.repository",
			"source_context", "deals",
			"message_id", messageID,
			"event_id", eventID,
			"event_type", eventType,
			"aggregate_id", aggregateID,
		)
	}

	return nil
}

func dealIDFor(event deal.Event) string {
	switch e := event.(type) {
	case deal.DealCreated:
		return e.DealID
	case deal.DealConfirmationRequested:
		return e.DealID
	case deal.DealConfirmationApproved:
		return e.DealID
	case deal.DealConfirmationRejected:
		return e.DealID
	case deal.DealConfirmed:
		return e.DealID
	case deal.ContractPrepared:
		return e.DealID
	case deal.ContractSigned:
		return e.DealID
	case deal.PaymentRequested:
		return e.DealID
	case deal.DealPaid:
		return e.DealID
	case deal.ShipmentRequested:
		return e.DealID
	case deal.DealShipped:
		return e.DealID
	case deal.DealCompleted:
		return e.DealID
	case deal.DealCancelled:
		return e.DealID
	case deal.WinnerRejected:
		return e.DealID
	case deal.WinnerConfirmed:
		return e.DealID
	case deal.NextWinnerSelected:
		return e.DealID
	case deal.WinnerSelectionFailed:
		return e.AuctionID
	case deal.PriceUpdated:
		return e.DealID
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
