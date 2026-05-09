-- Ensure start_price is positive for deposit calculation (5% rule).
UPDATE trading_auctions
SET start_price = current_price
WHERE start_price IS NULL OR start_price <= 0;

ALTER TABLE trading_auctions DROP CONSTRAINT IF EXISTS trading_auctions_start_price_positive;

ALTER TABLE trading_auctions
    ADD CONSTRAINT trading_auctions_start_price_positive
    CHECK (start_price > 0);
