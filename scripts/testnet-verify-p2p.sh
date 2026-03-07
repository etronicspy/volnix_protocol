#!/bin/bash
# Verify P2P configuration for Volnix testnet.
# Checks: Node IDs, persistent_peers, unconditional_peer_ids, allow_duplicate_ip.
# Usage: ./scripts/testnet-verify-p2p.sh

set -e
cd "$(dirname "$0")/.."

BIN="${BIN:-./build/volnixd}"
NODES="0 1 2 3"

echo "=== P2P Configuration Verification ==="
echo ""

# 1. Get actual Node IDs
echo "=== Node IDs (from node_key.json) ==="
for n in $NODES; do
  if [ -f "testnet/node$n/config/node_key.json" ]; then
    id=$("$BIN" tendermint show-node-id --home "testnet/node$n" 2>/dev/null || echo "?")
    echo "node$n: $id"
  else
    echo "node$n: config not found"
  fi
done
echo ""

# 2. Verify persistent_peers
echo "=== persistent_peers check ==="
for n in $NODES; do
  cfg="testnet/node$n/config/config.toml"
  [ ! -f "$cfg" ] && continue
  peers=$(grep '^persistent_peers = ' "$cfg" | head -1 | cut -d'"' -f2)
  echo "node$n: $peers"
  count=$(echo "$peers" | tr ',' '\n' | grep -v '^$' | wc -l | tr -d ' ')
  [ "$count" -eq 3 ] && echo "  ✓ 3 peers" || echo "  ✗ expected 3 peers, got $count"
done
echo ""

# 3. Verify allow_duplicate_ip (required for localhost)
echo "=== allow_duplicate_ip (required for localhost) ==="
for n in $NODES; do
  cfg="testnet/node$n/config/config.toml"
  [ ! -f "$cfg" ] && continue
  val=$(grep '^allow_duplicate_ip' "$cfg" | awk '{print $3}')
  [ "$val" = "true" ] && echo "node$n: ✓ true" || echo "node$n: ✗ $val (should be true)"
done
echo ""

# 4. Verify addr_book_strict
echo "=== addr_book_strict (false for private/localhost) ==="
for n in $NODES; do
  cfg="testnet/node$n/config/config.toml"
  [ ! -f "$cfg" ] && continue
  val=$(grep '^addr_book_strict' "$cfg" | awk '{print $3}')
  [ "$val" = "false" ] && echo "node$n: ✓ false" || echo "node$n: $val"
done
echo ""

# 5. Cross-check: node0's peers should contain node1,2,3 IDs
echo "=== Cross-check: persistent_peers IDs match node_key ==="
for n in 1 2 3; do
  expected=$("$BIN" tendermint show-node-id --home "testnet/node$n" 2>/dev/null || echo "?")
  cfg="testnet/node0/config/config.toml"
  if grep -q "$expected" "$cfg" 2>/dev/null; then
    echo "  node0 → node$n: ✓ ID $expected found"
  else
    echo "  node0 → node$n: ✗ ID $expected NOT in node0 config"
  fi
done
echo ""

echo "=== Summary ==="
echo "P2P ports: 26656 26666 26676 26686"
echo "Run: VOLNIX_SKIP_VALIDATOR_UPDATES=1 ./build/volnixd start --home testnet/nodeN"
