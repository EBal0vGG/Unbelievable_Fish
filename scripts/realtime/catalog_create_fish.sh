#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
require python3

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/lib/catalog_demo_fish.sh"

NAME="${1:-Cod}"
DESCRIPTION="${2:-Demo fish}"

extra_headers=()
if [[ -n "${CATALOG_ADMIN_TOKEN:-}" ]]; then
  extra_headers=(-H "Authorization: Bearer ${CATALOG_ADMIN_TOKEN}")
fi

resp="$(curl -sS -X POST "$CATALOG_URL/fish" \
  "${extra_headers[@]}" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$NAME\",\"description\":\"$DESCRIPTION\"}")"

echo "$resp"
fish_id="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get("fish_id",""))' "$resp")"
if [[ -n "$fish_id" ]]; then
  echo "fish_id=$fish_id"
else
  fish_id="$(catalog_demo_fish_id "$CATALOG_URL")"
  echo "fish_id=$fish_id  # from GET /fish (CreateFish is admin-only; set CATALOG_ADMIN_TOKEN to create new)"
fi
