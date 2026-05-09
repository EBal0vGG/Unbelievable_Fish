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
IDENTITY_URL="${IDENTITY_URL:-http://localhost:8084}"
BILLING_URL="${BILLING_URL:-http://localhost:8085/billing}"

PGUSER="${PGUSER:-fish}"
PGDATABASE="${PGDATABASE:-fish}"
PGPASSWORD="${PGPASSWORD:-fish}"
PGPORT="${PGPORT:-5433}"
PGSSLMODE="${PGSSLMODE:-disable}"

json_get() {
  python3 -c 'import json,sys; print(json.loads(sys.argv[1])[sys.argv[2]])' "$1" "$2"
}

json_get_optional() {
  python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get(sys.argv[2], ""))' "$1" "$2"
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

# Bidders need available balance to reserve the per-auction deposit (see trading reserve deposit).
billing_test_topup() {
  local token="$1"
  local amount="$2"
  curl -sS -o /dev/null -w "%{http_code}" -X POST "$BILLING_URL/accounts/me/top-up/test" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{\"amount\":$amount}"
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
if [[ -z "$(docker compose ps -q integration)" ]]; then
  echo "Integration service is not running. Start it: docker compose up -d integration" >&2
  exit 1
fi
if [[ -z "$(docker compose ps -q billing 2>/dev/null || true)" ]]; then
  echo "Billing service is not running. Start it: docker compose up -d billing" >&2
  exit 1
fi

fish_resp="$(curl -s -X POST "$CATALOG_URL/fish" -H "Content-Type: application/json" -d '{"name":"Cod","description":"desc"}')"
fish_id="$(json_get "$fish_resp" "fish_id")"

product_resp="$(curl -s -X POST "$CATALOG_URL/products" -H "Content-Type: application/json" -d "{\"fish_id\":\"$fish_id\",\"weight\":10,\"unit\":\"kg\",\"size\":\"M\",\"processing_type\":\"frozen\"}")"
product_id="$(json_get "$product_resp" "product_id")"

curl -s -X POST "$CATALOG_URL/products/$product_id/publish" >/dev/null

suffix="$(date +%s)"
seller_creds="$(valid_requisites "seller-$suffix")"
buyer1_creds="$(valid_requisites "buyer1-$suffix")"
buyer2_creds="$(valid_requisites "buyer2-$suffix")"

seller_inn="$(echo "$seller_creds" | awk '{print $1}')"
seller_ogrn="$(echo "$seller_creds" | awk '{print $2}')"
buyer1_inn="$(echo "$buyer1_creds" | awk '{print $1}')"
buyer1_ogrn="$(echo "$buyer1_creds" | awk '{print $2}')"
buyer2_inn="$(echo "$buyer2_creds" | awk '{print $1}')"
buyer2_ogrn="$(echo "$buyer2_creds" | awk '{print $2}')"

seller_company_id="$(register_company "Demo Seller $suffix" "$seller_inn" "$seller_ogrn")"
buyer1_company_id="$(register_company "Demo Buyer One $suffix" "$buyer1_inn" "$buyer1_ogrn")"
buyer2_company_id="$(register_company "Demo Buyer Two $suffix" "$buyer2_inn" "$buyer2_ogrn")"

seller_login="seller.$suffix@example.com"
buyer1_login="buyer1.$suffix@example.com"
buyer2_login="buyer2.$suffix@example.com"
password="secret123"

register_user "$seller_company_id" "Demo Seller" "seller" "$seller_login" "$password"
register_user "$buyer1_company_id" "Demo Buyer One" "buyer" "$buyer1_login" "$password"
register_user "$buyer2_company_id" "Demo Buyer Two" "buyer" "$buyer2_login" "$password"

seller_token="$(login_token "$seller_login" "$password")"
buyer1_token="$(login_token "$buyer1_login" "$password")"
buyer2_token="$(login_token "$buyer2_login" "$password")"

for _tok in "$buyer1_token" "$buyer2_token"; do
  code="$(billing_test_topup "$_tok" 500000)"
  if [[ "$code" != "204" ]]; then
    echo "billing test top-up failed HTTP $code (need billing up and BILLING_ENABLE_FAKE_PROVIDER=true for /accounts/me/top-up/test)" >&2
    exit 1
  fi
done

if command -v gdate >/dev/null 2>&1; then
  starts_at="$(gdate -u -d "-1 min" +%Y-%m-%dT%H:%M:%SZ)"
elif date -u -d "-1 min" +%Y-%m-%dT%H:%M:%SZ >/dev/null 2>&1; then
  starts_at="$(date -u -d "-1 min" +%Y-%m-%dT%H:%M:%SZ)"
else
  starts_at="$(date -u -v-1M +%Y-%m-%dT%H:%M:%SZ)"
fi
lot_resp="$(curl -fsS -X POST "$CATALOG_URL/lots" -H "Content-Type: application/json" -H "Authorization: Bearer $seller_token" -d "{\"product_id\":\"$product_id\",\"photo\":\"photo\",\"quantity\":10,\"start_price\":100,\"auction_starts_at\":\"$starts_at\",\"auction_duration_minutes\":2}")"
lot_id="$(json_get "$lot_resp" "lot_id")"

curl -fsS -X POST "$CATALOG_URL/lots/$lot_id/publish" -H "Authorization: Bearer $seller_token" >/dev/null

auction_id=""
for _ in $(seq 1 30); do
  code="$(curl -s -o /tmp/demo_happy_bylot.json -w "%{http_code}" "$TRADING_URL/auctions/by-lot/$lot_id" -H "Authorization: Bearer $seller_token")"
  if [[ "$code" == "200" ]]; then
    auction_id="$(json_get_optional "$(cat /tmp/demo_happy_bylot.json)" "auction_id")"
    if [[ -n "$auction_id" ]]; then
      break
    fi
  fi
  sleep 1
done
if [[ -z "$auction_id" ]]; then
  echo "Auction not found for lot: $lot_id (check integration service logs)" >&2
  cat /tmp/demo_happy_bylot.json >&2 || true
  exit 1
fi

curl -s -o /dev/null -w "%{http_code}\n" -X POST "$TRADING_URL/auctions/$auction_id/bids" -H "Content-Type: application/json" -H "Authorization: Bearer $buyer1_token" -d '{"amount":120}' | grep -q "202"
curl -s -o /dev/null -w "%{http_code}\n" -X POST "$TRADING_URL/auctions/$auction_id/bids" -H "Content-Type: application/json" -H "Authorization: Bearer $buyer2_token" -d '{"amount":150}' | grep -q "202"

PGHOST="localhost" PGUSER="$PGUSER" PGPASSWORD="$PGPASSWORD" PGDATABASE="$PGDATABASE" PGPORT="$PGPORT" PGSSLMODE="$PGSSLMODE" AUCTION_ID="$auction_id" go run ./cmd/admin close-auction >/dev/null

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select auction_id, state from trading_auctions where auction_id = '$auction_id';"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select auction_id, status, current_index, deal_id from deal_winner_selections where auction_id = '$auction_id';"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select deal_id, customer_id, status from deals where auction_id = '$auction_id';"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select lot_id, status, auction_id, final_price from catalog_lots where lot_id = '$lot_id';"

echo "OK: fish_id=$fish_id product_id=$product_id lot_id=$lot_id auction_id=$auction_id"

if [[ "$STOP_COMPOSE" == "1" ]]; then
  docker compose down
fi
