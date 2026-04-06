#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
require python3

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <fish_id> [weight] [unit] [size] [processing_type]"
  exit 2
fi

FISH_ID="$1"
WEIGHT="${2:-10}"
UNIT="${3:-kg}"
SIZE="${4:-M}"
PROCESSING_TYPE="${5:-frozen}"

resp="$(curl -sS -X POST "$CATALOG_URL/products" \
  -H "Content-Type: application/json" \
  -d "{\"fish_id\":\"$FISH_ID\",\"weight\":$WEIGHT,\"unit\":\"$UNIT\",\"size\":\"$SIZE\",\"processing_type\":\"$PROCESSING_TYPE\"}")"

echo "$resp"
product_id="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get("product_id",""))' "$resp")"
if [[ -n "$product_id" ]]; then
  echo "product_id=$product_id"
fi
