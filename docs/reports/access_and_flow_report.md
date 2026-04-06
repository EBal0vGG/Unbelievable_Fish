# Access & Flow Report

## 1) Main files are bloated

### `cmd/catalog/main.go`
- Responsibilities:
  - HTTP routing for `/fish`, `/products`, `/products/{id}/publish`, `/lots`, `/lots/{id}/assign-auction`, `/lots/{id}/publish` (handlers embedded in file).
  - Service construction with mixed persistence: Postgres for lots/outbox + in‑memory repos for fish/products/units/processing types.
  - DB connection and env parsing.
- What should be extracted:
  - HTTP handlers into `internal/catalog/http` (request/response DTO + error mapping).
  - Service wiring into a `composition root` package.
  - Config/DB connection into a shared `infra/config` or `infra/db` package.

### `cmd/trading/main.go`
- Responsibilities:
  - Full use case wiring for Create/Publish/PlaceBid/Close/Cancel.
  - HTTP router binding.
  - DB connection/env parsing.
- What should be extracted:
  - Composition/wiring into `internal/trading/compose` (or similar).
  - Shared DB config.

### `cmd/deals/main.go`
- Responsibilities:
  - All deal use cases + router wiring.
  - DB connection/env parsing.
- What should be extracted:
  - Composition/wiring package.
  - Shared DB config.

### `cmd/chain_runner/main.go`
- Responsibilities:
  - Migrations, truncation, chain orchestration, outbox relay wiring.
  - In‑memory repos + creation of catalog data.
  - Deal creation verification.
- Intended use: local verification tool (not production).
- Extraction: move migration/truncate helpers to `internal/infra/db` or `internal/infra/migrate`.

### `cmd/migrate/main.go`
- Responsibility: apply all SQL migrations.
- OK to keep small, but can reuse shared migration helper used by chain runner.

## 2) Public vs internal access (per your matrix)

### Trading (Auction)
**Should be PUBLIC**  
- PlaceBid only

**Should be INTERNAL (event‑driven / system)**  
- CreateAuction, PublishAuction, CloseAuction, CancelAuction

**Current exposure (violations)**  
Trading HTTP router exposes all of these:
- `POST /auctions` → CreateAuction
- `POST /auctions/{id}/publish` → PublishAuction
- `POST /auctions/{id}/close` → CloseAuction
- `POST /auctions/{id}/cancel` → CancelAuction
These endpoints are reachable by any HTTP client now.  
Source: `internal/trading/http/router.go`, `cmd/trading/main.go`.

### Catalog
**Should be PUBLIC**  
- CreateFish / UpdateFish
- CreateProduct / UpdateProduct
- PublishProduct / UnpublishProduct
- CreateLot
- PublishLot / UnpublishLot

**Should be INTERNAL (event‑driven)**  
- AssignAuctionID
- HandleAuctionWon / HandleBidPlaced / HandleAuctionClosed / HandleAuctionCancelled

**Current exposure**  
`cmd/catalog/main.go` exposes:
- `/lots/{id}/assign-auction` (INTERNAL but currently public).
This violates the boundary: only the system should assign auction IDs after LotPublished.

### Deals
**Should be PUBLIC**  
- GetDealByID / GetDealByAuctionID
- ConfirmDeal / PrepareContract / SignContract
- RequestPayment / MarkDealPaid / RequestShipment / MarkDealShipped
- CompleteDeal / CancelDeal / UpdateDealPrice

**Should be INTERNAL (event‑driven)**  
- CreateProjection
- CreateDealFromAuctionWon / CreateDealSelectionFromAuctionWon
- HandleDealDeclined

**Current exposure (violations)**  
Deals HTTP router exposes:
- `POST /deal-projections` (internal)
- `POST /deals/from-auction-won` (internal)
Source: `internal/deals/http/router.go`, `cmd/deals/main.go`.

## 3) Scenario walkthroughs (request → response → internal chain → DB changes)

### Chain 1: Catalog publishes `LotPublished` → Trading creates auction → Deals creates projection

**Request (public):**
1) `POST /lots` → CreateLot (Catalog)
2) `POST /lots/{id}/publish` → PublishLot (Catalog)

**Internal flow:**
1) `CatalogService.PublishLot`:
   - Loads lot + product
   - Emits `LotPublished`
   - Saves lot
   - Writes outbox event
2) Outbox relay reads `catalog.LotPublished` and publishes to in‑memory bus.
3) Event handler calls:
   - `Trading.CreateAuction`
   - `Trading.PublishAuction`
   - `Deals.CreateProjection`

**DB changes:**
- `catalog_lots` (insert/updates):
  - Created in `CreateLot`, status `DRAFT`.
  - Updated to `PUBLISHED` in `PublishLot`.
  - Queries + upserts in `internal/catalog/postgres/lot_repository.go`.
- `outbox_messages`:
  - One row for `catalog.LotPublished` inserted in `internal/catalog/postgres/outbox_repository.go`.
