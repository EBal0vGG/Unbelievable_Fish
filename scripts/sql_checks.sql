-- Replace placeholders before running.
-- Example: psql -U fish -d fish -f scripts/sql_checks.sql

-- Latest auction state
SELECT auction_id, lot_id, state, starts_at, ends_at, current_price
FROM trading_auctions
ORDER BY starts_at DESC
LIMIT 5;

-- Winner selection state
SELECT auction_id, status, current_index, deal_id
FROM deal_winner_selections
ORDER BY won_at DESC
LIMIT 5;

-- Recent deals
SELECT deal_id, auction_id, customer_id, status
FROM deals
ORDER BY created_at DESC
LIMIT 5;

-- Catalog lots
SELECT lot_id, status, auction_id, final_price
FROM catalog_lots
ORDER BY auction_starts_at DESC
LIMIT 5;
