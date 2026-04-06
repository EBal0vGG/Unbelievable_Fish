# Admin CLI

The admin CLI is a narrow, non-public interface for internal operations.

## Commands

### Close an auction
```
PGHOST=localhost PGUSER=fish PGPASSWORD=fish PGDATABASE=fish PGPORT=5433 \
AUCTION_ID=<auction_id> \
go run ./cmd/admin close-auction
```

### Decline a deal (advance to next winner)
```
PGHOST=localhost PGUSER=fish PGPASSWORD=fish PGDATABASE=fish PGPORT=5433 \
AUCTION_ID=<auction_id> \
DEAL_ID=<current_deal_id> \
go run ./cmd/admin decline-deal
```

Notes:
- `DEAL_ID` is optional but recommended for idempotency. If provided and it does not match the current selection, the command becomes a no-op.

## Required env
- `PGHOST`, `PGUSER`, `PGDATABASE` (and `PGPASSWORD` if needed)
- `AUCTION_ID`

## Optional env
- `PGPORT` (default `5432`)
- `PGSSLMODE` (default `disable`)
- `COMPANY_ID`, `USER_ID`, `CORRELATION_ID`, `CAUSATION_ID`
- `DEAL_ID` (decline-deal only)
