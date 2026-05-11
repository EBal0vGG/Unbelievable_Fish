ALTER TABLE identity_users
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE identity_users
SET email_verified = TRUE
WHERE email_verified = FALSE;

CREATE TABLE IF NOT EXISTS identity_email_verification_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES identity_users(user_id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_identity_email_verification_tokens_user_id
    ON identity_email_verification_tokens (user_id);

CREATE INDEX IF NOT EXISTS idx_identity_email_verification_tokens_expires_at
    ON identity_email_verification_tokens (expires_at);
