ALTER TABLE outbox_messages
    ADD COLUMN IF NOT EXISTS source_context TEXT NOT NULL DEFAULT 'unknown';

CREATE INDEX IF NOT EXISTS idx_outbox_messages_source_context
    ON outbox_messages (source_context);
