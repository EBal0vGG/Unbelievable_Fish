#!/usr/bin/env bash
set -euo pipefail

if [[ "${VERBOSE:-}" == "1" ]]; then
  set -x
fi

LOG_FILE="${LOG_FILE:-}"
if [[ -n "$LOG_FILE" ]]; then
  exec > >(tee -a "$LOG_FILE") 2>&1
fi

banner() {
  echo
  echo "== $1 =="
}

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; exit 1; }
}

require curl
require python3
require docker

START_COMPOSE="${START_COMPOSE:-}"
STOP_COMPOSE="${STOP_COMPOSE:-}"

CATALOG_URL="${CATALOG_URL:-http://localhost:8081}"
TRADING_URL="${TRADING_URL:-http://localhost:8082}"

PGUSER="${PGUSER:-fish}"
PGDATABASE="${PGDATABASE:-fish}"
PGPASSWORD="${PGPASSWORD:-fish}"
PGPORT="${PGPORT:-5433}"
PGSSLMODE="${PGSSLMODE:-disable}"

json_get() {
  python3 -c 'import json,sys; print(json.loads(sys.argv[1])[sys.argv[2]])' "$1" "$2"
}

banner "Compose up (optional)"
if [[ "$START_COMPOSE" == "1" ]]; then
  docker compose up -d --build
fi

PG_CONTAINER="${PG_CONTAINER:-}"
if [[ -z "$PG_CONTAINER" ]]; then
  PG_CONTAINER="$(docker compose ps -q postgres)"
fi
if [[ -z "$PG_CONTAINER" ]]; then
  PG_CONTAINER="fish-postgres-1"
fi

banner "Create fish/product/lot and publish"
fish_resp="$(curl -s -X POST "$CATALOG_URL/fish" -H "Content-Type: application/json" -d '{"name":"Pollock","description":"desc"}')"
fish_id="$(json_get "$fish_resp" "fish_id")"

product_resp="$(curl -s -X POST "$CATALOG_URL/products" -H "Content-Type: application/json" -d "{\"fish_id\":\"$fish_id\",\"weight\":12,\"unit\":\"kg\",\"size\":\"M\",\"processing_type\":\"frozen\"}")"
product_id="$(json_get "$product_resp" "product_id")"

curl -s -X POST "$CATALOG_URL/products/$product_id/publish" >/dev/null

starts_at="$(date -u -d "-2 min" +%Y-%m-%dT%H:%M:%SZ)"
lot_resp="$(curl -s -X POST "$CATALOG_URL/lots" -H "Content-Type: application/json" -H "X-Company-ID: seller-1" -d "{\"product_id\":\"$product_id\",\"photo\":\"photo\",\"quantity\":10,\"start_price\":100,\"auction_starts_at\":\"$starts_at\",\"auction_duration_minutes\":1}")"
lot_id="$(json_get "$lot_resp" "lot_id")"

curl -s -X POST "$CATALOG_URL/lots/$lot_id/publish" >/dev/null

sleep 2
auction_id="$(docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "select auction_id from trading_auctions where lot_id = '$lot_id' order by starts_at desc limit 1;")"
auction_id="$(echo "$auction_id" | tr -d '[:space:]')"
if [[ -z "$auction_id" ]]; then
  echo "Auction not found for lot: $lot_id" >&2
  exit 1
fi

banner "Wait for scheduler auto-close"
state=""
for _ in $(seq 1 20); do
  state="$(docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "select state from trading_auctions where auction_id = '$auction_id';" | tr -d '[:space:]')"
  if [[ "$state" != "PUBLISHED" && -n "$state" ]]; then
    break
  fi
  sleep 3
done

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select auction_id, state, ends_at from trading_auctions where auction_id = '$auction_id';"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select lot_id, status, auction_id, final_price from catalog_lots where lot_id = '$lot_id';"

if [[ "$state" == "PUBLISHED" || -z "$state" ]]; then
  echo "Auto-close did not trigger in time" >&2
  docker logs --tail 50 fish-integration-1 || true
  exit 1
fi

echo "OK: fish_id=$fish_id product_id=$product_id lot_id=$lot_id auction_id=$auction_id state=$state"

banner "Compose down (optional)"
if [[ "$STOP_COMPOSE" == "1" ]]; then
  docker compose down
fi
