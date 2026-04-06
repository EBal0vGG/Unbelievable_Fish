#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"

if [[ $# -lt 1 ]]; then
  print_usage_header
  echo "Usage: $0 <deal_id> [company_id] [user_id]"
  exit 2
fi

DEAL_ID="$1"
COMPANY_ID="${2:-seller-1}"
USER_ID="${3:-manager-1}"

curl -sS -i -X POST "$DEALS_URL/deals/$DEAL_ID/confirm" \
  -H "X-Company-ID: $COMPANY_ID" \
  -H "X-User-ID: $USER_ID"
echo
