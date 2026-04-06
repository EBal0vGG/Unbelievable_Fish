#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <lot_id>"
  exit 2
fi

LOT_ID="$1"

curl -sS -i -X POST "$CATALOG_URL/lots/$LOT_ID/publish"
echo
