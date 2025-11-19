#!/bin/bash

# Скрипт тестирования мультинод сети с параметризованными портами

set -e

BINARY="./build/volnixd-standalone"
TEST_DIR="multinode-test"

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== Тестирование мультинод сети ===${NC}"
echo ""

# Проверка бинарника
if [ ! -f "$BINARY" ]; then
    echo "Бинарник не найден. Собираю..."
    make build-standalone
fi

# Остановка существующих узлов
echo -e "${YELLOW}🛑 Остановка существующих узлов...${NC}"
pkill -f "volnixd-standalone" || true
sleep 2

# Очистка
if [ -d "$TEST_DIR" ]; then
    rm -rf "$TEST_DIR"
fi

mkdir -p "$TEST_DIR"

# Создание и инициализация узлов
echo -e "${BLUE}📦 Создание и инициализация узлов...${NC}"
for i in 0 1 2; do
    NODE_DIR="$TEST_DIR/node$i"
    mkdir -p "$NODE_DIR"
    
    # Инициализация узла
    (cd "$NODE_DIR" && VOLNIX_HOME=".volnix" "$BINARY" init "node$i" > /dev/null 2>&1)
    
    echo "   ✅ node$i инициализирован"
done

echo ""
echo -e "${GREEN}🚀 Запуск узлов с разными портами...${NC}"
echo ""

# Массив для PIDs
declare -a PIDS
declare -a RPC_PORTS

# Запуск узлов
for i in 0 1 2; do
    NODE_DIR="$TEST_DIR/node$i"
    RPC_PORT=$((26657 + i * 100))
    P2P_PORT=$((26656 + i * 100))
    LOG_FILE="$TEST_DIR/node$i.log"
    
    echo -e "${BLUE}🚀 Запуск node$i (RPC: $RPC_PORT, P2P: $P2P_PORT)...${NC}"
    
    # Запуск с env переменными (используем абсолютный путь)
    BINARY_ABS="$(cd "$(dirname "$BINARY")" && pwd)/$(basename "$BINARY")"
    (cd "$NODE_DIR" && \
     VOLNIX_HOME=".volnix" \
     VOLNIX_RPC_PORT=$RPC_PORT \
     VOLNIX_P2P_PORT=$P2P_PORT \
     "$BINARY_ABS" start > "../node$i.log" 2>&1 &)
    
    PID=$!
    PIDS[$i]=$PID
    RPC_PORTS[$i]=$RPC_PORT
    
    echo "   PID: $PID"
    sleep 3
done

echo ""
echo -e "${GREEN}✅ Все узлы запущены!${NC}"
echo ""
echo "PIDs: ${PIDS[@]}"
echo ""

# Сохранение PIDs
echo "${PIDS[@]}" > "$TEST_DIR/pids.txt"

# Ожидание запуска
echo -e "${YELLOW}⏳ Ожидание запуска узлов (10 секунд)...${NC}"
sleep 10

# Проверка статуса
echo ""
echo -e "${GREEN}🔍 Проверка статуса узлов...${NC}"
echo ""

RUNNING=0

for i in 0 1 2; do
    RPC_PORT=${RPC_PORTS[$i]}
    PID=${PIDS[$i]}
    
    echo -e "${BLUE}Node $i (http://localhost:$RPC_PORT):${NC}"
    
    # Проверка процесса
    if ! ps -p $PID > /dev/null 2>&1; then
        echo -e "   ${YELLOW}❌ Процесс не запущен${NC}"
        echo "   Проверьте лог: $TEST_DIR/node$i.log"
        continue
    fi
    
    # Проверка RPC
    if curl -s "http://localhost:$RPC_PORT/status" > /dev/null 2>&1; then
        HEIGHT=$(curl -s "http://localhost:$RPC_PORT/status" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null || echo "0")
        echo -e "   ${GREEN}✅ Работает${NC}"
        echo "      Высота блока: $HEIGHT"
        RUNNING=$((RUNNING+1))
    else
        echo -e "   ${YELLOW}⏳ RPC еще не доступен${NC}"
    fi
    echo ""
done

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}📊 Результаты теста${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Запущено узлов: $RUNNING/3"
echo ""

if [ $RUNNING -eq 3 ]; then
    echo -e "${GREEN}✅ ВСЕ УЗЛЫ РАБОТАЮТ!${NC}"
    echo ""
    echo "🎉 Мультинод сеть успешно запущена!"
    echo ""
    echo "📍 RPC Endpoints:"
    for i in 0 1 2; do
        echo "   Node $i: http://localhost:${RPC_PORTS[$i]}"
    done
else
    echo -e "${YELLOW}⚠️  Не все узлы запустились${NC}"
    echo ""
    echo "Проверьте логи:"
    for i in 0 1 2; do
        echo "   tail -f $TEST_DIR/node$i.log"
    done
fi

echo ""
echo "🛑 Остановка: kill ${PIDS[@]}"
echo "   или: pkill -f volnixd-standalone"
echo ""

