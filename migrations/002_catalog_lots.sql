CREATE TABLE IF NOT EXISTS catalog_lots (
    lot_id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    auction_id TEXT NULL,
    seller_company_id TEXT NOT NULL,
    photo TEXT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    start_price BIGINT NOT NULL,
    cur_price BIGINT NOT NULL,
    final_price BIGINT NOT NULL,
    status TEXT NOT NULL,
    auction_starts_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_catalog_lots_auction_id
    ON catalog_lots (auction_id)
    WHERE auction_id IS NOT NULL;
