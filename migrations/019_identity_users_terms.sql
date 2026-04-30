ALTER TABLE identity_users
    ADD COLUMN IF NOT EXISTS terms_accepted_at TIMESTAMPTZ;

ALTER TABLE identity_users
    ADD COLUMN IF NOT EXISTS terms_version TEXT;
