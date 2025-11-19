#!/bin/bash

# Скрипт запуска мультинод сети с общим genesis

set -e

TESTNET_DIR="testnet-multinode"
BINARY="./build/volnixd-standalone"

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== Запуск мультинод сети ===${NC}"
echo ""

# Проверка инициализации
if [ ! -d "$TESTNET_DIR" ]; then
    echo -e "${YELLOW}Сеть не инициализирована. Запускаю setup...${NC}"
    ./scripts/setup-multinode-genesis.sh
fi

# Останов существующих узлов
echo -e "${YELLOW}🛑 Остановка существующих узлов...${NC}"
pkill -f "volnixd-standalone" || true
sleep 2

# Создание директории логов
mkdir -p logs

# Массивы для PIDs
declare -a PIDS

echo -e "${BLUE}🚀 Запуск узлов...${NC}"
echo ""

for i in 0 1 2; do
    node_name="node$i"
    node_dir="$TESTNET_DIR/$node_name"
    rpc_port=$((26657 + i * 10))
    p2p_port=$((26656 + i * 10))
    log_file="logs/${node_name}.log"
    
    echo -e "${BLUE}🚀 Запуск $node_name (RPC: $rpc_port, P2P: $p2p_port)...${NC}"
    
    # Запуск узла с env переменными
    (cd "$node_dir" && \
     VOLNIX_HOME=".volnix" \
     VOLNIX_RPC_PORT=$rpc_port \
     VOLNIX_P2P_PORT=$p2p_port \
     "$BINARY" start > "../../$log_file" 2>&1 &)
    
    PID=$!
    PIDS[$i]=$PID
    
    echo "  ✅ PID: $PID"
    echo ""
    
    sleep 3
done

# Сохранение PIDs
echo "${PIDS[@]}" > "$TESTNET_DIR/pids.txt"

echo -e "${GREEN}✅ Все узлы запущены!${NC}"
echo ""
echo "PIDs: ${PIDS[@]}"
echo ""

# Ожидание запуска
echo -e "${YELLOW}⏳ Ожидание запуска (15 секунд)...${NC}"
sleep 15

# Проверка статуса
echo ""
echo -e "${GREEN}🔍 Проверка статуса узлов...${NC}"
echo ""

RUNNING=0

for i in 0 1 2; do
    rpc_port=$((26657 + i * 10))
    pid=${PIDS[$i]}
    
    echo -e "${BLUE}Node $i (http://localhost:$rpc_port):${NC}"
    
    # Проверка процесса
    if ! ps -p $pid > /dev/null 2>&1; then
        echo -e "  ${YELLOW}⚠️  Процесс завершился${NC}"
        echo "  Проверьте лог: logs/node$i.log"
        continue
    fi
    
    # Проверка RPC
    if curl -s "http://localhost:$rpc_port/status" > /dev/null 2>&1; then
        HEIGHT=$(curl -s "http://localhost:$rpc_port/status" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null || echo "0")
        PEERS=$(curl -s "http://localhost:$rpc_port/net_info" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('n_peers',0))" 2>/dev/null || echo "0")
        
        echo -e "  ${GREEN}✅ Работает${NC}"
        echo "     Блок: $HEIGHT"
        echo "     Peers: $PEERS"
        RUNNING=$((RUNNING+1))
    else
        echo -e "  ${YELLOW}⏳ RPC не доступен${NC}"
    fi
    echo ""
done

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}📊 Статус сети${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Запущено узлов: $RUNNING/3"
echo ""

if [ $RUNNING -eq 3 ]; then
    echo -e "${GREEN}✅ ВСЕ УЗЛЫ РАБОТАЮТ!${NC}"
    echo ""
    
    # Проверка P2P соединений
    echo "🔗 P2P соединения:"
    for i in 0 1 2; do
        rpc_port=$((26657 + i * 10))
        peers=$(curl -s "http://localhost:$rpc_port/net_info" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('n_peers',0))" 2>/dev/null || echo "0")
        echo "  Node $i: $peers peers"
    done
    echo ""
    
    # Проверка синхронизации блоков
    echo "📊 Высота блоков:"
    for i in 0 1 2; do
        rpc_port=$((26657 + i * 10))
        height=$(curl -s "http://localhost:$rpc_port/status" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null || echo "0")
        echo "  Node $i: $height"
    done
fi

echo ""
echo -e "${BLUE}🌐 Endpoints:${NC}"
for i in 0 1 2; do
    rpc_port=$((26657 + i * 10))
    echo "  Node $i: http://localhost:$rpc_port"
done

echo ""
echo -e "${BLUE}📋 Логи:${NC}"
for i in 0 1 2; do
    echo "  tail -f logs/node$i.log"
done

echo ""
echo -e "${BLUE}🛑 Остановка:${NC}"
echo "  kill ${PIDS[@]}"
echo "  или: pkill -f volnixd-standalone"
echo ""


