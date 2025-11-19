#!/bin/bash

# Скрипт остановки продакшн сети Volnix Protocol

NETWORK_DIR="mainnet"
PIDS_FILE="$NETWORK_DIR/pids.txt"

echo "🛑 Остановка продакшн сети..."
echo ""

# Остановка по PID файлу
if [ -f "$PIDS_FILE" ]; then
    echo "Остановка узлов по PID файлу..."
    PIDS=$(cat "$PIDS_FILE")
    for PID in $PIDS; do
        if ps -p $PID > /dev/null 2>&1; then
            echo "   Остановка процесса $PID..."
            kill $PID 2>/dev/null || true
        fi
    done
    rm -f "$PIDS_FILE"
fi

# Остановка всех процессов volnixd
echo "Остановка всех процессов volnixd..."
pkill -f "volnixd start" || true

sleep 2

# Проверка
REMAINING=$(ps aux | grep "volnixd start" | grep -v grep | wc -l | tr -d ' ')
if [ "$REMAINING" -eq 0 ]; then
    echo "✅ Все узлы остановлены"
else
    echo "⚠️  Осталось процессов: $REMAINING"
    echo "Принудительная остановка..."
    pkill -9 -f "volnixd start" || true
fi

echo ""
echo "✅ Продакшн сеть остановлена"

