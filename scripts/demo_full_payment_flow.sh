#!/usr/bin/env bash
# End-to-end: catalog → auction → deal lifecycle → billing invoice (fake pay) →
# admin payout READY/PAID → seller available + single SELLER_PAYOUT_CREDITED ledger row.
#
# Requires: docker compose with postgres, identity, catalog, trading, deals, billing, integration.
# Billing must expose fake + admin routes (see docker-compose env).
# Admin JWT: this script promotes a dedicated ops user to role admin via SQL (registration as admin is forbidden).
set -euo pipefail

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
DEALS_URL="${DEALS_URL:-http://localhost:8083}"
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

billing_test_topup() {
  local token="$1"
  local amount="$2"
  curl -sS -o /dev/null -w "%{http_code}" -X POST "$BILLING_URL/accounts/me/top-up/test" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{\"amount\":$amount}"
}

deal_confirmation_create() {
  local deal_id="$1"
  local token="$2"
  local stage="$3"
  curl -fsS -X POST "$DEALS_URL/deals/$deal_id/confirmations" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{\"stage\":\"$stage\",\"verification_method\":\"manual\",\"comment\":\"demo-full-pay\"}"
}

deal_confirmation_approve() {
  local deal_id="$1"
  local conf_id="$2"
  local token="$3"
  curl -fsS -o /dev/null -w "%{http_code}" -X POST "$DEALS_URL/deals/$deal_id/confirmations/$conf_id/approve" \
    -H "Authorization: Bearer $token"
}

if [[ "$START_COMPOSE" == "1" ]]; then
  docker compose up -d --build
fi

PG_CONTAINER="${PG_CONTAINER:-}"
if [[ -z "$PG_CONTAINER" ]]; then
  PG_CONTAINER="$(docker compose ps -q postgres 2>/dev/null || true)"
fi
if [[ -z "$PG_CONTAINER" ]]; then
  PG_CONTAINER="fish-postgres-1"
fi
if [[ -z "$(docker compose ps -q integration 2>/dev/null || true)" ]]; then
  echo "Integration service is not running. Start: docker compose up -d integration" >&2
  exit 1
fi
if [[ -z "$(docker compose ps -q billing 2>/dev/null || true)" ]]; then
  echo "Billing service is not running. Start: docker compose up -d billing" >&2
  exit 1
fi

fish_id="$(catalog_demo_fish_id "$CATALOG_URL")"

suffix="$(date +%s)"
seller_creds="$(valid_requisites "seller-$suffix")"
buyer1_creds="$(valid_requisites "buyer1-$suffix")"
buyer2_creds="$(valid_requisites "buyer2-$suffix")"
ops_creds="$(valid_requisites "ops-$suffix")"

seller_inn="$(echo "$seller_creds" | awk '{print $1}')"
seller_ogrn="$(echo "$seller_creds" | awk '{print $2}')"
buyer1_inn="$(echo "$buyer1_creds" | awk '{print $1}')"
buyer1_ogrn="$(echo "$buyer1_creds" | awk '{print $2}')"
buyer2_inn="$(echo "$buyer2_creds" | awk '{print $1}')"
buyer2_ogrn="$(echo "$buyer2_creds" | awk '{print $2}')"
ops_inn="$(echo "$ops_creds" | awk '{print $1}')"
ops_ogrn="$(echo "$ops_creds" | awk '{print $2}')"

seller_company_id="$(register_company "Demo Seller $suffix" "$seller_inn" "$seller_ogrn")"
buyer1_company_id="$(register_company "Demo Buyer One $suffix" "$buyer1_inn" "$buyer1_ogrn")"
buyer2_company_id="$(register_company "Demo Buyer Two $suffix" "$buyer2_inn" "$buyer2_ogrn")"
ops_company_id="$(register_company "Demo Ops $suffix" "$ops_inn" "$ops_ogrn")"

seller_login="seller.$suffix@example.com"
buyer1_login="buyer1.$suffix@example.com"
buyer2_login="buyer2.$suffix@example.com"
ops_login="ops.$suffix@example.com"
password="secret123"

register_user "$seller_company_id" "Demo Seller" "seller" "$seller_login" "$password"
register_user "$buyer1_company_id" "Demo Buyer One" "buyer" "$buyer1_login" "$password"
register_user "$buyer2_company_id" "Demo Buyer Two" "buyer" "$buyer2_login" "$password"
register_user "$ops_company_id" "Demo Ops" "buyer" "$ops_login" "$password"

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "UPDATE identity_users SET role = 'admin' WHERE login = '$ops_login';" >/dev/null

seller_token="$(login_token "$seller_login" "$password")"
buyer1_token="$(login_token "$buyer1_login" "$password")"
buyer2_token="$(login_token "$buyer2_login" "$password")"
ops_token="$(login_token "$ops_login" "$password")"

product_resp="$(curl -fsS -X POST "$CATALOG_URL/products" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $seller_token" \
  -d "{\"fish_id\":\"$fish_id\",\"weight\":10,\"unit\":\"kg\",\"size\":\"M\",\"processing_type\":\"frozen\"}")"
product_id="$(json_get "$product_resp" "product_id")"

curl -fsS -X POST "$CATALOG_URL/products/$product_id/publish" -H "Authorization: Bearer $seller_token" >/dev/null

code="$(billing_test_topup "$buyer1_token" 500000)"
[[ "$code" == "204" ]] || { echo "buyer1 test top-up expected 204, got $code" >&2; exit 1; }
code="$(billing_test_topup "$buyer2_token" 500000)"
[[ "$code" == "204" ]] || { echo "buyer2 test top-up expected 204, got $code" >&2; exit 1; }

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
  code="$(curl -s -o /tmp/demo_full_bylot.json -w "%{http_code}" "$TRADING_URL/auctions/by-lot/$lot_id" -H "Authorization: Bearer $seller_token")"
  if [[ "$code" == "200" ]]; then
    auction_id="$(json_get_optional "$(cat /tmp/demo_full_bylot.json)" "auction_id")"
    if [[ -n "$auction_id" ]]; then
      break
    fi
  fi
  sleep 1
done
if [[ -z "$auction_id" ]]; then
  echo "Auction not found for lot: $lot_id" >&2
  exit 1
fi

curl -fsS -o /dev/null -w "%{http_code}\n" -X POST "$TRADING_URL/auctions/$auction_id/bids" -H "Content-Type: application/json" -H "Authorization: Bearer $buyer1_token" -d '{"amount":120}' | grep -q "202"
curl -fsS -o /dev/null -w "%{http_code}\n" -X POST "$TRADING_URL/auctions/$auction_id/bids" -H "Content-Type: application/json" -H "Authorization: Bearer $buyer2_token" -d '{"amount":150}' | grep -q "202"

PGHOST="localhost" PGUSER="$PGUSER" PGPASSWORD="$PGPASSWORD" PGDATABASE="$PGDATABASE" PGPORT="$PGPORT" PGSSLMODE="$PGSSLMODE" AUCTION_ID="$auction_id" go run ./cmd/admin close-auction >/dev/null

sleep 2

deal_json="$(curl -fsS "$DEALS_URL/deals/by-auction/$auction_id" -H "Authorization: Bearer $buyer2_token")"
deal_id="$(json_get "$deal_json" "id")"

resp="$(deal_confirmation_create "$deal_id" "$seller_token" confirmed)"
c1="$(json_get "$resp" id)"
code="$(deal_confirmation_approve "$deal_id" "$c1" "$buyer2_token")"
[[ "$code" == "200" ]] || { echo "approve confirmed: $code" >&2; exit 1; }

curl -fsS -o /dev/null -w "%{http_code}\n" -X POST "$DEALS_URL/deals/$deal_id/contract/prepare" \
  -H "Authorization: Bearer $seller_token" \
  -H "Content-Type: application/json" \
  -d '{"contract_number":"CNT-DEMO","document_url":"https://contracts/demo.pdf"}' | grep -q "202"

curl -fsS -o /dev/null -w "%{http_code}\n" -X POST "$DEALS_URL/deals/$deal_id/contract/sign" \
  -H "Authorization: Bearer $buyer2_token" \
  -H "Content-Type: application/json" \
  -d '{"signature_ref":"SIG-DEMO"}' | grep -q "202"

due_iso="$(python3 - <<'PY'
from datetime import datetime, timedelta, timezone
print((datetime.now(timezone.utc) + timedelta(days=7)).strftime("%Y-%m-%dT%H:%M:%SZ"))
PY
)"

