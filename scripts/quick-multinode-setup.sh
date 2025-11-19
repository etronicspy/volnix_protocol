#!/bin/bash

# Быстрая настройка 3-нод сети

set -e

TESTNET_DIR="testnet-quick"
CHAIN_ID="volnix-testnet"
BINARY="./build/volnixd-standalone"

echo "🚀 Быстрая настройка мультинод сети"
echo ""

# Остановка и очистка
pkill -f volnixd || true
sleep 1
rm -rf "$TESTNET_DIR" logs
mkdir -p "$TESTNET_DIR" logs

echo "📦 Инициализация 3 узлов..."

# Инициализация node0
mkdir -p "$TESTNET_DIR/node0"
(cd "$TESTNET_DIR/node0" && VOLNIX_HOME=".volnix" "$BINARY" init node0 > /dev/null 2>&1) &
sleep 2
pkill -f "volnixd-standalone init" || true
sleep 1

# Инициализация node1  
mkdir -p "$TESTNET_DIR/node1"
(cd "$TESTNET_DIR/node1" && VOLNIX_HOME=".volnix" "$BINARY" init node1 > /dev/null 2>&1) &
sleep 2
pkill -f "volnixd-standalone init" || true
sleep 1

# Инициализация node2
mkdir -p "$TESTNET_DIR/node2"
(cd "$TESTNET_DIR/node2" && VOLNIX_HOME=".volnix" "$BINARY" init node2 > /dev/null 2>&1) &
sleep 2
pkill -f "volnixd-standalone init" || true
sleep 1

echo "✅ Узлы инициализированы"
echo ""

echo "📝 Сборка общего genesis..."

# Собираем validator keys запуская узлы кратко
for i in 0 1 2; do
    if [ ! -f "$TESTNET_DIR/node$i/.volnix/config/priv_validator_key.json" ]; then
        (cd "$TESTNET_DIR/node$i" && VOLNIX_HOME=".volnix" VOLNIX_RPC_PORT=$((29000+i)) VOLNIX_P2P_PORT=$((29100+i)) "$BINARY" start > /dev/null 2>&1) &
        TEMP_PID=$!
        sleep 2
        kill $TEMP_PID 2>/dev/null || true
    fi
done

sleep 1

# Создаем общий genesis вручную
python3 << 'PYEOF'
import json
import os

TESTNET_DIR = "testnet-quick"
CHAIN_ID = "volnix-testnet"

# Читаем genesis первого узла
with open(f'{TESTNET_DIR}/node0/.volnix/config/genesis.json', 'r') as f:
    genesis = json.load(f)

# Обновляем chain_id
genesis['chain_id'] = CHAIN_ID

# Собираем всех валидаторов
all_validators = []

for i in range(3):
    node_genesis = f'{TESTNET_DIR}/node{i}/.volnix/config/genesis.json'
    if os.path.exists(node_genesis):
        with open(node_genesis, 'r') as f:
            node_gen = json.load(f)
            validators = node_gen.get('validators', [])
            all_validators.extend(validators)

# Обновляем validators
genesis['validators'] = all_validators

# Сохраняем общий genesis
shared_genesis = f'{TESTNET_DIR}/genesis.json'
with open(shared_genesis, 'w') as f:
    json.dump(genesis, f, indent=2)

# Копируем на все узлы
for i in range(3):
    node_genesis = f'{TESTNET_DIR}/node{i}/.volnix/config/genesis.json'
    with open(shared_genesis, 'r') as f:
        genesis_data = f.read()
    with open(node_genesis, 'w') as f:
        f.write(genesis_data)

print(f"✅ Genesis создан с {len(all_validators)} валидаторами")
PYEOF

echo ""
echo "🔗 Настройка persistent_peers..."

# Получаем node IDs
NODE0_ID=$(cat "$TESTNET_DIR/node0/.volnix/config/node_key.json" | jq -r '.id' 2>/dev/null || echo "node0")
NODE1_ID=$(cat "$TESTNET_DIR/node1/.volnix/config/node_key.json" | jq -r '.id' 2>/dev/null || echo "node1")
NODE2_ID=$(cat "$TESTNET_DIR/node2/.volnix/config/node_key.json" | jq -r '.id' 2>/dev/null || echo "node2")

# Настраиваем peers для каждого узла
# Node0 -> connect to node1, node2
PEERS0="${NODE1_ID}@127.0.0.1:26666,${NODE2_ID}@127.0.0.1:26676"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS0\"|g" "$TESTNET_DIR/node0/.volnix/config/config.toml" 2>/dev/null || true

# Node1 -> connect to node0, node2  
PEERS1="${NODE0_ID}@127.0.0.1:26656,${NODE2_ID}@127.0.0.1:26676"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS1\"|g" "$TESTNET_DIR/node1/.volnix/config/config.toml" 2>/dev/null || true

# Node2 -> connect to node0, node1
PEERS2="${NODE0_ID}@127.0.0.1:26656,${NODE1_ID}@127.0.0.1:26666"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS2\"|g" "$TESTNET_DIR/node2/.volnix/config/config.toml" 2>/dev/null || true

echo "✅ Persistent peers настроены"
echo ""

echo "✅ Готово к запуску!"
echo ""
echo "Запуск:"
echo "  ./scripts/launch-multinode.sh"

