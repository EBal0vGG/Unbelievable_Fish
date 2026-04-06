CREATE TABLE IF NOT EXISTS deals (
    deal_id TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    supplier_id TEXT NOT NULL,
    auction_id TEXT NOT NULL,
    quantity BIGINT NOT NULL,
    unit_price BIGINT NOT NULL,
    status TEXT NOT NULL,
    type_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    confirmed_at TIMESTAMPTZ NULL,
    contract_number TEXT NULL,
    contract_prepared_at TIMESTAMPTZ NULL,
    contract_signed_at TIMESTAMPTZ NULL,
    contract_signed_by TEXT NULL,
    signature_ref TEXT NULL,
    document_url TEXT NULL,
    product_id TEXT NOT NULL,
    product_name TEXT NOT NULL,
    product_description TEXT NOT NULL,
    product_category TEXT NOT NULL,
    product_weight DOUBLE PRECISION NOT NULL,
    product_volume DOUBLE PRECISION NOT NULL,
    product_origin_country TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_deals_auction_id
    ON deals (auction_id);
