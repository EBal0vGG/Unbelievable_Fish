CREATE TABLE IF NOT EXISTS trading_bids (
    auction_id TEXT NOT NULL,
    bidder_company_id TEXT NOT NULL,
    amount BIGINT NOT NULL,
    placed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_trading_bids_auction_id
    ON trading_bids (auction_id);

CREATE INDEX IF NOT EXISTS idx_trading_bids_order
    ON trading_bids (auction_id, amount DESC, placed_at ASC);
