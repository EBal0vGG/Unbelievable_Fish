#!/usr/bin/env bash
set -euo pipefail

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; exit 1; }
}

require curl
require python3
require docker
require psql

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/catalog_demo_fish.sh"

START_COMPOSE="${START_COMPOSE:-}"
STOP_COMPOSE="${STOP_COMPOSE:-}"

IDENTITY_URL="${IDENTITY_URL:-http://localhost:8084}"
CATALOG_URL="${CATALOG_URL:-http://localhost:8081}"
TRADING_URL="${TRADING_URL:-http://localhost:8082}"
BILLING_URL="${BILLING_URL:-http://localhost:8085/billing}"

PGUSER="${PGUSER:-fish}"
PGDATABASE="${PGDATABASE:-fish}"
PGPASSWORD="${PGPASSWORD:-fish}"
PGPORT="${PGPORT:-5433}"
PGSSLMODE="${PGSSLMODE:-disable}"

PASS=0
FAIL=0

pass() {
  echo "PASS: $1"
  PASS=$((PASS + 1))
}

fail() {
  echo "FAIL: $1" >&2
  FAIL=$((FAIL + 1))
}

assert_eq() {
  local name="$1"
  local got="$2"
  local want="$3"
  if [[ "$got" == "$want" ]]; then
    pass "$name (got=$got)"
  else
    fail "$name (got=$got want=$want)"
  fi
}

json_get() {
  python3 -c 'import json,sys; obj=json.loads(sys.argv[1]); print(obj[sys.argv[2]])' "$1" "$2"
}

gen_requisites() {
  python3 - <<'PY'
import random
import time

random.seed(time.time_ns())
w = [2, 4, 10, 3, 5, 9, 4, 6, 8]
base = [random.randint(0, 9) for _ in range(9)]
inn = "".join(map(str, base + [sum(a*b for a, b in zip(base, w)) % 11 % 10]))
ogrn_base = "1" + "".join(str(random.randint(0, 9)) for _ in range(11))
ogrn = ogrn_base + str(int(ogrn_base) % 11 % 10)
print(inn, ogrn)
PY
}

mk_company() {
  local name="$1"
  local inn="$2"
  local ogrn="$3"
  curl -s -X POST "$IDENTITY_URL/companies" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"inn\":\"$inn\",\"ogrn\":\"$ogrn\"}"
}

mk_user() {
  local company_id="$1"
  local name="$2"
  local role="$3"
  local login="$4"
  curl -s -X POST "$IDENTITY_URL/users" \
    -H "Content-Type: application/json" \
    -d "{\"company_id\":\"$company_id\",\"name\":\"$name\",\"role\":\"$role\",\"login\":\"$login\",\"password\":\"pass12345\",\"accepted_terms\":true,\"terms_version\":\"v1\"}"
}

login() {
  local login_value="$1"
  curl -s -X POST "$IDENTITY_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"login\":\"$login_value\",\"password\":\"pass12345\"}"
}

