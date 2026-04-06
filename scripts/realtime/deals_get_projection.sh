#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <auction_id>"
  exit 2
fi

AUCTION_ID="$1"

curl -sS -i "$DEALS_URL/deal-projections/$AUCTION_ID"
echo
