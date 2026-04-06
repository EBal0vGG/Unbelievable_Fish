## Local Chain Verification + Deployment

### Prereqs
- Postgres reachable via env: `PGHOST`, `PGUSER`, `PGDATABASE` (optional `PGPASSWORD`, `PGPORT`, `PGSSLMODE`).
- Go installed.

### Verify the chains locally (script)
This runs both chains in a single process using the outbox relay + in-memory bus.

```
PGHOST=localhost PGUSER=fish PGDATABASE=fish PGPASSWORD=fish PGPORT=5432 PGSSLMODE=disable \
go run ./cmd/chain_runner
```

Expected output: `chains verified` and a deal created.

### Docker Compose (3 services + Postgres)

Start Postgres and run migrations:
```
docker compose up -d postgres
docker compose run --rm migrate
```

Start services:
```
docker compose up -d catalog trading deals
```

Ports:
- Catalog: `8081`
- Trading: `8082`
- Deals: `8083`

### Trigger the chains (HTTP)
Catalog service exposes minimal endpoints for local flow:

1) Create fish:
```
curl -X POST localhost:8081/fish -d '{"name":"Fish","description":"Fish"}'
```

2) Create product:
```
curl -X POST localhost:8081/products -d '{"fish_id":"<FISH_ID>","weight":1.5,"unit":"kg","size":"M","processing_type":"frozen"}'
```

3) Publish product:
```
curl -X POST localhost:8081/products/<PRODUCT_ID>/publish
```

4) Create lot (requires `X-Company-ID`):
```
curl -X POST localhost:8081/lots \
  -H "X-Company-ID: seller-1" \
  -d '{"product_id":"<PRODUCT_ID>","photo":"photo","quantity":10,"start_price":100,"auction_starts_at":"2026-01-01T10:00:00Z"}'
```

5) Assign auction ID:
```
curl -X POST localhost:8081/lots/<LOT_ID>/assign-auction -d '{"auction_id":"auc-1"}'
```

6) Publish lot:
```
curl -X POST localhost:8081/lots/<LOT_ID>/publish
```

7) Place bid + close auction:
```
curl -X POST localhost:8082/auctions/auc-1/bids \
  -H "X-Company-ID: buyer-1" \
  -H "X-User-ID: buyer-1" \
  -d '{"amount":150}'

curl -X POST localhost:8082/auctions/auc-1/close \
  -H "X-Company-ID: buyer-1" \
  -H "X-User-ID: buyer-1"
```

### Note on event delivery
Current cross‑domain event delivery uses the in‑memory bus in a **single process** (the `chain_runner` script). Docker Compose wiring is a deployment skeleton; cross‑container eventing will need a transport (Kafka / HTTP relay / outbox worker) to propagate events between services.
