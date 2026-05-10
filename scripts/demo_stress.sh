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
RESET_DB="${RESET_DB:-}"
CURL_TIMEOUT="${CURL_TIMEOUT:-30}"

# Concurrent bid waves: max parallel POSTs per batch (local Docker often can't handle LOT_COUNT at once).
CONCURRENT_LIMIT="${CONCURRENT_LIMIT:-4}"
# Pause between batches inside the same wave (milliseconds).
WAVE_SLEEP_MS="${WAVE_SLEEP_MS:-200}"

CATALOG_URL="${CATALOG_URL:-http://localhost:8081}"
TRADING_URL="${TRADING_URL:-http://localhost:8082}"
IDENTITY_URL="${IDENTITY_URL:-http://localhost:8084}"
BILLING_URL="${BILLING_URL:-http://localhost:8085/billing}"

PGUSER="${PGUSER:-fish}"
PGDATABASE="${PGDATABASE:-fish}"
PGPASSWORD="${PGPASSWORD:-fish}"
PGPORT="${PGPORT:-5433}"
PGSSLMODE="${PGSSLMODE:-disable}"

LOT_COUNT="${LOT_COUNT:-10}"
BIDS_PER_AUCTION="${BIDS_PER_AUCTION:-20}"
INVALID_BIDS="${INVALID_BIDS:-5}"
POST_CLOSE_BIDS="${POST_CLOSE_BIDS:-5}"
CONCURRENT_BIDS="${CONCURRENT_BIDS:-1}"

# Registered buyer JWTs (bids require Authorization like production). Default pool size covers sequential + concurrent modulo.
_pool_default="$BIDS_PER_AUCTION"
if (( LOT_COUNT > _pool_default )); then _pool_default=$LOT_COUNT; fi
if (( 32 > _pool_default )); then _pool_default=32; fi
NUM_STRESS_BIDDERS="${NUM_STRESS_BIDDERS:-$_pool_default}"

# Stress scenario should not race with scheduler. Use a comfortably long duration.
AUCTION_DURATION_MINUTES="${AUCTION_DURATION_MINUTES:-10}"
# Start now (or slightly in the future if overridden), not in the past.
AUCTION_STARTS_AT="${AUCTION_STARTS_AT:-}"
# Poll until every WON auction from *this run* has a deal row (integration relay drains outbox). 0 = skip wait.
DEAL_RELAY_MAX_WAIT_SEC="${DEAL_RELAY_MAX_WAIT_SEC:-90}"
DEAL_RELAY_POLL_INTERVAL="${DEAL_RELAY_POLL_INTERVAL:-1}"
# If 1, exit non-zero when WON-without-deal remains after max wait (default: warn only).
DEAL_RELAY_STRICT="${DEAL_RELAY_STRICT:-0}"

json_get() {
  python3 -c '
import json, sys
body, key = sys.argv[1], sys.argv[2]
try:
    print(json.loads(body)[key])
except Exception as e:
    sys.stderr.write("json_get failed: %s\n" % e)
    sys.stderr.write("body (first 800 chars): %r\n" % (body[:800],))
    sys.exit(1)
' "$1" "$2"
}

curl_cmd() {
  curl --connect-timeout "$CURL_TIMEOUT" --max-time "$CURL_TIMEOUT" "$@"
}

sleep_ms() {
  local ms="${1:-0}"
  [[ "$ms" =~ ^[0-9]+$ ]] || ms=0
  python3 -c "import time; time.sleep(max(0, int('$ms')) / 1000.0)"
}

curl_post_json() {
  local url="$1"
  local data="$2"
  shift 2
  local tmp ec=0 code
  tmp="$(mktemp)"
  code="$(curl_cmd -sS -o "$tmp" -w "%{http_code}" -X POST "$url" \
    -H "Content-Type: application/json" "$@" -d "$data")" || ec=$?
  local body
  body="$(cat "$tmp" 2>/dev/null || true)"
  rm -f "$tmp"
  if [[ $ec -ne 0 ]]; then
    echo "curl failed (exit $ec) for POST $url" >&2
    [[ -n "$body" ]] && echo "$body" >&2
    exit 1
  fi
  if [[ "$code" -lt 200 || "$code" -ge 300 ]]; then
    echo "HTTP $code from POST $url" >&2
    [[ -n "$body" ]] && echo "$body" >&2
    exit 1
  fi
  printf '%s' "$body"
}

