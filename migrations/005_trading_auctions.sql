CREATE TABLE IF NOT EXISTS trading_auctions (
    auction_id TEXT PRIMARY KEY,
    lot_id TEXT NOT NULL,
    state TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    current_price BIGINT NOT NULL,
    leader_company_id TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trading_auctions_lot_id
    ON trading_auctions (lot_id);
