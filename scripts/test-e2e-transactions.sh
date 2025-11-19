#!/bin/bash

# End-to-End тест для проверки транзакций и балансов

set -e  # Exit on error

echo "═══════════════════════════════════════════════════════════════"
echo "🧪 E2E TEST: Transactions & Balances"
echo "═══════════════════════════════════════════════════════════════"
echo ""

RPC="http://localhost:26657"
SENDER="volnix19rl4cm2hmr8afy4kldpxz3fka4jguq0a9r0ces"
RECIPIENT="volnix1abc123def456"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

test_pass() {
    echo -e "${GREEN}✅ PASS:${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

test_fail() {
    echo -e "${RED}❌ FAIL:${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: Node Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

STATUS=$(curl -s "$RPC/status" 2>&1)
if echo "$STATUS" | grep -q "latest_block_height"; then
    LATEST_HEIGHT=$(echo "$STATUS" | python3 -c "import sys, json; print(json.load(sys.stdin)['result']['sync_info']['latest_block_height'])" 2>/dev/null)
    test_pass "Node is running (block: $LATEST_HEIGHT)"
else
    test_fail "Node is not responding"
    exit 1
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: Send Transaction"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

cd "$(dirname "$0")/../frontend/wallet-ui"
TX_OUTPUT=$(node scripts/test-send-direct.js 2>&1)

if echo "$TX_OUTPUT" | grep -q "Code: 0"; then
    TX_HASH=$(echo "$TX_OUTPUT" | grep "Hash:" | awk '{print $2}')
    TX_HEIGHT=$(echo "$TX_OUTPUT" | grep "Height:" | awk '{print $2}')
    test_pass "Transaction sent (Hash: ${TX_HASH:0:16}..., Height: $TX_HEIGHT)"
else
    test_fail "Transaction failed to send"
    echo "$TX_OUTPUT"
    exit 1
fi

sleep 3  # Wait for block

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 3: Transaction in Block"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

BLOCK_DATA=$(curl -s "$RPC/block?height=$TX_HEIGHT" 2>/dev/null)
TX_COUNT=$(echo "$BLOCK_DATA" | python3 -c "import sys, json; print(len(json.load(sys.stdin).get('result', {}).get('block', {}).get('data', {}).get('txs', [])))" 2>/dev/null)

if [ "$TX_COUNT" -gt 0 ]; then
    test_pass "Transaction found in block $TX_HEIGHT"
else
    test_fail "Transaction NOT in block"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 4: Transfer Events"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

BLOCK_RESULTS=$(curl -s "$RPC/block_results?height=$TX_HEIGHT" 2>/dev/null)

HAS_TRANSFER=$(echo "$BLOCK_RESULTS" | python3 << 'PYEOF'
import sys, json
data = json.load(sys.stdin)
tx_results = data.get('result', {}).get('txs_results', [])
if tx_results:
    events = tx_results[0].get('events', [])
    for event in events:
        if event.get('type') == 'transfer':
            print('true')
            sys.exit(0)
print('false')
PYEOF
)

if [ "$HAS_TRANSFER" = "true" ]; then
    test_pass "transfer event exists"
else
    test_fail "transfer event missing"
fi

HAS_COIN_SPENT=$(echo "$BLOCK_RESULTS" | python3 -c "import sys, json; events = json.load(sys.stdin).get('result', {}).get('txs_results', [{}])[0].get('events', []); print('true' if any(e.get('type') == 'coin_spent' for e in events) else 'false')" 2>/dev/null)

if [ "$HAS_COIN_SPENT" = "true" ]; then
    test_pass "coin_spent event exists"
else
    test_fail "coin_spent event missing"
fi

HAS_COIN_RECEIVED=$(echo "$BLOCK_RESULTS" | python3 -c "import sys, json; events = json.load(sys.stdin).get('result', {}).get('txs_results', [{}])[0].get('events', []); print('true' if any(e.get('type') == 'coin_received' for e in events) else 'false')" 2>/dev/null)

if [ "$HAS_COIN_RECEIVED" = "true" ]; then
    test_pass "coin_received event exists"
else
    test_fail "coin_received event missing"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 5: Event Attributes"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

EVENT_DATA=$(echo "$BLOCK_RESULTS" | python3 << 'PYEOF'
import sys, json, base64

data = json.load(sys.stdin)
tx_results = data.get('result', {}).get('txs_results', [])

if tx_results:
    events = tx_results[0].get('events', [])
    for event in events:
        if event.get('type') == 'transfer':
            attrs = event.get('attributes', [])
            attr_dict = {}
            
            for attr in attrs:
                # Проверяем index
                if attr.get('index') == True:
                    key = attr.get('key', '')
                    value = attr.get('value', '')
                else:
                    try:
                        key = base64.b64decode(attr.get('key', '')).decode('utf-8')
                        value = base64.b64decode(attr.get('value', '')).decode('utf-8')
                    except:
                        key = attr.get('key', '')
                        value = attr.get('value', '')
                
                attr_dict[key] = value
            
            print(f"{attr_dict.get('sender', 'MISSING')}")
            print(f"{attr_dict.get('recipient', 'MISSING')}")
            print(f"{attr_dict.get('amount', 'MISSING')}")
            break
PYEOF
)

EVENT_SENDER=$(echo "$EVENT_DATA" | sed -n '1p')
EVENT_RECIPIENT=$(echo "$EVENT_DATA" | sed -n '2p')
EVENT_AMOUNT=$(echo "$EVENT_DATA" | sed -n '3p')

if [ "$EVENT_SENDER" = "$SENDER" ]; then
    test_pass "Event sender matches"
else
    test_fail "Event sender mismatch (expected: $SENDER, got: $EVENT_SENDER)"
fi

if [ "$EVENT_RECIPIENT" = "$RECIPIENT" ]; then
    test_pass "Event recipient matches"
else
    test_fail "Event recipient mismatch (expected: $RECIPIENT, got: $EVENT_RECIPIENT)"
fi

if echo "$EVENT_AMOUNT" | grep -q "uwrt"; then
    test_pass "Event amount has correct format"
else
    test_fail "Event amount format incorrect (got: $EVENT_AMOUNT)"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 6: /tx?hash= Endpoint"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TX_RESPONSE=$(curl -s "$RPC/tx?hash=0x$TX_HASH" 2>/dev/null)

if echo "$TX_RESPONSE" | grep -q '"result"'; then
    test_pass "Transaction found via /tx?hash="
    
    # Check if result contains events
    HAS_EVENTS=$(echo "$TX_RESPONSE" | python3 -c "import sys, json; events = json.load(sys.stdin).get('result', {}).get('tx_result', {}).get('events', []); print('true' if len(events) > 0 else 'false')" 2>/dev/null)
    
    if [ "$HAS_EVENTS" = "true" ]; then
        test_pass "Transaction has events in /tx response"
    else
        test_fail "Transaction missing events in /tx response"
    fi
else
    test_fail "Transaction NOT found via /tx?hash="
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 7: Balance Updates"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Проверяем баланс получателя через CosmJS
cd "$(dirname "$0")/../frontend/wallet-ui"
RECIPIENT_BALANCE=$(node << 'NODEEOF'
const { StargateClient } = require('@cosmjs/stargate');
const { Comet38Client } = require('@cosmjs/tendermint-rpc');

(async () => {
  try {
    const cometClient = await Comet38Client.connect('http://localhost:26657');
    const client = await StargateClient.create(cometClient);
    
    const balances = await client.getAllBalances('volnix1abc123def456');
    
    const uwrtBalance = balances.find(b => b.denom === 'uwrt');
    console.log(uwrtBalance ? uwrtBalance.amount : '0');
    
    await cometClient.disconnect();
  } catch (e) {
    console.log('ERROR');
  }
})();
NODEEOF
)

if [ "$RECIPIENT_BALANCE" != "0" ] && [ "$RECIPIENT_BALANCE" != "ERROR" ]; then
    test_pass "Recipient balance updated ($RECIPIENT_BALANCE uwrt)"
else
    test_fail "Recipient balance NOT updated (got: $RECIPIENT_BALANCE)"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "📊 TEST RESULTS"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo -e "${GREEN}✅ Passed: $TESTS_PASSED${NC}"
echo -e "${RED}❌ Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED!${NC}"
    exit 0
else
    echo -e "${RED}⚠️  SOME TESTS FAILED${NC}"
    exit 1
fi

