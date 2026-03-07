#!/bin/bash
# Reset and start 3-node Volnix testnet.
# Ensures allow_duplicate_ip=true so late joiners can connect (all nodes on 127.0.0.1).
# Usage: ./scripts/testnet-reset-and-start.sh

set -eu
cd "$(dirname "$0")/.."

echo "🛑 Stopping any running nodes..."
pkill -f "volnixd start" 2>/dev/null || true
sleep 2

echo "🗑️  Clearing node data and addrbook..."
rm -rf testnet/node0/data testnet/node1/data testnet/node2/data testnet/node3/data testnet/node4/data testnet/node5/data testnet/node6/data
rm -f testnet/node*/config/addrbook.json 2>/dev/null || true

echo "✅ Reset complete."
echo ""
echo "Start nodes in 3 terminals (within ~10 seconds):"
echo ""
echo "  Terminal 1: ./build/volnixd start --home testnet/node0  # gRPC 9090, RPC 26657"
echo "  Terminal 2: VOLNIX_GRPC_PORT=9091 ./build/volnixd start --home testnet/node1  # RPC 26667"
echo "  Terminal 3: VOLNIX_GRPC_PORT=9092 ./build/volnixd start --home testnet/node2  # RPC 26677"
echo ""
echo "Verify: curl http://localhost:26657/net_info | jq '.result.n_peers'  # expect 2"
echo "        curl http://localhost:26677/net_info | jq '.result.n_peers'  # node2"
