CREATE TABLE IF NOT EXISTS deal_confirmations (
    confirmation_id TEXT PRIMARY KEY,
    deal_id TEXT NOT NULL REFERENCES deals(deal_id) ON DELETE CASCADE,
    stage TEXT NOT NULL,
    requested_by_user_id TEXT NOT NULL,
    requested_by_company_id TEXT NOT NULL,
    counterparty_company_id TEXT NOT NULL,
    status TEXT NOT NULL,
    verification_method TEXT NOT NULL,
    verification_token_hash TEXT NULL,
    signature_ref TEXT NULL,
    requested_at TIMESTAMPTZ NOT NULL,
    approved_at TIMESTAMPTZ NULL,
    rejected_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NULL,
    comment TEXT NULL,
    reason TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_deal_confirmations_deal_id
    ON deal_confirmations (deal_id);

CREATE INDEX IF NOT EXISTS idx_deal_confirmations_stage_status
    ON deal_confirmations (deal_id, stage, status);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_deal_confirmations_pending_stage
    ON deal_confirmations (deal_id, stage)
    WHERE status = 'pending';
