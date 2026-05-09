CREATE TABLE IF NOT EXISTS billing_seller_payouts (
    id TEXT PRIMARY KEY,
    deal_id TEXT NOT NULL,
    invoice_id TEXT NOT NULL,
    auction_id TEXT NOT NULL,
    seller_company_id TEXT NOT NULL,
    buyer_company_id TEXT NOT NULL,
    amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    ready_at TIMESTAMPTZ NULL,
    paid_at TIMESTAMPTZ NULL,
    cancelled_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,

    CONSTRAINT uq_billing_seller_payouts_deal UNIQUE (deal_id),
    CONSTRAINT uq_billing_seller_payouts_invoice UNIQUE (invoice_id),

    CHECK (amount > 0),
    CHECK (currency = 'RUB'),
    CHECK (buyer_company_id <> seller_company_id),
    CHECK (status IN ('PENDING', 'READY', 'PAID', 'CANCELLED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_billing_seller_payouts_seller
ON billing_seller_payouts(seller_company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_seller_payouts_status
ON billing_seller_payouts(status, created_at ASC);
