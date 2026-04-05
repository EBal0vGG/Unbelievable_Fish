ALTER TABLE outbox_messages
    ADD COLUMN IF NOT EXISTS correlation_id TEXT NULL,
    ADD COLUMN IF NOT EXISTS causation_id TEXT NULL,
    ADD COLUMN IF NOT EXISTS company_id TEXT NULL,
    ADD COLUMN IF NOT EXISTS user_id TEXT NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_messages_correlation_id
    ON outbox_messages (correlation_id);

CREATE INDEX IF NOT EXISTS idx_outbox_messages_causation_id
    ON outbox_messages (causation_id);
