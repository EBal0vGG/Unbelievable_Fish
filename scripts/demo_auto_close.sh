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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/catalog_demo_fish.sh"

START_COMPOSE="${START_COMPOSE:-}"
STOP_COMPOSE="${STOP_COMPOSE:-}"

CATALOG_URL="${CATALOG_URL:-http://localhost:8081}"
TRADING_URL="${TRADING_URL:-http://localhost:8082}"
IDENTITY_URL="${IDENTITY_URL:-http://localhost:8084}"

PGUSER="${PGUSER:-fish}"
PGDATABASE="${PGDATABASE:-fish}"
PGPASSWORD="${PGPASSWORD:-fish}"
PGPORT="${PGPORT:-5433}"
PGSSLMODE="${PGSSLMODE:-disable}"

json_get() {
  python3 -c 'import json,sys; print(json.loads(sys.argv[1])[sys.argv[2]])' "$1" "$2"
}

valid_requisites() {
  python3 -c 'import sys,random
seed=sys.argv[1]
r=random.Random(seed)
inn_base=[r.randint(0,9) for _ in range(9)]
weights=[2,4,10,3,5,9,4,6,8]
checksum=(sum(a*b for a,b in zip(inn_base,weights))%11)%10
inn="".join(map(str,inn_base+[checksum]))
ogrn_base=[1]+[r.randint(0,9) for _ in range(11)]
base=int("".join(map(str,ogrn_base)))
ogrn=str(base)+str((base%11)%10)
print(inn,ogrn)' "$1"
}

register_company() {
  local name="$1"
  local inn="$2"
  local ogrn="$3"
  local resp
  resp="$(curl -fsS -X POST "$IDENTITY_URL/companies" -H "Content-Type: application/json" -d "{\"name\":\"$name\",\"inn\":\"$inn\",\"ogrn\":\"$ogrn\"}")"
  json_get "$resp" "id"
}

register_user() {
  local company_id="$1"
  local name="$2"
  local role="$3"
  local login="$4"
  local password="$5"
  curl -fsS -X POST "$IDENTITY_URL/users" -H "Content-Type: application/json" -d "{\"company_id\":\"$company_id\",\"name\":\"$name\",\"role\":\"$role\",\"login\":\"$login\",\"password\":\"$password\",\"accepted_terms\":true,\"terms_version\":\"2026-04-24\"}" >/dev/null
}

login_token() {
  local login="$1"
  local password="$2"
  local resp
  resp="$(curl -fsS -X POST "$IDENTITY_URL/auth/login" -H "Content-Type: application/json" -d "{\"login\":\"$login\",\"password\":\"$password\"}")"
  json_get "$resp" "token"
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
if [[ -z "$(docker compose ps -q integration)" ]]; then
  echo "Integration service is not running. Start it: docker compose up -d integration" >&2
  exit 1
fi

banner "Create fish/product/lot and publish"
fish_id="$(catalog_demo_fish_id "$CATALOG_URL")"

suffix="$(date +%s)"
seller_creds="$(valid_requisites "auto-seller-$suffix")"
seller_company_id="$(register_company "Auto Seller $suffix" "$(echo "$seller_creds" | awk '{print $1}')" "$(echo "$seller_creds" | awk '{print $2}')")"
seller_login="auto.seller.$suffix@example.com"
password="secret123"
register_user "$seller_company_id" "Auto Seller" "seller" "$seller_login" "$password"
seller_token="$(login_token "$seller_login" "$password")"

product_resp="$(curl -fsS -X POST "$CATALOG_URL/products" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $seller_token" \
  -d "{\"fish_id\":\"$fish_id\",\"weight\":12,\"unit\":\"kg\",\"size\":\"M\",\"processing_type\":\"frozen\"}")"
product_id="$(json_get "$product_resp" "product_id")"

curl -fsS -X POST "$CATALOG_URL/products/$product_id/publish" -H "Authorization: Bearer $seller_token" >/dev/null

if command -v gdate >/dev/null 2>&1; then
  starts_at="$(gdate -u -d "-2 min" +%Y-%m-%dT%H:%M:%SZ)"
elif date -u -d "-2 min" +%Y-%m-%dT%H:%M:%SZ >/dev/null 2>&1; then
  starts_at="$(date -u -d "-2 min" +%Y-%m-%dT%H:%M:%SZ)"
else
  starts_at="$(date -u -v-2M +%Y-%m-%dT%H:%M:%SZ)"
fi
lot_resp="$(curl -fsS -X POST "$CATALOG_URL/lots" -H "Content-Type: application/json" -H "Authorization: Bearer $seller_token" -d "{\"product_id\":\"$product_id\",\"photo\":\"photo\",\"quantity\":10,\"start_price\":100,\"auction_starts_at\":\"$starts_at\",\"auction_duration_minutes\":1}")"
lot_id="$(json_get "$lot_resp" "lot_id")"

curl -fsS -X POST "$CATALOG_URL/lots/$lot_id/publish" -H "Authorization: Bearer $seller_token" >/dev/null

auction_id=""
for _ in $(seq 1 20); do
  auction_id="$(docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "select auction_id from catalog_lots where lot_id = '$lot_id' and auction_id is not null limit 1;")"
  auction_id="$(echo "$auction_id" | tr -d '[:space:]')"
  if [[ -n "$auction_id" ]]; then
    break
  fi
  sleep 1
done
if [[ -z "$auction_id" ]]; then
  echo "Auction not found for lot: $lot_id (check integration service logs)" >&2
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
