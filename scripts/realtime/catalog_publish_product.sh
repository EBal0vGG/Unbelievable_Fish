#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <product_id>"
  echo "Requires CATALOG_SELLER_TOKEN (seller JWT)."
  exit 2
fi

if [[ -z "${CATALOG_SELLER_TOKEN:-}" ]]; then
  echo "Set CATALOG_SELLER_TOKEN to a seller JWT." >&2
  exit 2
fi

PRODUCT_ID="$1"

curl -sS -i -X POST "$CATALOG_URL/products/$PRODUCT_ID/publish" \
  -H "Authorization: Bearer $CATALOG_SELLER_TOKEN"
echo