- `trading_auctions`:
  - Inserted as `DRAFT` by `CreateAuction`, updated to `PUBLISHED` by `PublishAuction`.
  - Upsert in `internal/trading/postgres/auction_repository.go`.
- `outbox_messages`:
  - `trading.AuctionPublished` inserted by `internal/trading/postgres/outbox_repository.go`.
- `deal_projections`:
  - Inserted/updated by `internal/deals/postgres/projection_repository.go`.

**Observed DB writes (SQL):**
```1:12:internal/catalog/postgres/lot_repository.go
INSERT INTO catalog_lots (...) VALUES (...) ON CONFLICT (lot_id) DO UPDATE ...
```
```1:35:internal/catalog/postgres/outbox_repository.go
INSERT INTO outbox_messages (...) VALUES (...)
```
```21:38:internal/trading/postgres/auction_repository.go
INSERT INTO trading_auctions (...) VALUES (...) ON CONFLICT (auction_id) DO UPDATE ...
```
```21:48:internal/deals/postgres/projection_repository.go
INSERT INTO deal_projections (...) VALUES (...) ON CONFLICT (auction_id) DO UPDATE ...
```

### Chain 2: Auction closes with winners → Deals creates deal

**Request (public):**
- `POST /auctions/{id}/bids` → PlaceBid (Trading)

**Internal flow:**
1) `Trading.PlaceBid`:
   - Loads auction `FOR UPDATE`
   - Validates bid
   - Writes `trading_bids`
   - Updates `trading_auctions`
   - Writes `trading.BidPlaced` to outbox
2) `Trading.CloseAuction` (should be internal/scheduler):
   - Loads auction `FOR UPDATE`
   - Reads top bids from `trading_bids`
   - Calculates winners
   - Writes `trading_auction_winners`
   - Updates `trading_auctions`
   - Writes `trading.AuctionClosed` + `trading.AuctionWon` to outbox
3) Outbox relay publishes `trading.AuctionWon` → `Deals.CreateDealSelectionFromAuctionWon`
4) Deals creates `Deal` and persists winner selection.

**DB changes:**
- `trading_bids`: insert (bid history).
- `trading_auctions`: update current price/leader/state.
- `trading_auction_winners`: delete + insert top 3.
- `outbox_messages`: `trading.BidPlaced`, `trading.AuctionClosed`, `trading.AuctionWon`.
- `deals`: insert/update deal row.
- `deal_winner_selections`: insert/update selection state.
- `deal_projections`: updated to `converted`.

**Observed DB writes (SQL):**
```21:39:internal/trading/postgres/bid_repository.go
INSERT INTO trading_bids (...) VALUES (...)
```
```20:49:internal/trading/postgres/winners_repository.go
DELETE FROM trading_auction_winners WHERE auction_id = $1
INSERT INTO trading_auction_winners (...) VALUES (...)
```
```22:48:internal/deals/postgres/deal_repository.go
INSERT INTO deals (...) VALUES (...) ON CONFLICT (deal_id) DO UPDATE ...
```
```22:41:internal/deals/postgres/selection_repository.go
INSERT INTO deal_winner_selections (...) VALUES (...) ON CONFLICT (auction_id) DO UPDATE ...
```

## 4) Deployment sanity check

**Current deployment layout:**
- `docker-compose.yml` defines:
  - `postgres`, `migrate`, `catalog`, `trading`, `deals`.
  - Env vars (`PGHOST/PGUSER/PGDATABASE/PGPASSWORD/PGPORT/PGSSLMODE`).
  - Exposed ports: `8081` (Catalog), `8082` (Trading), `8083` (Deals).
  - `migrate` is a one‑shot container running `cmd/migrate`.

**Important missing wiring:**
Cross‑service event delivery is not wired outside `chain_runner`.  
The in‑memory bus is single‑process, so in real multi‑service deployment you still need:
- a transport (Kafka/HTTP relay/outbox worker)
- a relay service that reads outbox and publishes to that transport

## 5) Findings summary (discrepancies vs access model)

- Trading exposes internal lifecycle use cases (`CreateAuction`, `PublishAuction`, `CloseAuction`, `CancelAuction`) via HTTP. This violates the intended “event‑driven engine” boundary.
- Catalog exposes `AssignAuctionID` (internal), currently publicly reachable.
- Deals exposes internal event‑driven endpoints (`/deal-projections`, `/deals/from-auction-won`).
- Multi‑service deployment lacks a real cross‑service event transport; chains only truly work in single‑process (`chain_runner`).

## 6) Recommended access cleanup (by your matrix)

- **Trading HTTP**: leave only `POST /auctions/{id}/bids`; remove/guard all other endpoints.
- **Catalog HTTP**: keep CRUD + publish/unpublish; remove `assign-auction` from public.
- **Deals HTTP**: keep read + user lifecycle actions; remove `create-projection` and `from-auction-won` endpoints.

