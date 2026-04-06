#!/usr/bin/env bash
set -euo pipefail

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; exit 1; }
}

require curl

CATALOG_URL="${CATALOG_URL:-http://localhost:8081}"
TRADING_URL="${TRADING_URL:-http://localhost:8082}"
DEALS_URL="${DEALS_URL:-http://localhost:8083}"

print_usage_header() {
  echo "Realtime endpoint script"
  echo "Override URLs via CATALOG_URL/TRADING_URL/DEALS_URL"
}
