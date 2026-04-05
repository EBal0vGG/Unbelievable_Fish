ALTER TABLE outbox_messages
    ADD COLUMN IF NOT EXISTS locked_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_outbox_messages_locked_at
    ON outbox_messages (locked_at);
