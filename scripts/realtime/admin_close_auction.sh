#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <auction_id>"
  echo "Optional: PGHOST/PGUSER/PGPASSWORD/PGDATABASE/PGPORT/PGSSLMODE"
  exit 2
fi

AUCTION_ID="$1"

PGHOST="${PGHOST:-localhost}"
PGUSER="${PGUSER:-fish}"
PGPASSWORD="${PGPASSWORD:-fish}"
PGDATABASE="${PGDATABASE:-fish}"
PGPORT="${PGPORT:-5433}"
PGSSLMODE="${PGSSLMODE:-disable}"

PGHOST="$PGHOST" \
PGUSER="$PGUSER" \
PGPASSWORD="$PGPASSWORD" \
PGDATABASE="$PGDATABASE" \
PGPORT="$PGPORT" \
PGSSLMODE="$PGSSLMODE" \
AUCTION_ID="$AUCTION_ID" \
go run ./cmd/admin close-auction
