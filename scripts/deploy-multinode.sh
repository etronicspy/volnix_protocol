#!/bin/bash

# Полная настройка и запуск 3-нод сети
# Все в одном скрипте

set -e

TESTNET_DIR="testnet-multinode-final"
CHAIN_ID="volnix-testnet"
BINARY="./build/volnixd-standalone"

echo "🚀 Развертывание 3-нод сети Volnix Protocol"
echo "============================================"
echo ""

# Остановка и очистка
echo "🛑 Очистка..."
pkill -f volnixd || true
sleep 2
rm -rf "$TESTNET_DIR" logs
mkdir -p "$TESTNET_DIR" logs

echo "✅ Очищено"
echo ""

#═══════════════════════════════════════════════════════════════
# ШАГ 1: Инициализация узлов
#═══════════════════════════════════════════════════════════════

echo "📦 Шаг 1/5: Инициализация узлов..."
echo ""

for i in 0 1 2; do
    echo "  Инициализация node$i..."
    mkdir -p "$TESTNET_DIR/node$i"
    (cd "$TESTNET_DIR/node$i" && VOLNIX_HOME=".volnix" "$BINARY" init "node$i" > /dev/null 2>&1)
done

echo "  ✅ Все узлы инициализированы"
echo ""

#═══════════════════════════════════════════════════════════════
# ШАГ 2: Создание validator keys (запуск и остановка)
#═══════════════════════════════════════════════════════════════

echo "🔑 Шаг 2/5: Создание validator keys..."
echo ""

for i in 0 1 2; do
    if [ ! -f "$TESTNET_DIR/node$i/.volnix/config/priv_validator_key.json" ]; then
        echo "  Генерация ключей для node$i..."
        (cd "$TESTNET_DIR/node$i" && \
         VOLNIX_HOME=".volnix" \
         VOLNIX_RPC_PORT=$((29000+i)) \
         VOLNIX_P2P_PORT=$((29100+i)) \
         "$BINARY" start > /dev/null 2>&1) &
        TEMP_PID=$!
        sleep 3
        kill $TEMP_PID 2>/dev/null || true
        wait $TEMP_PID 2>/dev/null || true
    fi
done

echo "  ✅ Validator keys созданы"
echo ""

#═══════════════════════════════════════════════════════════════
# ШАГ 3: Создание общего genesis
#═══════════════════════════════════════════════════════════════

echo "📝 Шаг 3/5: Создание общего genesis..."
echo ""

python3 << 'PYEOF'
import json
import os

TESTNET_DIR = "testnet-multinode-final"
CHAIN_ID = "volnix-testnet"

# Читаем genesis первого узла как базу
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
            for val in validators:
                # Обновляем name для читаемости
                val['name'] = f'node{i}'
            all_validators.extend(validators)

# Устанавливаем всех валидаторов
genesis['validators'] = all_validators

# Сохраняем общий genesis
shared_genesis = f'{TESTNET_DIR}/genesis.json'
with open(shared_genesis, 'w') as f:
    json.dump(genesis, f, indent=2)

print(f"  ✅ Создан genesis с {len(all_validators)} валидаторами")

# Копируем общий genesis на все узлы
for i in range(3):
    node_genesis = f'{TESTNET_DIR}/node{i}/.volnix/config/genesis.json'
    with open(shared_genesis, 'r') as f:
        genesis_data = f.read()
    with open(node_genesis, 'w') as f:
        f.write(genesis_data)
    print(f"  ✅ Genesis скопирован на node{i}")
PYEOF

echo ""

#═══════════════════════════════════════════════════════════════
# ШАГ 4: Настройка persistent_peers
#═══════════════════════════════════════════════════════════════

echo "🔗 Шаг 4/5: Настройка P2P соединений..."
echo ""

# Получаем node IDs
if [ -f "$TESTNET_DIR/node0/.volnix/config/node_key.json" ]; then
    NODE0_ID=$(cat "$TESTNET_DIR/node0/.volnix/config/node_key.json" | jq -r '.id' 2>/dev/null)
