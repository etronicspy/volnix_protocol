#!/bin/bash

# Простой скрипт запуска 3 узлов ПРЯМО СЕЙЧАС

# Получаем абсолютный путь к бинарнику
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="$PROJECT_ROOT/build/volnixd-standalone"

if [ ! -f "$BINARY" ]; then
    echo "❌ Бинарник не найден: $BINARY"
    exit 1
fi

echo "🚀 Запуск 3-нод сети"
echo ""

# Остановка
pkill -f volnixd || true
sleep 2

# Используем существующую инициализацию
BASE_DIR="testnet/node0/.volnix"

if [ ! -d "$BASE_DIR" ]; then
    echo "❌ testnet/node0 не найден"
    echo "Запустите: bash scripts/start-minimal-network.sh"
    exit 1
fi

# Копируем конфигурацию для 3 узлов
rm -rf multinode logs
mkdir -p multinode/node0 multinode/node1 multinode/node2 logs

echo "📦 Копирование конфигураций..."

for i in 0 1 2; do
    cp -r "$BASE_DIR" "multinode/node$i/.volnix"
    # Очищаем БД
    rm -rf "multinode/node$i/.volnix/data"/*.db* 2>/dev/null || true
    echo "  ✅ node$i"
done

# Обновляем genesis на общий chain-id
python3 << 'PYEOF'
import json

for i in range(3):
    genesis_file = f'multinode/node{i}/.volnix/config/genesis.json'
    with open(genesis_file, 'r') as f:
        genesis = json.load(f)
    
    genesis['chain_id'] = 'volnix-multinode'
    
    with open(genesis_file, 'w') as f:
        json.dump(genesis, f, indent=2)
PYEOF

echo ""
echo "🚀 Запуск узлов..."
echo ""

# Получаем node IDs
NODE0_ID=$(cat multinode/node0/.volnix/config/node_key.json | jq -r '.id' 2>/dev/null)
NODE1_ID=$(cat multinode/node1/.volnix/config/node_key.json | jq -r '.id' 2>/dev/null)
NODE2_ID=$(cat multinode/node2/.volnix/config/node_key.json | jq -r '.id' 2>/dev/null)

# Настраиваем persistent_peers
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"${NODE1_ID}@127.0.0.1:26666,${NODE2_ID}@127.0.0.1:26676\"|g" "multinode/node0/.volnix/config/config.toml"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"${NODE0_ID}@127.0.0.1:26656,${NODE2_ID}@127.0.0.1:26676\"|g" "multinode/node1/.volnix/config/config.toml"
sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"${NODE0_ID}@127.0.0.1:26656,${NODE1_ID}@127.0.0.1:26666\"|g" "multinode/node2/.volnix/config/config.toml"

# Запуск
(cd multinode/node0 && VOLNIX_HOME=".volnix" VOLNIX_RPC_PORT=26657 VOLNIX_P2P_PORT=26656 "$BINARY" start > ../../logs/node0.log 2>&1 &)
echo "Node 0: PID $!"
sleep 4

(cd multinode/node1 && VOLNIX_HOME=".volnix" VOLNIX_RPC_PORT=26667 VOLNIX_P2P_PORT=26666 "$BINARY" start > ../../logs/node1.log 2>&1 &)
echo "Node 1: PID $!"
sleep 4

(cd multinode/node2 && VOLNIX_HOME=".volnix" VOLNIX_RPC_PORT=26677 VOLNIX_P2P_PORT=26676 "$BINARY" start > ../../logs/node2.log 2>&1 &)
echo "Node 2: PID $!"

echo ""
echo "⏳ Ожидание (15 секунд)..."
sleep 15

echo ""
echo "🔍 Проверка..."
echo ""

for i in 0 1 2; do
    port=$((26657 + i * 10))
    if curl -s "http://localhost:$port/status" > /dev/null 2>&1; then
        height=$(curl -s "http://localhost:$port/status" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null)
        peers=$(curl -s "http://localhost:$port/net_info" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('n_peers',0))" 2>/dev/null)
        echo "✅ Node $i: блок $height, peers $peers"
    else
        echo "❌ Node $i: не отвечает"
    fi
done

echo ""
echo "Логи: tail -f logs/node*.log"

