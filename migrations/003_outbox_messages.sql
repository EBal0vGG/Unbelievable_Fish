CREATE TABLE IF NOT EXISTS outbox_messages (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    aggregate_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_messages_published_at
    ON outbox_messages (published_at);

CREATE INDEX IF NOT EXISTS idx_outbox_messages_created_at
    ON outbox_messages (created_at);
