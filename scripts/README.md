# Demo Scripts

## Prerequisites
- Docker and Docker Compose
- curl
- python3
- bash
- Go toolchain (for `go run ./cmd/admin` used by scripts)

## How to bring the system up
From repo root:
- `docker compose up -d --build`

To stop:
- `docker compose down`

The scripts can also manage this for you if you set `START_COMPOSE=1` and/or `STOP_COMPOSE=1`.

## Environment variables
Common defaults (you can override as needed):
- `CATALOG_URL` (default `http://localhost:8081`)
- `TRADING_URL` (default `http://localhost:8082`)
- `PGUSER` (default `fish`)
- `PGPASSWORD` (default `fish`)
- `PGDATABASE` (default `fish`)
- `PGPORT` (default `5433`)
- `PGSSLMODE` (default `disable`)
- `PG_CONTAINER` (optional, defaults to `docker compose ps -q postgres`)

Compose control:
- `START_COMPOSE=1` to run `docker compose up -d --build` before the demo.
- `STOP_COMPOSE=1` to run `docker compose down` after the demo.
 - `RESET_DB=1` to run `docker compose down -v` before the demo (fresh DB).
Logging:
- `LOG_FILE=/path/to/log.txt` to tee all output to a file.
- `VERBOSE=1` to show full command traces (default is quiet).

Admin idempotency:
- `DEAL_ID` is used by `demo_fallback_winner.sh` to make `decline-deal` idempotent.

## Scripts and what they demonstrate

### `scripts/demo_happy_path.sh`
Flow:
- Create fish → create product → publish product → create lot → publish lot
- Place bids
- Close auction via admin CLI

Expected effects:
- HTTP: all commands return `202`.
- DB: `trading_auctions` state becomes `WON`.
- DB: `deal_winner_selections` created with `current_index=0` and `deal_id` set.
- DB: `deals` row created for the top bidder.
- DB: `catalog_lots` set to `CLOSED` with `final_price`.

### `scripts/demo_fallback_winner.sh`
Flow:
- Same setup and bids as happy path
- Close auction
- Decline the first deal (`DEAL_ID` taken from selection)

Expected effects:
- DB: `deal_winner_selections.current_index` increments to `1`.
- DB: a second deal is created for the next bidder.

### `scripts/demo_auto_close.sh`
Flow:
- Create fish/product/lot and publish
- No bids are placed
- Wait for scheduler to auto-close expired auction

Expected effects:
- DB: `trading_auctions` state becomes `CANCELLED`.
- DB: `catalog_lots` status becomes `CANCELLED`.

### `scripts/demo_stress.sh`
Flow:
- Create multiple lots and publish them
- Resolve multiple auctions
- Place multiple bids per auction
- Send invalid bids to test rejection paths
- Close all auctions via admin CLI
- Try post-close bids to ensure rejection

Expected effects:
- DB: multiple auctions transition to `WON`
- DB: deals created for top bidders (via integration relay + `trading.AuctionWon` in `outbox_messages`)
- HTTP: invalid bids return non-202 statuses

Config:
- `LOT_COUNT` (default `10`)
- `BIDS_PER_AUCTION` (default `20`)
- `INVALID_BIDS` (default `5`)
- `POST_CLOSE_BIDS` (default `5`)
- `CONCURRENT_BIDS` (default `1`, set to `0` to disable parallel bids)
- `CURL_TIMEOUT` (default `30` seconds)
- `CONCURRENT_LIMIT`, `WAVE_SLEEP_MS` (throttle concurrent bid batches)
- `DEAL_RELAY_MAX_WAIT_SEC` (default `90`): poll until this run’s `WON` auctions all have `deals` rows
- `DEAL_RELAY_POLL_INTERVAL` (default `1` second)
- `DEAL_RELAY_STRICT=1`: fail if deals are still missing after max wait

### `scripts/e2e_bid_race_extension.sh`
Flow:
- Registers seller + two buyers in Identity (real auth flow with JWT).
- Creates fish/product/lot and publishes lot.
- Waits for async auction creation (`GET /auctions/by-lot/{lot_id}` polling).
- Sends two near-concurrent bids: `110` and `120`.
- Verifies race result and extension behavior via direct SQL checks.
- Verifies bid validation:
  - bid below min step -> `400`
  - bid after `ends_at` -> `409`
  - bid outside extension window does not change `ends_at`

Expected output:
- Per-check `PASS/FAIL` lines.
- Final summary: `SUMMARY: PASS=<N> FAIL=<M>`.
- Script exits non-zero if any check failed.

## How to run each script
From repo root:
- `./scripts/demo_happy_path.sh`
- `./scripts/demo_fallback_winner.sh`
- `./scripts/demo_auto_close.sh`
- `./scripts/demo_stress.sh`
- `./scripts/e2e_bid_race_extension.sh`

Optional: let scripts manage compose:
- `START_COMPOSE=1 STOP_COMPOSE=1 ./scripts/demo_happy_path.sh`

## Logs and checks
- Admin command logs print to stdout in each script run.
- Scheduler logs are emitted by the `integration` service container.
- For DB checks, each script prints the SQL query results.

### Sample SQL checks
You can run common checks with:
- `psql -U fish -d fish -f scripts/sql_checks.sql`
