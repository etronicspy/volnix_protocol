#!/bin/bash
# Sequential start: node0 first (produces blocks alone), then node1 syncs and joins.
# Phase 1: genesis with 1 validator (node0) — node0 produces blocks.
# Phase 2: node1 joins as full node, syncs from node0.
# Proves late-joiner sync works. For multi-validator, genesis would need ValidatorUpdates.
# Usage: ./scripts/testnet-sequential-start.sh

set -e
cd "$(dirname "$0")/.."

if [ ! -f build/volnixd ]; then
  echo "❌ Build first: go build -o build/volnixd ./cmd/volnixd"
  exit 1
fi

echo "🛑 Stopping any running nodes..."
pkill -f "volnixd start" 2>/dev/null || true
sleep 2

echo "🗑️  Resetting testnet..."
rm -rf testnet/node0/data testnet/node1/data testnet/node2/data testnet/node3/data
rm -f testnet/node*/config/addrbook.json 2>/dev/null || true

# Phase 1: single-validator genesis (node0 only) so node0 can produce blocks alone
# CRITICAL: node0 and node1 MUST have byte-identical genesis for AppHash to match during sync
GENESIS_FULL="testnet/node0/config/genesis.json"
GENESIS_BACKUP="testnet/node0/config/genesis.json.bak"
cp "$GENESIS_FULL" "$GENESIS_BACKUP"
jq -c '.validators = [.validators[0]]' "$GENESIS_FULL" > "${GENESIS_FULL}.tmp" && mv "${GENESIS_FULL}.tmp" "$GENESIS_FULL"
cp "$GENESIS_FULL" testnet/node1/config/genesis.json
# Ensure identical (jq -c strips whitespace; both must match)
if ! diff -q "$GENESIS_FULL" testnet/node1/config/genesis.json > /dev/null; then
  echo "❌ Genesis files differ!"
  exit 1
fi
echo "   Genesis: 1 validator (node0), identical on both nodes"

mkdir -p testnet/logs

echo ""
echo "📌 Phase 1: Starting node0 alone (single validator)..."
VOLNIX_GRPC_PORT=9090 ./build/volnixd start --home testnet/node0 > testnet/logs/node0.log 2>&1 &
NODE0_PID=$!
echo "   node0 PID: $NODE0_PID"

echo "⏳ Waiting 25s for node0 to produce blocks..."
sleep 25

HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')
echo "   node0 block height: $HEIGHT"

if [ "$HEIGHT" = "0" ] || [ "$HEIGHT" = "1" ]; then
  echo "❌ node0 failed to produce blocks. Restoring genesis."
  mv "$GENESIS_BACKUP" "$GENESIS_FULL"
  cp "$GENESIS_FULL" testnet/node1/config/genesis.json
  kill $NODE0_PID 2>/dev/null || true
  exit 1
fi

echo ""
echo "📌 Phase 2: Starting node1 (full node, will sync from node0)..."
VOLNIX_GRPC_PORT=9091 ./build/volnixd start --home testnet/node1 > testnet/logs/node1.log 2>&1 &
NODE1_PID=$!
echo "   node1 PID: $NODE1_PID"

echo "⏳ Waiting 30s for node1 to sync..."
sleep 30

HEIGHT0=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')
HEIGHT1=$(curl -s http://localhost:26667/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "0"')
PEERS0=$(curl -s http://localhost:26657/net_info 2>/dev/null | jq -r '.result.n_peers // "?"')
PEERS1=$(curl -s http://localhost:26667/net_info 2>/dev/null | jq -r '.result.n_peers // "?"')

echo ""
echo "=== Result ==="
echo "   node0: height=$HEIGHT0, peers=$PEERS0"
echo "   node1: height=$HEIGHT1, peers=$PEERS1"

# Restore full genesis for next runs
mv "$GENESIS_BACKUP" "$GENESIS_FULL"
cp "$GENESIS_FULL" testnet/node1/config/genesis.json 2>/dev/null || true

if [ "$HEIGHT1" != "0" ] && [ "$HEIGHT1" != "1" ]; then
  echo "✅ node1 synced to height $HEIGHT1 (node0 at $HEIGHT0)"
  echo "   Late-joiner sync works. Sequential start validated."
else
  echo "⚠️  node1 may still be syncing. Check testnet/logs/node1.log"
fi
