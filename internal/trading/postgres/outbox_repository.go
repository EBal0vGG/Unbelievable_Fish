package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/app"
	"github.com/EBal0vGG/Unbelievable_Fish/internal/trading/auction"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

var _ app.OutboxRepository = (*OutboxRepository)(nil)

func (r *OutboxRepository) Add(ctx context.Context, events []auction.Event) error {
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
	meta, _ := app.CommandMetaFromContext(ctx)

	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		eventType, ok := eventTypeFor(event)
		if !ok {
			return errors.New("unknown trading event type")
		}
		aggregateID := auctionIDFor(event)

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
			nullIfBlank(meta.CorrelationID),
			nullIfBlank(meta.CausationID),
			nullIfBlank(meta.CompanyID),
			nullIfBlank(meta.UserID),
			"trading",
			nil,
		); err != nil {
			return err
		}
		slog.InfoContext(
			ctx,
			"outbox_message_enqueued",
			"component", "outbox.repository",
			"source_context", "trading",
			"message_id", messageID,
			"event_id", eventID,
			"event_type", eventType,
			"aggregate_id", aggregateID,
			"company_id", meta.CompanyID,
			"user_id", meta.UserID,
			"correlation_id", meta.CorrelationID,
			"causation_id", meta.CausationID,
		)
	}

	return nil
}

func auctionIDFor(event auction.Event) string {
	switch e := event.(type) {
	case auction.AuctionPublished:
		return e.AuctionID
	case auction.BidPlaced:
		return e.AuctionID
	case auction.AuctionClosed:
		return e.AuctionID
	case auction.AuctionWon:
		return e.AuctionID
	case auction.AuctionCancelled:
		return e.AuctionID
	default:
		return ""
	}
}

func eventTypeFor(event auction.Event) (string, bool) {
	switch event.(type) {
	case auction.AuctionPublished:
		return "trading.AuctionPublished", true
	case auction.BidPlaced:
		return "trading.BidPlaced", true
	case auction.AuctionClosed:
		return "trading.AuctionClosed", true
	case auction.AuctionWon:
		return "trading.AuctionWon", true
	case auction.AuctionCancelled:
		return "trading.AuctionCancelled", true
	default:
		return "", false
	}
}

func nullIfBlank(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func newOutboxID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