billing_test_topup() {
  local token="$1"
  local amount="$2"
  curl -sS -o /dev/null -w "%{http_code}" -X POST "$BILLING_URL/accounts/me/top-up/test" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{\"amount\":$amount}"
}

db_scalar() {
  local sql="$1"
  PGPASSWORD="$PGPASSWORD" psql \
    -h localhost \
    -p "$PGPORT" \
    -U "$PGUSER" \
    -d "$PGDATABASE" \
    -t -A -c "$sql" | tr -d '[:space:]'
}

if [[ "$START_COMPOSE" == "1" ]]; then
  docker compose up -d --build
fi

TS="$(date +%s)"
read -r SELLER_INN SELLER_OGRN <<< "$(gen_requisites)"
read -r BUYER1_INN BUYER1_OGRN <<< "$(gen_requisites)"
read -r BUYER2_INN BUYER2_OGRN <<< "$(gen_requisites)"

seller_company_resp="$(mk_company "seller-$TS" "$SELLER_INN" "$SELLER_OGRN")"
seller_company_id="$(json_get "$seller_company_resp" "id")"
mk_user "$seller_company_id" "Seller User" "seller" "seller_${TS}@example.com" >/dev/null
seller_login_resp="$(login "seller_${TS}@example.com")"
seller_token="$(json_get "$seller_login_resp" "token")"

buyer1_company_resp="$(mk_company "buyer1-$TS" "$BUYER1_INN" "$BUYER1_OGRN")"
buyer1_company_id="$(json_get "$buyer1_company_resp" "id")"
mk_user "$buyer1_company_id" "Buyer One" "buyer" "buyer1_${TS}@example.com" >/dev/null
buyer1_login_resp="$(login "buyer1_${TS}@example.com")"
buyer1_token="$(json_get "$buyer1_login_resp" "token")"

buyer2_company_resp="$(mk_company "buyer2-$TS" "$BUYER2_INN" "$BUYER2_OGRN")"
buyer2_company_id="$(json_get "$buyer2_company_resp" "id")"
mk_user "$buyer2_company_id" "Buyer Two" "buyer" "buyer2_${TS}@example.com" >/dev/null
buyer2_login_resp="$(login "buyer2_${TS}@example.com")"
buyer2_token="$(json_get "$buyer2_login_resp" "token")"

for _tok in "$buyer1_token" "$buyer2_token"; do
  code="$(billing_test_topup "$_tok" 500000)"
  if [[ "$code" != "204" ]]; then
    echo "billing test top-up failed HTTP $code (start billing; BILLING_ENABLE_FAKE_PROVIDER=true for /accounts/me/top-up/test)" >&2
    exit 1
  fi
done

fish_id="$(catalog_demo_fish_id "$CATALOG_URL")"
product_resp="$(curl -fsS -X POST "$CATALOG_URL/products" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $seller_token" \
  -d "{\"fish_id\":\"$fish_id\",\"weight\":10,\"unit\":\"kg\",\"size\":\"M\",\"processing_type\":\"frozen\"}")"
product_id="$(json_get "$product_resp" "product_id")"
curl -fsS -X POST "$CATALOG_URL/products/$product_id/publish" -H "Authorization: Bearer $seller_token" >/dev/null

starts_at="$(date -u -d "-1 min" +%Y-%m-%dT%H:%M:%SZ)"
lot_resp="$(curl -s -X POST "$CATALOG_URL/lots" \
  -H "Authorization: Bearer $seller_token" \
  -H "Content-Type: application/json" \
  -d "{\"product_id\":\"$product_id\",\"photo\":\"photo\",\"quantity\":10,\"start_price\":100,\"min_bid_step\":10,\"auction_starts_at\":\"$starts_at\",\"auction_duration_minutes\":2}")"
lot_id="$(json_get "$lot_resp" "lot_id")"
curl -s -X POST "$CATALOG_URL/lots/$lot_id/publish" -H "Authorization: Bearer $seller_token" >/dev/null

auction_id=""
for _ in $(seq 1 20); do
  code="$(curl -s -o /tmp/e2e_bylot.json -w "%{http_code}" "$TRADING_URL/auctions/by-lot/$lot_id" -H "Authorization: Bearer $seller_token")"
  if [[ "$code" == "200" ]]; then
    auction_id="$(python3 -c 'import json; print(json.load(open("/tmp/e2e_bylot.json"))["auction_id"])')"
    break
  fi
  sleep 0.5
done
if [[ -z "$auction_id" ]]; then
  echo "FAILED: auction_id not created for lot=$lot_id" >&2
  exit 1
fi

end_before="$(db_scalar "select ends_at from trading_auctions where auction_id = '$auction_id';")"
bid_at_1="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
(
  curl -s -o /tmp/e2e_bid1.out -w "%{http_code}" -X POST "$TRADING_URL/auctions/$auction_id/bids" \
    -H "Authorization: Bearer $buyer1_token" \
    -H "Content-Type: application/json" \
    -d "{\"amount\":110,\"placed_at\":\"$bid_at_1\"}" > /tmp/e2e_bid1.code
) &
(
  sleep 0.1
  curl -s -o /tmp/e2e_bid2.out -w "%{http_code}" -X POST "$TRADING_URL/auctions/$auction_id/bids" \
    -H "Authorization: Bearer $buyer2_token" \
    -H "Content-Type: application/json" \
    -d "{\"amount\":120,\"placed_at\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" > /tmp/e2e_bid2.code
) &
wait