curl -fsS -o /dev/null -w "%{http_code}\n" -X POST "$DEALS_URL/deals/$deal_id/payment/request" \
  -H "Authorization: Bearer $seller_token" \
  -H "Content-Type: application/json" \
  -d "{\"invoice_number\":\"INV-DEMO-$suffix\",\"due_date\":\"$due_iso\"}" | grep -q "202"

sleep 3

inv_json="$(curl -fsS "$BILLING_URL/invoices/by-deal/$deal_id" -H "Authorization: Bearer $buyer2_token")"
invoice_id="$(json_get "$inv_json" id)"

http_code="$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BILLING_URL/invoices/$invoice_id/fake-confirm" -H "Authorization: Bearer $buyer2_token")"
if [[ "$http_code" != "204" ]]; then
  echo "fake-confirm invoice: expected 204, got $http_code (enable BILLING_ENABLE_FAKE_PROVIDER on billing)" >&2
  exit 1
fi

sleep 5

deal_after="$(curl -fsS "$DEALS_URL/deals/$deal_id" -H "Authorization: Bearer $buyer2_token")"
deal_status="$(json_get "$deal_after" status)"
if [[ "$deal_status" != "paid" ]]; then
  echo "expected deal status paid, got $deal_status" >&2
  exit 1
fi