# Prints HTTP status; uses 000 when curl got no response (timeout, reset, etc.). Always exits 0 from the function body.
http_code() {
  local url="$1"
  local data="$2"
  shift 2
  local code
  code="$(curl_cmd -sS -o /dev/null -w "%{http_code}" -X POST "$url" \
    -H "Content-Type: application/json" "$@" -d "$data")" || true
  [[ -z "$code" ]] && code=000
  printf '%s' "$code"
}

post_expect_202() {
  local url="$1"
  local data="$2"
  shift 2
  local code
  code="$(http_code "$url" "$data" "$@")"
  if [[ "$code" != "202" ]]; then
    echo "expected HTTP 202, got $code for POST $url" >&2
    return 1
  fi
}

post_expect_not_202() {
  local url="$1"
  local data="$2"
  shift 2
  local code
  code="$(http_code "$url" "$data" "$@")"
  if [[ "$code" == "202" ]]; then
    echo "expected non-202, got 202 for POST $url" >&2
    return 1
  fi
}

stress_valid_requisites() {
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

stress_register_company() {
  local name="$1"
  local inn="$2"
  local ogrn="$3"
  local resp
  resp="$(curl_cmd -fsS -X POST "$IDENTITY_URL/companies" -H "Content-Type: application/json" -d "{\"name\":\"$name\",\"inn\":\"$inn\",\"ogrn\":\"$ogrn\"}")"
  json_get "$resp" "id"
}

stress_register_user() {
  local company_id="$1"
  local name="$2"
  local role="$3"
  local login="$4"
  local password="$5"
  curl_cmd -fsS -X POST "$IDENTITY_URL/users" -H "Content-Type: application/json" \
    -d "{\"company_id\":\"$company_id\",\"name\":\"$name\",\"role\":\"$role\",\"login\":\"$login\",\"password\":\"$password\",\"accepted_terms\":true,\"terms_version\":\"2026-04-24\"}" >/dev/null
}

stress_login_token() {
  local login="$1"
  local password="$2"
  local resp
  resp="$(curl_cmd -fsS -X POST "$IDENTITY_URL/auth/login" -H "Content-Type: application/json" -d "{\"login\":\"$login\",\"password\":\"$password\"}")"
  json_get "$resp" "token"
}

billing_test_topup() {
  local token="$1"
  local amount="$2"
  curl_cmd -sS -o /dev/null -w "%{http_code}" -X POST "$BILLING_URL/accounts/me/top-up/test" \
    -H "Authorization: Bearer $token" \
    -H "Content-Type: application/json" \
    -d "{\"amount\":$amount}"
}

require_catalog_up() {
  local code ec=0
  code="$(curl_cmd -sS -o /dev/null -w "%{http_code}" "$CATALOG_URL/health")" || ec=$?
  if [[ $ec -ne 0 || "$code" != "200" ]]; then
    echo "Catalog is not reachable at $CATALOG_URL (GET /health -> HTTP ${code:-?}, curl exit $ec)." >&2
    echo "Start stack: START_COMPOSE=1 ./scripts/demo_stress.sh   or   docker compose up -d" >&2
    exit 1
  fi
}

require_billing_up() {
  if [[ -z "$(docker compose ps -q billing 2>/dev/null || true)" ]]; then
    echo "Billing service is not running. Start stack: docker compose up -d billing" >&2
    exit 1
  fi
}

query_scalar() {
  local sql="$1"
  docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -t -A -c "$sql" | tr -d '[:space:]'
}

_sql_auction_id_list() {
  local first=1
  for id in "${AUCTION_IDS[@]}"; do
    if [[ -n "$first" ]]; then
      first=
    else
      echo -n ","
    fi
    printf "'%s'" "$id"
  done
}

# WON auctions created in this script that still lack a deals row (relay not finished or failed outbox).
won_without_deal_count_this_run() {
  if ((${#AUCTION_IDS[@]} == 0)); then
    echo "0"
    return
  fi
  local ids sql
  ids="$(_sql_auction_id_list)"
  sql="select count(*)::text from trading_auctions a where a.state = 'WON' and a.auction_id in ($ids) and not exists (select 1 from deals d where d.auction_id = a.auction_id)"
  query_scalar "$sql"
}

print_auction_debug() {
  local auction_id="$1"
  echo "-- debug for auction $auction_id --" >&2
  docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c \
    "select auction_id, lot_id, state, starts_at, ends_at, current_price from trading_auctions where auction_id = '$auction_id';" >&2 || true
}

banner "Compose up/reset (optional)"
if [[ "$RESET_DB" == "1" ]]; then
  docker compose down -v
fi
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

banner "Create fish and product"
require_catalog_up
require_billing_up
fish_id="$(catalog_demo_fish_id "$CATALOG_URL")"

stress_suffix="$(date +%s)"
read -r _stress_inn _stress_ogrn <<< "$(stress_valid_requisites "stress-seller-$stress_suffix")"
stress_seller_company_id="$(stress_register_company "Stress Seller $stress_suffix" "$_stress_inn" "$_stress_ogrn")"
stress_seller_login="stress.seller.$stress_suffix@example.com"
stress_password="secret123"
stress_register_user "$stress_seller_company_id" "Stress Seller" "seller" "$stress_seller_login" "$stress_password"
stress_seller_token="$(stress_login_token "$stress_seller_login" "$stress_password")"

declare -a STRESS_BIDDER_TOKENS
banner "Register stress bidders (JWT + billing top-up for deposit reserve)"
for n in $(seq 1 "$NUM_STRESS_BIDDERS"); do
  read -r _bin _bog <<< "$(stress_valid_requisites "stress-bidder-$stress_suffix-$n")"
  _bcid="$(stress_register_company "Stress Bidder $stress_suffix $n" "$_bin" "$_bog")"
  _login="stress.bidder.$stress_suffix.$n@example.com"
  stress_register_user "$_bcid" "Stress Bidder $n" "buyer" "$_login" "$stress_password"
  _tok="$(stress_login_token "$_login" "$stress_password")"
  _code="$(billing_test_topup "$_tok" 500000)"
  if [[ "$_code" != "204" ]]; then
    echo "billing test top-up failed HTTP $_code for bidder $n (need BILLING_ENABLE_FAKE_PROVIDER=true)" >&2
    exit 1
  fi
  STRESS_BIDDER_TOKENS+=("$_tok")
done

product_resp="$(curl_post_json "$CATALOG_URL/products" "{\"fish_id\":\"$fish_id\",\"weight\":5,\"unit\":\"kg\",\"size\":\"M\",\"processing_type\":\"frozen\"}" \
  -H "Authorization: Bearer $stress_seller_token")"
product_id="$(json_get "$product_resp" "product_id")"

curl_post_json "$CATALOG_URL/products/$product_id/publish" "{}" -H "Authorization: Bearer $stress_seller_token"

declare -a LOT_IDS
declare -a AUCTION_IDS

banner "Create and publish lots (repeat publish should fail)"
for i in $(seq 1 "$LOT_COUNT"); do
  if [[ -n "$AUCTION_STARTS_AT" ]]; then
    starts_at="$AUCTION_STARTS_AT"
  else
    starts_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  fi

  lot_resp="$(curl_post_json "$CATALOG_URL/lots" "{\"product_id\":\"$product_id\",\"photo\":\"photo\",\"quantity\":10,\"start_price\":100,\"auction_starts_at\":\"$starts_at\",\"auction_duration_minutes\":$AUCTION_DURATION_MINUTES}" \
    -H "Authorization: Bearer $stress_seller_token")"
  lot_id="$(json_get "$lot_resp" "lot_id")"
  LOT_IDS+=("$lot_id")

  post_expect_202 "$CATALOG_URL/lots/$lot_id/publish" "{}" -H "Authorization: Bearer $stress_seller_token"
  post_expect_not_202 "$CATALOG_URL/lots/$lot_id/publish" "{}" -H "Authorization: Bearer $stress_seller_token"
done

banner "Resolve auctions"
sleep 2

for lot_id in "${LOT_IDS[@]}"; do
  auction_id="$(query_scalar "select auction_id from catalog_lots where lot_id = '$lot_id' limit 1;")"
  if [[ -z "$auction_id" ]]; then
    echo "failed to resolve auction_id for lot $lot_id" >&2
    exit 1
  fi
  AUCTION_IDS+=("$auction_id")
done

banner "Place valid bids"
_n_pool=${#STRESS_BIDDER_TOKENS[@]}
if ((_n_pool < 1)); then
  echo "internal error: no bidder tokens" >&2
  exit 1
fi
for auction_id in "${AUCTION_IDS[@]}"; do
  for b in $(seq 1 "$BIDS_PER_AUCTION"); do
    amount=$((100 + b * 10))
    _idx=$(( (b - 1) % _n_pool ))
    _tok="${STRESS_BIDDER_TOKENS[$_idx]}"
    if ! post_expect_202 "$TRADING_URL/auctions/$auction_id/bids" "{\"amount\":$amount}" \
      -H "Authorization: Bearer $_tok"; then
      print_auction_debug "$auction_id"
      exit 1
    fi
  done
done

SEQUENTIAL_TOP=$((100 + BIDS_PER_AUCTION * 10))

# Parallelize across auctions, not within the same auction. Each wave submits one bid per auction.
# Batches of CONCURRENT_LIMIT avoid overloading local trading; wave summary shows 202 vs 000 vs errors.
if [[ "$CONCURRENT_BIDS" == "1" ]]; then
  banner "Place concurrent bids (limit=$CONCURRENT_LIMIT timeout=${CURL_TIMEOUT}s sleep_ms=$WAVE_SLEEP_MS)"
  n_auctions=${#AUCTION_IDS[@]}
  for b in $(seq 1 "$BIDS_PER_AUCTION"); do
    declare -i c202=0 c000=0 c4xx=0 c5xx=0 cother=0
    wave_dir="$(mktemp -d)"
    for ((offset = 0; offset < n_auctions; offset += CONCURRENT_LIMIT)); do
      declare -a _batch_pids=()
      for ((k = 0; k < CONCURRENT_LIMIT && offset + k < n_auctions; k++)); do
        i=$((offset + k))
        auction_id="${AUCTION_IDS[$i]}"
        amount=$((SEQUENTIAL_TOP + 1 + b * 5))
        out="$wave_dir/${offset}_$k.code"
        (
          _idx=$(( (i + b) % _n_pool ))
          _tok="${STRESS_BIDDER_TOKENS[$_idx]}"
          code="$(http_code "$TRADING_URL/auctions/$auction_id/bids" "{\"amount\":$amount}" \
            -H "Authorization: Bearer $_tok")"
          printf '%s %s %s\n' "$code" "$auction_id" "$amount" >"$out"
        ) &
        _batch_pids+=("$!")
      done
      for _pid in "${_batch_pids[@]}"; do
        wait "$_pid" || true
      done
      while IFS= read -r -d '' f; do
        read -r code _aid _amt <"$f" || true
        [[ -z "$code" ]] && code=000
        if [[ "$code" == "202" ]]; then
          c202+=1
        elif [[ "$code" == "000" ]]; then
          c000+=1
        elif [[ "$code" =~ ^4[0-9][0-9]$ ]]; then
          c4xx+=1
        elif [[ "$code" =~ ^5[0-9][0-9]$ ]]; then
          c5xx+=1
        else
          cother+=1
        fi
        rm -f "$f"
      done < <(find "$wave_dir" -maxdepth 1 -name '*.code' -type f -print0 2>/dev/null)
      if ((WAVE_SLEEP_MS > 0 && offset + CONCURRENT_LIMIT < n_auctions)); then
        sleep_ms "$WAVE_SLEEP_MS"
      fi
    done
    rmdir "$wave_dir" 2>/dev/null || rm -rf "$wave_dir"
    echo "concurrent wave $b/$BIDS_PER_AUCTION: 202=$c202 000=$c000 4xx=$c4xx 5xx=$c5xx other=$cother (expected 202 count=$n_auctions)" >&2
    if ((WAVE_SLEEP_MS > 0 && b < BIDS_PER_AUCTION)); then
      sleep_ms "$WAVE_SLEEP_MS"
    fi
    if ((c202 != n_auctions)); then
      echo "concurrent wave $b failed: got $c202 successes, need $n_auctions (check 000=timeout/no-response, 4xx=client error)" >&2
      if ((n_auctions > 0)); then
        print_auction_debug "${AUCTION_IDS[0]}"
      fi
      exit 1
    fi
  done
fi

banner "Invalid bids (must be rejected)"
for auction_id in "${AUCTION_IDS[@]}"; do
  for _ in $(seq 1 "$INVALID_BIDS"); do
    post_expect_not_202 "$TRADING_URL/auctions/$auction_id/bids" '{"amount":1}' \
      -H "Authorization: Bearer ${STRESS_BIDDER_TOKENS[0]}"
  done
done

banner "Close auctions"
for auction_id in "${AUCTION_IDS[@]}"; do
  PGHOST="localhost" PGUSER="$PGUSER" PGPASSWORD="$PGPASSWORD" PGDATABASE="$PGDATABASE" PGPORT="$PGPORT" PGSSLMODE="$PGSSLMODE" AUCTION_ID="$auction_id" go run ./cmd/admin close-auction >/dev/null
done

banner "Post-close bids (must be rejected)"
for auction_id in "${AUCTION_IDS[@]}"; do
  for _ in $(seq 1 "$POST_CLOSE_BIDS"); do
    post_expect_not_202 "$TRADING_URL/auctions/$auction_id/bids" '{"amount":999}' \
      -H "Authorization: Bearer ${STRESS_BIDDER_TOKENS[0]}"
  done
done

banner "DB summary"
if (( DEAL_RELAY_MAX_WAIT_SEC > 0 && ${#AUCTION_IDS[@]} > 0 )); then
  echo "Polling for deals (this run: ${#AUCTION_IDS[@]} auctions, max ${DEAL_RELAY_MAX_WAIT_SEC}s, interval ${DEAL_RELAY_POLL_INTERVAL}s)..."
  _miss=999
  _elapsed=0
  while ((_elapsed <= DEAL_RELAY_MAX_WAIT_SEC)); do
    _miss="$(won_without_deal_count_this_run)"
    _miss="${_miss:-0}"
    if [[ "$_miss" == "0" ]]; then
      echo "All WON auctions from this run have matching deals (waited ${_elapsed}s)."
      break
    fi
    if ((_elapsed >= DEAL_RELAY_MAX_WAIT_SEC)); then
      echo "Still ${_miss} WON auction(s) from this run without deals after ${DEAL_RELAY_MAX_WAIT_SEC}s (see outbox stats below)." >&2
      break
    fi
    sleep "$DEAL_RELAY_POLL_INTERVAL"
    _elapsed=$((_elapsed + DEAL_RELAY_POLL_INTERVAL))
  done
fi

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "\
select event_type, \
  count(*) filter (where published_at is null and failed_at is null) as unpublished_ok, \
  count(*) filter (where failed_at is not null) as failed, \
  count(*) filter (where published_at is not null) as published \
from outbox_messages \
where event_type = 'trading.AuctionWon' \
group by event_type;"

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select count(*) as auctions_total from trading_auctions;"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select count(*) as deals_total from deals;"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select state, count(*) as n from trading_auctions group by state order by state;"
docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select count(*) as deal_projections_total from deal_projections;"

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "\
select deal_id, auction_id, customer_id, unit_price, status, created_at \
from deals order by created_at desc limit 15;"

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "\
select a.auction_id, a.state, a.current_price \
from trading_auctions a \
where a.state = 'WON' \
and not exists (select 1 from deals d where d.auction_id = a.auction_id) \
order by a.starts_at desc limit 20;"

docker exec -i "$PG_CONTAINER" psql -U "$PGUSER" -d "$PGDATABASE" -c "select auction_id, state, current_price from trading_auctions order by starts_at desc limit 10;"

echo ""
echo "Why auctions_total vs deals_total (whole DB) can differ:"
echo "  - Counts are over all rows in Postgres, not only this script (use RESET_DB=1 for a clean volume)."
echo "  - A deal appears only after integration processes trading.AuctionWon from the shared outbox_messages table."
echo "  - Under load, many catalog/trading rows are ahead of AuctionWon in ORDER BY created_at; relay does ~100 msgs per ~100ms."
echo "  - Fixed sleep is unreliable; this script polls WON-without-deal for *this run* up to DEAL_RELAY_MAX_WAIT_SEC."
echo "  - If trading.AuctionWon rows show failed>0, check last_error in outbox_messages and integration logs."
echo ""

if [[ "${DEAL_RELAY_STRICT:-0}" == "1" ]]; then
  _final_miss="$(won_without_deal_count_this_run)"
  _final_miss="${_final_miss:-0}"
  if [[ "$_final_miss" != "0" ]]; then
    echo "DEAL_RELAY_STRICT=1: ${_final_miss} WON auction(s) from this run still without deals." >&2
    exit 1
  fi
fi

echo "OK: lots=${#LOT_IDS[@]} auctions=${#AUCTION_IDS[@]} bids_per_auction=$BIDS_PER_AUCTION duration_min=$AUCTION_DURATION_MINUTES"

banner "Compose down (optional)"
if [[ "$STOP_COMPOSE" == "1" ]]; then
  docker compose down
fi
