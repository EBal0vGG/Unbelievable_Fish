CREATE TABLE IF NOT EXISTS billing_top_ups (
    id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL,
    account_id TEXT NOT NULL REFERENCES billing_accounts(id),
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_payment_id TEXT NOT NULL DEFAULT '',
    confirmation_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,

    CHECK (amount > 0),
    CHECK (status IN ('PENDING', 'SUCCEEDED', 'FAILED', 'CANCELLED')),
    CHECK (currency = 'RUB')
);

CREATE INDEX IF NOT EXISTS idx_billing_top_ups_company
    ON billing_top_ups(company_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_top_ups_provider_payment_id
    ON billing_top_ups(provider_payment_id)
    WHERE provider_payment_id <> '';

CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_top_ups_provider_and_payment
    ON billing_top_ups(provider, provider_payment_id)
    WHERE provider_payment_id <> '';
