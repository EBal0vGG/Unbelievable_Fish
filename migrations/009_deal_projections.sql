CREATE TABLE IF NOT EXISTS deal_projections (
    auction_id TEXT PRIMARY KEY,
    supplier_id TEXT NOT NULL,
    start_price BIGINT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL,
    product_id TEXT NOT NULL,
    product_name TEXT NOT NULL,
    product_description TEXT NOT NULL,
    product_category TEXT NOT NULL,
    product_weight DOUBLE PRECISION NOT NULL,
    product_volume DOUBLE PRECISION NOT NULL,
    product_origin_country TEXT NOT NULL
);
