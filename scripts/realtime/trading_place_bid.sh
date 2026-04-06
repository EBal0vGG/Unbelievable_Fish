#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <auction_id> [amount] [company_id] [user_id]"
  exit 2
fi

AUCTION_ID="$1"
AMOUNT="${2:-120}"
COMPANY_ID="${3:-buyer-1}"
USER_ID="${4:-user-1}"

curl -sS -i -X POST "$TRADING_URL/auctions/$AUCTION_ID/bids" \
  -H "Content-Type: application/json" \
  -H "X-Company-ID: $COMPANY_ID" \
  -H "X-User-ID: $USER_ID" \
  -d "{\"amount\":$AMOUNT}"
echo
