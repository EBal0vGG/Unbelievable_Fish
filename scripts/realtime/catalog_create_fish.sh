#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
require python3

NAME="${1:-Cod}"
DESCRIPTION="${2:-Demo fish}"

resp="$(curl -sS -X POST "$CATALOG_URL/fish" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$NAME\",\"description\":\"$DESCRIPTION\"}")"

echo "$resp"
fish_id="$(python3 -c 'import json,sys; print(json.loads(sys.argv[1]).get("fish_id",""))' "$resp")"
if [[ -n "$fish_id" ]]; then
  echo "fish_id=$fish_id"
fi
