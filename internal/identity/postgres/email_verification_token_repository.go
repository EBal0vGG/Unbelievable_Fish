package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/EBal0vGG/Unbelievable_Fish/internal/identity/app"
)

type EmailVerificationTokenRepository struct {
	db *sql.DB
}

func NewEmailVerificationTokenRepository(db *sql.DB) *EmailVerificationTokenRepository {
	return &EmailVerificationTokenRepository{db: db}
}

var _ app.EmailVerificationTokenRepository = (*EmailVerificationTokenRepository)(nil)

func (r *EmailVerificationTokenRepository) Save(ctx context.Context, token app.EmailVerificationToken) error {
	const query = `
INSERT INTO identity_email_verification_tokens (
    id,
    user_id,
    token_hash,
    expires_at,
    used_at,
    revoked_at,
    created_at,
    sent_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.UsedAt,
		token.RevokedAt,
		token.CreatedAt,
		token.SentAt,
	)
	return err
}

func (r *EmailVerificationTokenRepository) GetByHash(ctx context.Context, tokenHash string) (app.EmailVerificationToken, error) {
	const query = `
SELECT id, user_id, token_hash, expires_at, used_at, revoked_at, created_at, sent_at
FROM identity_email_verification_tokens
WHERE token_hash = $1
`
	dbtx := DBTXFromContext(ctx, r.db)
	row := dbtx.QueryRowContext(ctx, query, tokenHash)
	token, err := scanEmailVerificationToken(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return app.EmailVerificationToken{}, app.ErrVerificationTokenInvalid
		}
		return app.EmailVerificationToken{}, err
	}
	return token, nil
}

func (r *EmailVerificationTokenRepository) MarkUsed(ctx context.Context, tokenID string, usedAt time.Time) error {
	const query = `
UPDATE identity_email_verification_tokens
SET used_at = $2
WHERE id = $1 AND used_at IS NULL
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, query, tokenID, usedAt)
	return err
}

func (r *EmailVerificationTokenRepository) RevokeActiveForUser(ctx context.Context, userID string, revokedAt time.Time) error {
	const query = `
UPDATE identity_email_verification_tokens
SET revoked_at = $2
WHERE user_id = $1
  AND used_at IS NULL
  AND revoked_at IS NULL
`
	dbtx := DBTXFromContext(ctx, r.db)
	_, err := dbtx.ExecContext(ctx, query, userID, revokedAt)
	return err
}

func (r *EmailVerificationTokenRepository) LastSentAtForUser(ctx context.Context, userID string) (time.Time, bool, error) {
	const query = `
SELECT sent_at
FROM identity_email_verification_tokens
WHERE user_id = $1
  AND used_at IS NULL
  AND revoked_at IS NULL
ORDER BY sent_at DESC
LIMIT 1
`
	dbtx := DBTXFromContext(ctx, r.db)
	var sentAt time.Time
	if err := dbtx.QueryRowContext(ctx, query, userID).Scan(&sentAt); err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, err
	}
	return sentAt, true, nil
}

func scanEmailVerificationToken(scanner interface {
	Scan(dest ...any) error
}) (app.EmailVerificationToken, error) {
	var (
		token     app.EmailVerificationToken
		usedAt    sql.NullTime
		revokedAt sql.NullTime
	)
	if err := scanner.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&usedAt,
		&revokedAt,
		&token.CreatedAt,
		&token.SentAt,
	); err != nil {
		return app.EmailVerificationToken{}, err
	}
	if usedAt.Valid {
		token.UsedAt = &usedAt.Time
	}
	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}
	return token, nil
}
