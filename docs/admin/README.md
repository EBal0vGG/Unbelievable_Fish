# Admin CLI

The admin CLI is a narrow, non-public interface for internal operations.

## Commands

### Close an auction
```
PGHOST=localhost PGUSER=fish PGPASSWORD=fish PGDATABASE=fish PGPORT=5433 \
AUCTION_ID=<auction_id> \
go run ./cmd/admin close-auction
```

### Cancel a deal (buyer forfeits / rejects as winner — normal demo path)
Emits `DealCancelled` and (for `WINNER_REJECTED`) `WinnerRejected`; integration advances the winner selection on `DealCancelled`.
```
PGHOST=localhost PGUSER=fish PGPASSWORD=fish PGDATABASE=fish PGPORT=5433 \
DEAL_ID=<deal_id> \
COMPANY_ID=<deal_customer_company_id> \
go run ./cmd/admin cancel-deal
```
Optional: `CANCEL_REASON` (default `WINNER_REJECTED`), `USER_ID`, `CORRELATION_ID`, `CAUSATION_ID`.

### Decline a deal (advance selection — deal must already be cancelled)
Low-level `HandleDealDeclined` call. Prefer `cancel-deal` for demos unless you are replaying integration after a cancel.
```
PGHOST=localhost PGUSER=fish PGPASSWORD=fish PGDATABASE=fish PGPORT=5433 \
AUCTION_ID=<auction_id> \
DEAL_ID=<current_deal_id> \
go run ./cmd/admin decline-deal
```

Notes:
- `DEAL_ID` is optional but recommended for idempotency on `decline-deal`. If provided and it does not match the current selection, the command becomes a no-op.

## Required env
- `PGHOST`, `PGUSER`, `PGDATABASE` (and `PGPASSWORD` if needed)
- `close-auction` / `decline-deal`: `AUCTION_ID`
- `cancel-deal`: `DEAL_ID`, `COMPANY_ID` (must be the deal’s customer / winning buyer when using `WINNER_REJECTED`)

## Optional env
- `PGPORT` (default `5432`)
- `PGSSLMODE` (default `disable`)
- `COMPANY_ID`, `USER_ID`, `CORRELATION_ID`, `CAUSATION_ID` (for `close-auction` / `decline-deal` command meta)
- `DEAL_ID` (decline-deal only)
- `CANCEL_REASON` (cancel-deal only)
