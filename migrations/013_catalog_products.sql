CREATE TABLE IF NOT EXISTS catalog_products (
    product_id TEXT PRIMARY KEY,
    fish_id TEXT NOT NULL,
    weight DOUBLE PRECISION NOT NULL,
    unit TEXT NOT NULL,
    size TEXT NOT NULL DEFAULT '',
    processing_type TEXT NOT NULL,
    status TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_catalog_products_fish_id
    ON catalog_products (fish_id);
