#!/usr/bin/env bash
set -euo pipefail

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

fish_resp="$(curl -s -X POST "$CATALOG_URL/fish" -H "Content-Type: application/json" -d '{"name":"Cod","description":"desc"}')"
fish_id="$(json_get "$fish_resp" "fish_id")"

product_resp="$(curl -s -X POST "$CATALOG_URL/products" -H "Content-Type: application/json" -d "{\"fish_id\":\"$fish_id\",\"weight\":10,\"unit\":\"kg\",\"size\":\"M\",\"processing_type\":\"frozen\"}")"
product_id="$(json_get "$product_resp" "product_id")"

curl -s -X POST "$CATALOG_URL/products/$product_id/publish" >/dev/null

starts_at="$(date -u -d "-1 min" +%Y-%m-%dT%H:%M:%SZ)"
lot_resp="$(curl -s -X POST "$CATALOG_URL/lots" -H "Content-Type: application/json" -H "X-Company-ID: seller-1" -d "{\"product_id\":\"$product_id\",\"photo\":\"photo\",\"quantity\":10,\"start_price\":100,\"auction_starts_at\":\"$starts_at\",\"auction_duration_minutes\":2}")"
lot_id="$(json_get "$lot_resp" "lot_id")"

curl -s -X POST "$CATALOG_URL/lots/$lot_id/publish" >/dev/null

sleep 2
auction_id="$(docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "select auction_id from trading_auctions where lot_id = '$lot_id' order by starts_at desc limit 1;")"
auction_id="$(echo "$auction_id" | tr -d '[:space:]')"
if [[ -z "$auction_id" ]]; then
  echo "Auction not found for lot: $lot_id" >&2
  exit 1
fi

curl -s -o /dev/null -w "%{http_code}\n" -X POST "$TRADING_URL/auctions/$auction_id/bids" -H "Content-Type: application/json" -H "X-Company-ID: buyer-1" -H "X-User-ID: user-1" -d '{"amount":120}' | grep -q "202"
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$TRADING_URL/auctions/$auction_id/bids" -H "Content-Type: application/json" -H "X-Company-ID: buyer-2" -H "X-User-ID: user-2" -d '{"amount":150}' | grep -q "202"

PGHOST="localhost" PGUSER="$PGUSER" PGPASSWORD="$PGPASSWORD" PGDATABASE="$PGDATABASE" PGPORT="$PGPORT" PGSSLMODE="$PGSSLMODE" AUCTION_ID="$auction_id" go run ./cmd/admin close-auction >/dev/null

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select auction_id, state from trading_auctions where auction_id = '$auction_id';"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select auction_id, status, current_index, deal_id from deal_winner_selections where auction_id = '$auction_id';"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select deal_id, customer_id, status from deals where auction_id = '$auction_id';"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select lot_id, status, auction_id, final_price from catalog_lots where lot_id = '$lot_id';"

echo "OK: fish_id=$fish_id product_id=$product_id lot_id=$lot_id auction_id=$auction_id"

if [[ "$STOP_COMPOSE" == "1" ]]; then
  docker compose down
fi