payout_json="$(curl -fsS "$BILLING_URL/payouts/me" -H "Authorization: Bearer $seller_token")"
payout_id="$(printf '%s' "$payout_json" | python3 -c 'import json,sys; deal=sys.argv[1]; d=json.load(sys.stdin)
for p in d.get("payouts",[]):
  if p.get("deal_id")==deal:
    print(p["payout_id"])
    break
' "$deal_id")"
if [[ -z "$payout_id" ]]; then
  echo "seller payout for deal not found" >&2
  exit 1
fi

curl -fsS -X POST "$BILLING_URL/admin/payouts/$payout_id/ready" -H "Authorization: Bearer $ops_token" >/dev/null
curl -fsS -X POST "$BILLING_URL/admin/payouts/$payout_id/paid" -H "Authorization: Bearer $ops_token" >/dev/null
curl -fsS -X POST "$BILLING_URL/admin/payouts/$payout_id/paid" -H "Authorization: Bearer $ops_token" >/dev/null

goods_amount="$(json_get "$inv_json" goods_amount)"

bal_json="$(curl -fsS "$BILLING_URL/accounts/me" -H "Authorization: Bearer $seller_token")"
avail="$(json_get "$bal_json" available)"
if [[ "$avail" != "$goods_amount" ]]; then
  echo "seller available: want $goods_amount (goods_amount) got $avail" >&2
  exit 1
fi

ledger_count="$(docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -tAc \
  "SELECT count(*) FROM billing_ledger_entries WHERE company_id = '$seller_company_id' AND type = 'SELLER_PAYOUT_CREDITED' AND reference_type = 'seller_payout' AND reference_id = '$payout_id'")"
if [[ "$ledger_count" != "1" ]]; then
  echo "ledger SELLER_PAYOUT_CREDITED rows: want 1 got $ledger_count" >&2
  exit 1
fi

echo "OK demo_full_payment_flow: deal_id=$deal_id invoice_id=$invoice_id payout_id=$payout_id seller_avail=$avail"

if [[ "$STOP_COMPOSE" == "1" ]]; then
  docker compose down
fi
