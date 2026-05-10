ALTER TABLE trading_bids
    ADD COLUMN IF NOT EXISTS chain_bid_hash TEXT,
    ADD COLUMN IF NOT EXISTS chain_tx_hash TEXT,
    ADD COLUMN IF NOT EXISTS chain_status TEXT NOT NULL DEFAULT 'NOT_ANCHORED',
    ADD COLUMN IF NOT EXISTS chain_wallet_address TEXT,
    ADD COLUMN IF NOT EXISTS chain_block_number BIGINT;

CREATE INDEX IF NOT EXISTS idx_trading_bids_chain_status
    ON trading_bids (chain_status);

CREATE UNIQUE INDEX IF NOT EXISTS uq_trading_bids_auction_chain_bid_hash
    ON trading_bids (auction_id, chain_bid_hash)
    WHERE chain_bid_hash IS NOT NULL;

ALTER TABLE trading_auctions
    ADD COLUMN IF NOT EXISTS chain_result_hash TEXT,
    ADD COLUMN IF NOT EXISTS chain_finalize_tx_hash TEXT,
    ADD COLUMN IF NOT EXISTS chain_finalize_status TEXT NOT NULL DEFAULT 'NOT_FINALIZED',
    ADD COLUMN IF NOT EXISTS chain_finalize_wallet_address TEXT,
    ADD COLUMN IF NOT EXISTS chain_finalize_block_number BIGINT;

CREATE INDEX IF NOT EXISTS idx_trading_auctions_chain_finalize_status
    ON trading_auctions (chain_finalize_status);

CREATE TABLE IF NOT EXISTS trading_chain_operations (
    op_id BIGSERIAL PRIMARY KEY,
    op_type TEXT NOT NULL,
    op_nonce BIGINT NOT NULL DEFAULT 0,
    auction_id TEXT NOT NULL,
    auction_ref_hash TEXT NOT NULL,
    bid_hash TEXT,
    placed_at TIMESTAMPTZ,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    min_bid_step BIGINT,
    result_hash TEXT,
    winner_company_id TEXT,
    final_price BIGINT,
    wallet_address TEXT,
    status TEXT NOT NULL,
    tx_hash TEXT,
    block_number BIGINT,
    last_error TEXT,
    attempt_count INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_trading_chain_operations_auction_nonce
    ON trading_chain_operations (auction_id, op_nonce);

CREATE UNIQUE INDEX IF NOT EXISTS uq_trading_chain_operations_tx_hash
    ON trading_chain_operations (tx_hash)
    WHERE tx_hash IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trading_chain_operations_status_retry
    ON trading_chain_operations (status, next_retry_at, op_id);

CREATE TABLE IF NOT EXISTS trading_chain_sync_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
