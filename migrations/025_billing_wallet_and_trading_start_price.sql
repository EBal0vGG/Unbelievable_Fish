-- Billing / Wallet (stage 2)
CREATE TABLE IF NOT EXISTS billing_accounts (
    id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL UNIQUE,
    currency TEXT NOT NULL,
    available_amount BIGINT NOT NULL,
    held_amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (available_amount >= 0),
    CHECK (held_amount >= 0)
);

CREATE TABLE IF NOT EXISTS billing_auction_deposits (
    auction_id TEXT NOT NULL,
    company_id TEXT NOT NULL,
    account_id TEXT NOT NULL REFERENCES billing_accounts(id),
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at TIMESTAMPTZ NULL,
    captured_at TIMESTAMPTZ NULL,

    PRIMARY KEY (auction_id, company_id),
    CHECK (amount > 0)
);

CREATE TABLE IF NOT EXISTS billing_ledger_entries (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES billing_accounts(id),
    company_id TEXT NOT NULL,
    type TEXT NOT NULL,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    reference_type TEXT NOT NULL,
    reference_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (amount > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_ledger_reference_company
    ON billing_ledger_entries(company_id, reference_type, reference_id, type);

CREATE TABLE IF NOT EXISTS billing_processed_top_ups (
    external_payment_id TEXT PRIMARY KEY,
    company_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (amount > 0)
);

-- Existing companies get a wallet (bootstrap / pre-billing data).
INSERT INTO billing_accounts (id, company_id, currency, available_amount, held_amount)
SELECT gen_random_uuid()::text, company_id, 'RUB', 0, 0
FROM identity_companies
ON CONFLICT (company_id) DO NOTHING;

-- Trading: persist start price for deposit calculation (5% of start, fixed for auction).
ALTER TABLE trading_auctions
    ADD COLUMN IF NOT EXISTS start_price BIGINT NOT NULL DEFAULT 0;

UPDATE trading_auctions
SET start_price = current_price
WHERE start_price = 0;
