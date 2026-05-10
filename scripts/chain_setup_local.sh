#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHAIN_DIR="$ROOT_DIR/.chain"
ENV_FILE="$CHAIN_DIR/chain.env"

LOCAL_RPC_URL="${LOCAL_CHAIN_RPC_URL:-http://127.0.0.1:8545}"
INTEGRATION_RPC_URL="${CHAIN_RPC_URL_INTEGRATION:-http://anvil:8545}"
DEFAULT_OPERATOR="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
CHAIN_FROM_ADDRESS="${CHAIN_FROM_ADDRESS:-${CHAIN_OPERATOR_ADDRESS:-$DEFAULT_OPERATOR}}"
CHAIN_CONFIRMATIONS="${CHAIN_CONFIRMATIONS:-1}"
CHAIN_SYNC_INTERVAL_SEC="${CHAIN_SYNC_INTERVAL_SEC:-2}"

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing dependency: $1" >&2
    exit 1
  }
}

require docker
require curl
require npx
require awk

mkdir -p "$CHAIN_DIR"

echo "==> Starting local EVM node (anvil)"
docker compose up -d anvil

echo "==> Waiting for RPC $LOCAL_RPC_URL"
for _ in $(seq 1 60); do
  if curl -sS -X POST "$LOCAL_RPC_URL" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' | grep -q '"result"'; then
    break
  fi
  sleep 1
done

if ! curl -sS -X POST "$LOCAL_RPC_URL" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' | grep -q '"result"'; then
  echo "Anvil RPC is not ready at $LOCAL_RPC_URL" >&2
  exit 1
fi

if [[ ! -d "$ROOT_DIR/contracts/node_modules" ]]; then
  echo "==> Installing contracts dependencies"
  npm --prefix "$ROOT_DIR/contracts" ci
fi

echo "==> Deploying AuctionAnchor contract"
DEPLOY_OUTPUT="$(
  cd "$ROOT_DIR/contracts" && \
    CHAIN_OPERATOR_ADDRESS="$CHAIN_FROM_ADDRESS" \
    npx hardhat run scripts/deploy.js --network localhost
)"
echo "$DEPLOY_OUTPUT"

CONTRACT_ADDRESS="$(printf "%s\n" "$DEPLOY_OUTPUT" | awk '/^address:/ {print $2}' | tail -n 1)"
if [[ -z "$CONTRACT_ADDRESS" ]]; then
  echo "Failed to parse contract address from deploy output" >&2
  exit 1
fi

cat >"$ENV_FILE" <<EOF
CHAIN_ENABLED=true
CHAIN_RPC_URL=$INTEGRATION_RPC_URL
CHAIN_FROM_ADDRESS=$CHAIN_FROM_ADDRESS
CHAIN_CONTRACT_ADDRESS=$CONTRACT_ADDRESS
CHAIN_CONFIRMATIONS=$CHAIN_CONFIRMATIONS
CHAIN_SYNC_INTERVAL_SEC=$CHAIN_SYNC_INTERVAL_SEC
EOF

echo "==> Chain env saved to $ENV_FILE"
cat "$ENV_FILE"
echo
echo "Run: docker compose --env-file .chain/chain.env up -d --build"
