#!/usr/bin/env bash
# shellcheck shell=bash
# Reusable helpers for demo/e2e scripts. Source from repo scripts after CATALOG_URL is set.
#
# CreateFish is admin-only; catalog bootstraps seed fish on startup. Demo scripts use the first
# bootstrapped fish instead of POST /fish without JWT.

catalog_demo_fish_id() {
  local base="${1:?catalog base url}"
  curl -fsS "$base/fish" | python3 -c 'import json,sys
a=json.load(sys.stdin)
if not a:
  sys.stderr.write(
    "catalog GET /fish is empty — enable bootstrap (CATALOG_BOOTSTRAP_FISH_ENABLED) "
    "or create fish as admin with Authorization\n"
  )
  sys.exit(1)
row=a[0]
print(row.get("fish_id") or row["id"])
'
}
