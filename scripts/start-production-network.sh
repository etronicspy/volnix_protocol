#!/bin/bash

# Скрипт запуска продакшн сети Volnix Protocol

set -e

BINARY="./build/volnixd"
NETWORK_DIR="mainnet"
LOG_DIR="$NETWORK_DIR/logs"

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== Запуск продакшн сети Volnix Protocol ===${NC}"
echo ""

# Проверка инициализации
if [ ! -d "$NETWORK_DIR" ]; then
    echo -e "${RED}❌ Сеть не инициализирована!${NC}"
    echo "Запустите: ./scripts/init-production-network.sh"
    exit 1
fi

# Создание директории для логов
mkdir -p "$LOG_DIR"

# Остановка существующих узлов
echo -e "${YELLOW}🛑 Остановка существующих узлов...${NC}"
pkill -f "volnixd start" || true
sleep 2

# Подсчет узлов
NUM_VALIDATORS=$(ls -d $NETWORK_DIR/validator-* 2>/dev/null | wc -l | tr -d ' ')

if [ "$NUM_VALIDATORS" -eq 0 ]; then
    echo -e "${RED}❌ Узлы не найдены в $NETWORK_DIR${NC}"
    exit 1
fi

echo "Найдено узлов: $NUM_VALIDATORS"
echo ""

# Массивы для хранения PIDs и портов
declare -a PIDS
declare -a RPC_PORTS
declare -a NODE_NAMES

# Запуск узлов
echo -e "${GREEN}🚀 Запуск узлов...${NC}"
echo ""

for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    NODE_NAME="validator-$i"
    NODE_DIR="$NETWORK_DIR/$NODE_NAME"
    LOG_FILE="$LOG_DIR/${NODE_NAME}.log"
    RPC_PORT=$((26657 + i * 100))
    
    echo -e "${BLUE}🚀 Запуск $NODE_NAME (RPC: $RPC_PORT)...${NC}"
    
    # Запуск узла в фоне
    $BINARY start --home "$NODE_DIR" > "$LOG_FILE" 2>&1 &
    PID=$!
    
    PIDS[$i]=$PID
    RPC_PORTS[$i]=$RPC_PORT
    NODE_NAMES[$i]=$NODE_NAME
    
    echo "   ✅ $NODE_NAME запущен (PID: $PID)"
    echo "      Лог: $LOG_FILE"
    
    # Задержка между запусками
    sleep 3
done

# Сохранение PIDs
PIDS_FILE="$NETWORK_DIR/pids.txt"
echo "${PIDS[@]}" > "$PIDS_FILE"
echo ""
echo -e "${GREEN}✅ Все узлы запущены!${NC}"
echo "📝 PIDs сохранены в $PIDS_FILE"
echo ""

# Ожидание запуска
echo -e "${YELLOW}⏳ Ожидание запуска узлов (15 секунд)...${NC}"
sleep 15

# Проверка статуса
echo ""
echo -e "${GREEN}🔍 Проверка статуса узлов...${NC}"
echo ""

RUNNING_COUNT=0

for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    NODE_NAME=${NODE_NAMES[$i]}
    RPC_PORT=${RPC_PORTS[$i]}
    PID=${PIDS[$i]}
    
    echo -e "${BLUE}Проверка $NODE_NAME (http://localhost:$RPC_PORT)...${NC}"
    
    # Проверка процесса
    if ! ps -p $PID > /dev/null 2>&1; then
        echo -e "   ${RED}❌ Процесс не запущен${NC}"
        continue
    fi
    
    # Проверка RPC
    if curl -s "http://localhost:$RPC_PORT/status" > /dev/null 2>&1; then
        STATUS=$(curl -s "http://localhost:$RPC_PORT/status" 2>/dev/null)
        HEIGHT=$(echo "$STATUS" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null || echo "0")
        CATCHING_UP=$(echo "$STATUS" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('catching_up','unknown'))" 2>/dev/null || echo "unknown")
        
        echo -e "   ${GREEN}✅ Узел работает${NC}"
        echo "      Высота блока: $HEIGHT"
        echo "      Синхронизация: $CATCHING_UP"
        RUNNING_COUNT=$((RUNNING_COUNT+1))
    else
        echo -e "   ${YELLOW}⏳ RPC еще не доступен${NC}"
    fi
    echo ""
done

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}📊 Статус сети${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Запущено узлов: $RUNNING_COUNT/$NUM_VALIDATORS"
echo ""
echo -e "${BLUE}🌐 Endpoint'ы:${NC}"
for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    RPC_PORT=$((26657 + i * 100))
    API_PORT=$((1317 + i * 10))
    echo "   ${NODE_NAMES[$i]}: RPC http://localhost:$RPC_PORT, API http://localhost:$API_PORT"
done
echo ""
echo -e "${BLUE}📋 Логи:${NC}"
echo "   tail -f $LOG_DIR/validator-*.log"
echo ""
echo -e "${BLUE}🛑 Остановка:${NC}"
echo "   kill ${PIDS[@]}"
echo "   или: pkill -f 'volnixd start'"
echo ""
echo -e "${BLUE}📈 Мониторинг:${NC}"
echo "   watch -n 2 'curl -s http://localhost:26657/status | jq .result.sync_info'"
echo ""

