#!/bin/bash

# Скрипт для мониторинга транзакций в блокчейне

RPC_ENDPOINT="http://localhost:26657"

echo "🔍 Мониторинг транзакций в блокчейне"
echo "======================================"
echo ""

# Функция для проверки последних блоков
check_recent_blocks() {
    local num_blocks=${1:-5}
    echo "Проверка последних $num_blocks блоков на наличие транзакций..."
    echo ""
    
    CURRENT_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height')
    START_HEIGHT=$((CURRENT_HEIGHT - num_blocks + 1))
    
    FOUND_TXS=0
    
    for height in $(seq $START_HEIGHT $CURRENT_HEIGHT); do
        BLOCK=$(curl -s "$RPC_ENDPOINT/block?height=$height" 2>/dev/null)
        if [ $? -eq 0 ]; then
            TXS=$(echo "$BLOCK" | jq -r '.result.block.data.txs | length')
            if [ "$TXS" -gt 0 ]; then
                FOUND_TXS=$((FOUND_TXS + TXS))
                TIME=$(echo "$BLOCK" | jq -r '.result.block.header.time')
                HASH=$(echo "$BLOCK" | jq -r '.result.block.header.hash')
                echo "✅ Блок $height: $TXS транзакций"
                echo "   Время: $TIME"
                echo "   Хеш: $HASH"
                echo ""
            fi
        fi
    done
    
    if [ $FOUND_TXS -eq 0 ]; then
        echo "Транзакций не найдено в последних $num_blocks блоках"
    else
        echo "Всего найдено транзакций: $FOUND_TXS"
    fi
}

# Функция для проверки конкретной транзакции
check_transaction() {
    local tx_hash=$1
    
    if [ -z "$tx_hash" ]; then
        echo "Использование: $0 check <tx_hash>"
        exit 1
    fi
    
    echo "Проверка транзакции: $tx_hash"
    echo ""
    
    TX=$(curl -s "$RPC_ENDPOINT/tx?hash=0x$tx_hash" 2>/dev/null)
    
    if [ $? -eq 0 ] && [ "$(echo "$TX" | jq -r '.result.tx')" != "null" ]; then
        echo "✅ Транзакция найдена!"
        echo ""
        echo "$TX" | jq -r '
            "Высота блока: " + (.result.height | tostring),
            "Хеш: " + .result.hash,
            "Код результата: " + (.result.tx_result.code | tostring),
            "Gas использовано: " + (.result.tx_result.gas_used | tostring),
            "Лог: " + .result.tx_result.log
        '
    else
        echo "❌ Транзакция не найдена или еще не включена в блок"
    fi
}

# Функция для непрерывного мониторинга
monitor_continuous() {
    echo "Непрерывный мониторинг новых транзакций..."
    echo "Нажмите Ctrl+C для остановки"
    echo ""
    
    LAST_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height')
    
    while true; do
        sleep 5
        CURRENT_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height')
        
        if [ "$CURRENT_HEIGHT" -gt "$LAST_HEIGHT" ]; then
            for height in $(seq $((LAST_HEIGHT + 1)) $CURRENT_HEIGHT); do
                BLOCK=$(curl -s "$RPC_ENDPOINT/block?height=$height" 2>/dev/null)
                TXS=$(echo "$BLOCK" | jq -r '.result.block.data.txs | length')
                
                if [ "$TXS" -gt 0 ]; then
                    TIME=$(echo "$BLOCK" | jq -r '.result.block.header.time')
                    echo "[$(date +%H:%M:%S)] 🎉 Новая транзакция в блоке $height!"
                    echo "   Время: $TIME"
                    echo "   Количество транзакций: $TXS"
                    echo ""
                fi
            done
            LAST_HEIGHT=$CURRENT_HEIGHT
        fi
    done
}

# Основная логика
case "${1:-recent}" in
    recent)
        check_recent_blocks "${2:-10}"
        ;;
    check)
        check_transaction "$2"
        ;;
    monitor|watch)
        monitor_continuous
        ;;
    *)
        echo "Использование: $0 [recent|check|monitor]"
        echo ""
        echo "Команды:"
        echo "  recent [N]  - Проверить последние N блоков (по умолчанию 10)"
        echo "  check <hash> - Проверить конкретную транзакцию по хешу"
        echo "  monitor      - Непрерывный мониторинг новых транзакций"
        exit 1
        ;;
esac

