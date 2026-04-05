CREATE TABLE IF NOT EXISTS deal_winner_selections (
    auction_id TEXT PRIMARY KEY,
    candidates JSONB NOT NULL,
    current_index INT NOT NULL,
    status TEXT NOT NULL,
    final_price BIGINT NOT NULL,
    won_at TIMESTAMPTZ NOT NULL,
    supplier_id TEXT NOT NULL,
    deal_id TEXT NULL,
    product_id TEXT NOT NULL,
    product_name TEXT NOT NULL,
    product_description TEXT NOT NULL,
    product_category TEXT NOT NULL,
    product_weight DOUBLE PRECISION NOT NULL,
    product_volume DOUBLE PRECISION NOT NULL,
    product_origin_country TEXT NOT NULL
);
