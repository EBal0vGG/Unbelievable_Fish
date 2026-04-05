CREATE TABLE IF NOT EXISTS trading_auction_winners (
    auction_id TEXT NOT NULL,
    place INT NOT NULL,
    company_id TEXT NOT NULL,
    amount BIGINT NOT NULL,
    placed_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (auction_id, place)
);
