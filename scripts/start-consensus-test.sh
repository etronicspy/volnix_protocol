#!/bin/bash

# Скрипт для запуска нескольких узлов для проверки консенсуса
# Запускает 3 узла на разных портах

set -e

echo "🚀 Запуск узлов для проверки консенсуса"
echo "========================================"
echo ""

# Получаем абсолютный путь к корню проекта
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Проверяем наличие бинарника
BINARY="$PROJECT_ROOT/build/volnixd-standalone"
if [ ! -f "$BINARY" ]; then
    echo "❌ Бинарник не найден: $BINARY"
    echo "Собираю бинарник..."
    make build-standalone || go build -o "$PROJECT_ROOT/build/volnixd-standalone" ./cmd/volnixd-standalone
fi

# Останавливаем существующие узлы
echo "🛑 Останавливаю существующие узлы..."
pkill -f "volnixd-standalone" || true
sleep 2

# Создаем директории для узлов
NODES_DIR="testnet-consensus"
mkdir -p "$NODES_DIR"

# Массивы для хранения информации об узлах
declare -a NODE_NAMES
declare -a NODE_DIRS
declare -a NODE_RPC_PORTS
declare -a NODE_P2P_PORTS
declare -a NODE_PIDS

# Настраиваем узлы
echo "📦 Настройка узлов..."
for i in 0 1 2; do
    node_name="node$i"
    node_dir="$NODES_DIR/$node_name"
    rpc_port=$((26657 + i * 100))
    p2p_port=$((26656 + i * 100))
    
    NODE_NAMES[$i]=$node_name
    NODE_DIRS[$i]=$node_dir
    NODE_RPC_PORTS[$i]=$rpc_port
    NODE_P2P_PORTS[$i]=$p2p_port
    
    echo "🔧 Настройка $node_name (RPC: $rpc_port, P2P: $p2p_port)..."
    
    # Создаем директорию
    mkdir -p "$node_dir/.volnix/config"
    mkdir -p "$node_dir/.volnix/data"
    
    # Инициализируем узел если нужно
    if [ ! -f "$node_dir/.volnix/config/config.toml" ]; then
        echo "   Инициализация $node_name..."
        (cd "$node_dir" && VOLNIX_HOME=".volnix" "$BINARY" init "$node_name" > /dev/null 2>&1 || true)
        
        # Настраиваем порты в config.toml
        if [ -f "$node_dir/.volnix/config/config.toml" ]; then
            # Используем sed для изменения портов (macOS совместимый)
            sed -i '' "s|laddr = \"tcp://0.0.0.0:26657\"|laddr = \"tcp://0.0.0.0:$rpc_port\"|g" "$node_dir/.volnix/config/config.toml" || true
            sed -i '' "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:$p2p_port\"|g" "$node_dir/.volnix/config/config.toml" || true
        fi
    else
        # Обновляем порты если конфиг уже существует
        sed -i '' "s|laddr = \"tcp://0.0.0.0:26657\"|laddr = \"tcp://0.0.0.0:$rpc_port\"|g" "$node_dir/.volnix/config/config.toml" || true
        sed -i '' "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:$p2p_port\"|g" "$node_dir/.volnix/config/config.toml" || true
    fi
done

echo ""
echo "🚀 Запуск узлов..."
echo ""

# Запускаем узлы с задержкой
for i in 0 1 2; do
    node_name=${NODE_NAMES[$i]}
    node_dir=${NODE_DIRS[$i]}
    rpc_port=${NODE_RPC_PORTS[$i]}
    
    echo "🚀 Запуск $node_name на порту $rpc_port..."
    
    # Запускаем узел в фоне
    (cd "$node_dir" && VOLNIX_HOME=".volnix" "$BINARY" start > "../${node_name}.log" 2>&1 &)
    NODE_PIDS[$i]=$!
    
    sleep 3
done

echo ""
echo "✅ Все узлы запущены!"
echo "===================="
echo ""
echo "📊 Информация об узлах:"
for i in 0 1 2; do
    echo "  ${NODE_NAMES[$i]}: PID ${NODE_PIDS[$i]}, RPC http://localhost:${NODE_RPC_PORTS[$i]}"
done
echo ""

# Сохраняем PIDs
echo "${NODE_PIDS[0]} ${NODE_PIDS[1]} ${NODE_PIDS[2]}" > "$NODES_DIR/pids.txt"
echo "📝 PIDs сохранены в $NODES_DIR/pids.txt"
echo ""

# Проверяем статус узлов
echo "🔍 Проверка статуса узлов..."
echo ""

for i in 0 1 2; do
    rpc_port=${NODE_RPC_PORTS[$i]}
    node_name=${NODE_NAMES[$i]}
    
    echo "Проверка $node_name (http://localhost:$rpc_port)..."
    if curl -s "http://localhost:$rpc_port/status" > /dev/null 2>&1; then
        height=$(curl -s "http://localhost:$rpc_port/status" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null || echo "0")
        echo "  ✅ Высота блока: $height"
    else
        echo "  ⏳ Узел еще запускается..."
    fi
    echo ""
done

echo ""
echo "🧪 Тестирование консенсуса..."
echo ""

# Ждем несколько блоков
echo "Ожидание создания блоков (15 секунд)..."
sleep 15

# Проверяем высоты блоков на всех узлах
echo "Проверка синхронизации блоков:"
for i in 0 1 2; do
    rpc_port=${NODE_RPC_PORTS[$i]}
    node_name=${NODE_NAMES[$i]}
    
    height=$(curl -s "http://localhost:$rpc_port/status" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null || echo "0")
    echo "  $node_name: блок $height"
done

echo ""
echo "📋 Логи узлов находятся в:"
for i in 0 1 2; do
    echo "  $NODES_DIR/${NODE_NAMES[$i]}.log"
done
echo ""
echo "🛑 Для остановки всех узлов:"
echo "  kill ${NODE_PIDS[0]} ${NODE_PIDS[1]} ${NODE_PIDS[2]}"
echo "  или: pkill -f volnixd-standalone"
echo ""
echo "⚠️  ВАЖНО: Standalone узлы работают независимо и не синхронизируются."
echo "   Каждый узел создает свои собственные блоки."
echo "   Для реального консенсуса нужна полная версия volnixd с настройкой валидаторов."
echo ""