fi
if [ -f "$TESTNET_DIR/node1/.volnix/config/node_key.json" ]; then
    NODE1_ID=$(cat "$TESTNET_DIR/node1/.volnix/config/node_key.json" | jq -r '.id' 2>/dev/null)
fi
if [ -f "$TESTNET_DIR/node2/.volnix/config/node_key.json" ]; then
    NODE2_ID=$(cat "$TESTNET_DIR/node2/.volnix/config/node_key.json" | jq -r '.id' 2>/dev/null)
fi

echo "  Node IDs:"
echo "    node0: $NODE0_ID"
echo "    node1: $NODE1_ID"
echo "    node2: $NODE2_ID"
echo ""

# Настраиваем peers
# Node0 соединяется с node1 и node2
PEERS0="${NODE1_ID}@127.0.0.1:26666,${NODE2_ID}@127.0.0.1:26676"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS0\"|g" "$TESTNET_DIR/node0/.volnix/config/config.toml" 2>/dev/null

# Node1 соединяется с node0 и node2
PEERS1="${NODE0_ID}@127.0.0.1:26656,${NODE2_ID}@127.0.0.1:26676"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS1\"|g" "$TESTNET_DIR/node1/.volnix/config/config.toml" 2>/dev/null

# Node2 соединяется с node0 и node1
PEERS2="${NODE0_ID}@127.0.0.1:26656,${NODE1_ID}@127.0.0.1:26666"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS2\"|g" "$TESTNET_DIR/node2/.volnix/config/config.toml" 2>/dev/null

echo "  ✅ Persistent peers настроены"
echo ""

#═══════════════════════════════════════════════════════════════
# ШАГ 5: Запуск узлов
#═══════════════════════════════════════════════════════════════

echo "🚀 Шаг 5/5: Запуск узлов..."
echo ""

declare -a PIDS

for i in 0 1 2; do
    node_name="node$i"
    node_dir="$TESTNET_DIR/$node_name"
    rpc_port=$((26657 + i * 10))
    p2p_port=$((26656 + i * 10))
    log_file="logs/${node_name}.log"
    
    echo "🚀 Запуск $node_name (RPC: $rpc_port, P2P: $p2p_port)..."
    
    (cd "$node_dir" && \
     VOLNIX_HOME=".volnix" \
     VOLNIX_RPC_PORT=$rpc_port \
     VOLNIX_P2P_PORT=$p2p_port \
     "$BINARY" start > "../../$log_file" 2>&1 &)
    
    PID=$!
    PIDS[$i]=$PID
    echo "  PID: $PID"
    
    sleep 4
done

echo ""
echo "✅ Все узлы запущены!"
echo ""

# Сохранение PIDs
echo "${PIDS[@]}" > "$TESTNET_DIR/pids.txt"

# Ожидание запуска
echo "⏳ Ожидание полного запуска (20 секунд)..."
sleep 20

#═══════════════════════════════════════════════════════════════
# ПРОВЕРКА
#═══════════════════════════════════════════════════════════════

echo ""
echo "🔍 Проверка статуса..."
echo ""

for i in 0 1 2; do
    rpc_port=$((26657 + i * 10))
    
    echo "Node $i (http://localhost:$rpc_port):"
    
    if curl -s "http://localhost:$rpc_port/status" > /dev/null 2>&1; then
        HEIGHT=$(curl -s "http://localhost:$rpc_port/status" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null)
        PEERS=$(curl -s "http://localhost:$rpc_port/net_info" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('n_peers',0))" 2>/dev/null)
        echo "  ✅ Работает"
        echo "  Блок: $HEIGHT"
        echo "  Peers: $PEERS"
    else
        echo "  ❌ Не отвечает"
    fi
    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Развертывание завершено!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Endpoints:"
echo "  Node 0: http://localhost:26657"
echo "  Node 1: http://localhost:26667"
echo "  Node 2: http://localhost:26677"
echo ""
echo "Логи:"
echo "  tail -f logs/node0.log"
echo "  tail -f logs/node1.log"
echo "  tail -f logs/node2.log"
echo ""
echo "Остановка:"
echo "  kill ${PIDS[@]}"
echo ""


