ALTER TABLE catalog_products
ADD COLUMN IF NOT EXISTS seller_company_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_catalog_products_seller_company
    ON catalog_products (seller_company_id);
