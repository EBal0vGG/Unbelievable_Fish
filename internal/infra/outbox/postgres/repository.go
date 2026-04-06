package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/infra/outbox"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

var _ outbox.Repository = (*Repository)(nil)

func (r *Repository) ListUnpublished(ctx context.Context, limit int) ([]outbox.Message, error) {
	const query = `
UPDATE outbox_messages
SET locked_at = $1
WHERE id IN (
    SELECT id
    FROM outbox_messages
    WHERE published_at IS NULL
      AND failed_at IS NULL
      AND (locked_at IS NULL OR locked_at < $2)
    ORDER BY created_at, id
    LIMIT $3
    FOR UPDATE SKIP LOCKED
)
RETURNING id, event_type, source_context, payload, occurred_at, attempts, correlation_id, causation_id, company_id, user_id
`
	lockTime := time.Now().UTC()
	lockCutoff := lockTime.Add(-1 * time.Minute)
	rows, err := r.db.QueryContext(ctx, query, lockTime, lockCutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []outbox.Message
	for rows.Next() {
		var msg outbox.Message
		var occurredAt time.Time
		var attempts int
		var (
			correlationID sql.NullString
			causationID   sql.NullString
			companyID     sql.NullString
			userID        sql.NullString
		)
		if err := rows.Scan(
			&msg.ID,
			&msg.EventType,
			&msg.SourceContext,
			&msg.Payload,
			&occurredAt,
			&attempts,
			&correlationID,
			&causationID,
			&companyID,
			&userID,
		); err != nil {
			return nil, err
		}
		msg.OccurredAt = occurredAt.UnixNano()
		msg.Attempts = attempts
		msg.CorrelationID = correlationID.String
		msg.CausationID = causationID.String
		msg.CompanyID = companyID.String
		msg.UserID = userID.String
		out = append(out, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) MarkPublished(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	const query = `UPDATE outbox_messages SET published_at = $1 WHERE id = $2`
	now := time.Now().UTC()
	for _, id := range ids {
		if _, err := r.db.ExecContext(ctx, query, now, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) RecordFailure(ctx context.Context, id string, attempts int, errMsg string, failedAt *time.Time) error {
	const query = `
UPDATE outbox_messages
SET attempts = $1,
    last_error = $2,
    failed_at = $3
WHERE id = $4
`
	_, err := r.db.ExecContext(ctx, query, attempts, errMsg, failedAt, id)
	return err
}
