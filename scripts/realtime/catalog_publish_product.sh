#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <product_id>"
  exit 2
fi

PRODUCT_ID="$1"

curl -sS -i -X POST "$CATALOG_URL/products/$PRODUCT_ID/publish"
echo
