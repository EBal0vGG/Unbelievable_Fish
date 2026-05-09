-- One DealInvoice per deal: named UNIQUE constraint (single btree on deal_id).
CREATE TABLE IF NOT EXISTS billing_deal_invoices (
    id TEXT PRIMARY KEY,
    deal_id TEXT NOT NULL,
    auction_id TEXT NOT NULL DEFAULT '',
    buyer_company_id TEXT NOT NULL,
    seller_company_id TEXT NOT NULL,
    goods_amount BIGINT NOT NULL,
    platform_fee_due_amount BIGINT NOT NULL,
    total_amount BIGINT NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_invoice_id TEXT NOT NULL DEFAULT '',
    payment_url TEXT NOT NULL DEFAULT '',
    due_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ NULL,
    expired_at TIMESTAMPTZ NULL,
    cancelled_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,

    CONSTRAINT uq_billing_deal_invoices_deal UNIQUE (deal_id),

    CHECK (goods_amount > 0),
    CHECK (platform_fee_due_amount >= 0),
    CHECK (total_amount = goods_amount + platform_fee_due_amount),
    CHECK (currency = 'RUB'),
    CHECK (buyer_company_id <> seller_company_id),
    CHECK (status IN ('PENDING', 'PAYMENT_PENDING', 'PAID', 'EXPIRED', 'CANCELLED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_billing_deal_invoices_buyer
ON billing_deal_invoices(buyer_company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_deal_invoices_seller
ON billing_deal_invoices(seller_company_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_billing_deal_invoices_due
ON billing_deal_invoices(status, due_at);

CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_deal_invoices_provider
ON billing_deal_invoices(provider, provider_invoice_id)
WHERE provider_invoice_id <> '';
