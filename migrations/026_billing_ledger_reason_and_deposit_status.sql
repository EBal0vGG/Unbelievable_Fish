ALTER TABLE billing_ledger_entries
    ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT '';

ALTER TABLE billing_auction_deposits
    DROP CONSTRAINT IF EXISTS billing_auction_deposits_status_check;

ALTER TABLE billing_auction_deposits
    ADD CONSTRAINT billing_auction_deposits_status_check
    CHECK (status IN ('HELD', 'RELEASED', 'CAPTURED', 'SETTLED'));
