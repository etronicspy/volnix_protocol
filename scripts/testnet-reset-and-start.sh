#!/bin/bash
# Reset and start 3-node Volnix testnet.
# Ensures allow_duplicate_ip=true so late joiners can connect (all nodes on 127.0.0.1).
# Usage: ./scripts/testnet-reset-and-start.sh

set -e
cd "$(dirname "$0")/.."

echo "🛑 Stopping any running nodes..."
pkill -f "volnixd start" 2>/dev/null || true
sleep 2

echo "🗑️  Clearing node data and addrbook..."
rm -rf testnet/node0/data testnet/node1/data testnet/node2/data testnet/node3/data testnet/node4/data testnet/node5/data testnet/node6/data
rm -f testnet/node*/config/addrbook.json 2>/dev/null || true

echo "✅ Reset complete."
echo ""
echo "Start nodes in 4 terminals (within ~10 seconds):"
echo "  Use VOLNIX_SKIP_VALIDATOR_UPDATES=1 for multi-validator consensus."
echo ""
echo "  Terminal 1: VOLNIX_SKIP_VALIDATOR_UPDATES=1 ./build/volnixd start --home testnet/node0  # gRPC 9090, RPC 26657"
echo "  Terminal 2: VOLNIX_SKIP_VALIDATOR_UPDATES=1 VOLNIX_GRPC_PORT=9091 ./build/volnixd start --home testnet/node1  # RPC 26667"
echo "  Terminal 3: VOLNIX_SKIP_VALIDATOR_UPDATES=1 VOLNIX_GRPC_PORT=9092 ./build/volnixd start --home testnet/node2  # RPC 26677"
echo "  Terminal 4: VOLNIX_SKIP_VALIDATOR_UPDATES=1 VOLNIX_GRPC_PORT=9093 ./build/volnixd start --home testnet/node3  # RPC 26687"
echo "  Terminal 5: VOLNIX_GRPC_PORT=9094 ./build/volnixd start --home testnet/node4  # RPC 26697 (full)"
echo "  Terminal 6: VOLNIX_GRPC_PORT=9095 ./build/volnixd start --home testnet/node5  # RPC 26707 (full)"
echo "  Terminal 7: VOLNIX_GRPC_PORT=9096 ./build/volnixd start --home testnet/node6  # RPC 26717 (full)"
echo ""
echo "Verify: curl http://localhost:26657/net_info | jq '.result.n_peers'  # expect 3-4"
echo "        curl http://localhost:26697/net_info | jq '.result.n_peers'  # node4"