code_bid1="$(cat /tmp/e2e_bid1.code)"
code_bid2="$(cat /tmp/e2e_bid2.code)"
current_price="$(db_scalar "select current_price from trading_auctions where auction_id = '$auction_id';")"
leader_company_id="$(db_scalar "select leader_company_id from trading_auctions where auction_id = '$auction_id';")"
end_after="$(db_scalar "select ends_at from trading_auctions where auction_id = '$auction_id';")"

low_code="$(curl -s -o /tmp/e2e_low.out -w "%{http_code}" -X POST "$TRADING_URL/auctions/$auction_id/bids" \
  -H "Authorization: Bearer $buyer1_token" \
  -H "Content-Type: application/json" \
  -d "{\"amount\":121,\"placed_at\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}")"

late_code="$(curl -s -o /tmp/e2e_late.out -w "%{http_code}" -X POST "$TRADING_URL/auctions/$auction_id/bids" \
  -H "Authorization: Bearer $buyer1_token" \
  -H "Content-Type: application/json" \
  -d "{\"amount\":130,\"placed_at\":\"2099-01-01T00:00:00Z\"}")"

starts_at_2="$(date -u -d "-1 min" +%Y-%m-%dT%H:%M:%SZ)"
lot2_resp="$(curl -s -X POST "$CATALOG_URL/lots" \
  -H "Authorization: Bearer $seller_token" \
  -H "Content-Type: application/json" \
  -d "{\"product_id\":\"$product_id\",\"photo\":\"photo\",\"quantity\":10,\"start_price\":100,\"min_bid_step\":10,\"auction_starts_at\":\"$starts_at_2\",\"auction_duration_minutes\":30}")"
lot2_id="$(json_get "$lot2_resp" "lot_id")"
curl -s -X POST "$CATALOG_URL/lots/$lot2_id/publish" -H "Authorization: Bearer $seller_token" >/dev/null

auction2_id=""
for _ in $(seq 1 20); do
  code="$(curl -s -o /tmp/e2e_bylot2.json -w "%{http_code}" "$TRADING_URL/auctions/by-lot/$lot2_id" -H "Authorization: Bearer $seller_token")"
  if [[ "$code" == "200" ]]; then
    auction2_id="$(python3 -c 'import json; print(json.load(open("/tmp/e2e_bylot2.json"))["auction_id"])')"
    break
  fi
  sleep 0.5
done
if [[ -z "$auction2_id" ]]; then
  echo "FAILED: auction2_id not created for lot2=$lot2_id" >&2
  exit 1
fi

end2_before="$(db_scalar "select ends_at from trading_auctions where auction_id = '$auction2_id';")"
mid_code="$(curl -s -o /tmp/e2e_mid.out -w "%{http_code}" -X POST "$TRADING_URL/auctions/$auction2_id/bids" \
  -H "Authorization: Bearer $buyer1_token" \
  -H "Content-Type: application/json" \
  -d "{\"amount\":110,\"placed_at\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}")"
end2_after="$(db_scalar "select ends_at from trading_auctions where auction_id = '$auction2_id';")"

assert_eq "bid1 accepted" "$code_bid1" "202"
assert_eq "bid2 accepted" "$code_bid2" "202"
assert_eq "race final currentPrice" "$current_price" "120"
assert_eq "race final leader" "$leader_company_id" "$buyer2_company_id"
if [[ "$end_after" != "$end_before" ]]; then
  pass "endsAt extended inside extension window"
else
  fail "endsAt should extend inside extension window"
fi
assert_eq "reject bid below min step" "$low_code" "400"
assert_eq "reject bid after endsAt" "$late_code" "409"
assert_eq "accept bid outside extension window" "$mid_code" "202"
assert_eq "endsAt unchanged outside extension window" "$end2_after" "$end2_before"

echo "----"
echo "SUMMARY: PASS=$PASS FAIL=$FAIL"
echo "ids: lot=$lot_id auction=$auction_id lot2=$lot2_id auction2=$auction2_id"

if [[ "$STOP_COMPOSE" == "1" ]]; then
  docker compose down
fi

if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
