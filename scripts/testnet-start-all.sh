#!/bin/bash
# Start all 4 validator nodes simultaneously to avoid round desync.
# Nodes must start within 1-2 seconds so they're in the same round when block 2 is proposed.
# Usage: ./scripts/testnet-start-all.sh
# Prerequisite: run ./scripts/testnet-reset-and-start.sh first (or have testnet reset)

set -e
cd "$(dirname "$0")/.."

if [ ! -f build/volnixd ]; then
  echo "❌ Build first: go build -o build/volnixd ./cmd/volnixd"
  exit 1
fi

echo "🛑 Stopping any running nodes..."
pkill -f "volnixd start" 2>/dev/null || true
sleep 2

mkdir -p testnet/logs
export VOLNIX_SKIP_VALIDATOR_UPDATES=1

echo "🚀 Starting all 4 nodes simultaneously..."
VOLNIX_GRPC_PORT=9090 ./build/volnixd start --home testnet/node0 > testnet/logs/node0.log 2>&1 &
PID0=$!
VOLNIX_GRPC_PORT=9091 ./build/volnixd start --home testnet/node1 > testnet/logs/node1.log 2>&1 &
PID1=$!
VOLNIX_GRPC_PORT=9092 ./build/volnixd start --home testnet/node2 > testnet/logs/node2.log 2>&1 &
PID2=$!
VOLNIX_GRPC_PORT=9093 ./build/volnixd start --home testnet/node3 > testnet/logs/node3.log 2>&1 &
PID3=$!

echo "   PIDs: $PID0 $PID1 $PID2 $PID3"
echo "   Logs: testnet/logs/node{0,1,2,3}.log"
echo ""
echo "⏳ Waiting 20s for consensus..."
sleep 20

HEIGHT=$(curl -s http://localhost:26657/status 2>/dev/null | jq -r '.result.sync_info.latest_block_height // "error"')
echo "📊 Block height: $HEIGHT"
if [ "$HEIGHT" != "error" ] && [ "$HEIGHT" != "1" ]; then
  echo "✅ Blocks are being produced!"
else
  echo "⚠️  Height still 1 — check testnet/logs/*.log for [CONSENSUS_DEBUG]"
fi
