#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
require python3

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <product_id> [seller_company_id] [start_price] [quantity] [duration_min]"
  exit 2
fi

PRODUCT_ID="$1"
SELLER_COMPANY_ID="${2:-seller-1}"
START_PRICE="${3:-100}"
QUANTITY="${4:-10}"
DURATION_MIN="${5:-5}"
STARTS_AT="${AUCTION_STARTS_AT:-$(date -u -d '-1 min' +%Y-%m-%dT%H:%M:%SZ)}"

resp="$(curl -sS -X POST "$CATALOG_URL/lots" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: $SELLER_COMPANY_ID" \
  -d "{\"product_id\":\"$PRODUCT_ID\",\"photo\":\"demo-photo\",\"quantity\":$QUANTITY,\"start_price\":$START_PRICE,\"auction_starts_at\":\"$STARTS_AT\",\"auction_duration_minutes\":$DURATION_MIN}")"

echo "$resp"
lot_id="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get("lot_id",""))' "$resp")"
if [[ -n "$lot_id" ]]; then
  echo "lot_id=$lot_id"
fi
