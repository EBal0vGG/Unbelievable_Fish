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
- `DEALS_URL` (default `http://localhost:8083`)
- `IDENTITY_URL` (default `http://localhost:8084`)
- `BILLING_URL` (default `http://localhost:8085/billing` — include `/billing` prefix)
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
- `DEAL_ID` / `COMPANY_ID` for `cmd/admin cancel-deal` are derived inside `demo_fallback_winner.sh` from the DB (winning buyer cancels with `WINNER_REJECTED`).

## Scripts and what they demonstrate

### `scripts/demo_full_payment_flow.sh`
Flow:
- Same catalog/trading setup as happy path (fish → product → lot → publish → bids → `cmd/admin close-auction`)
- Billing test top-ups for bidders, full **deals** confirmation + contract path to `payment/request`
- `GET` deal invoice, `POST` **fake-confirm** invoice (requires `BILLING_ENABLE_FAKE_PROVIDER` on billing)
- Wait for relay: deal **`paid`**, seller payout **`PENDING`**
- Promote a dedicated ops user to **admin** (SQL; self-registration as admin is forbidden), then `POST /billing/admin/payouts/{id}/ready` and `/paid` (requires `BILLING_ENABLE_ADMIN_ACTIONS`)
- Assert seller **`available`** equals invoice **`goods_amount`** and exactly one ledger row **`SELLER_PAYOUT_CREDITED`**; second `/paid` is idempotent

Expected effects:
- `billing_seller_payouts.status`: `PENDING` → `READY` → `PAID`
- `billing_accounts.available` for seller matches goods amount only after **`PAID`**

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
- Same setup and bids as happy path (including billing test top-ups)
- Close auction
- Cancel the first winning deal as the deal customer via `cmd/admin cancel-deal` (`CANCEL_REASON` defaults to `WINNER_REJECTED`); integration handles `deals.DealCancelled` and advances the winner selection

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

## Realtime endpoint scripts

For live demos where you need to call one endpoint at a time, use:
- `scripts/realtime/` (see `scripts/realtime/README.md`)

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
