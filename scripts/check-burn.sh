#!/bin/bash

# Скрипт для проверки сжигания ANT токенов
# Проверяет аукционы, валидаторов и балансы ANT

RPC_ENDPOINT="http://localhost:26657"
LCD_ENDPOINT="http://localhost:1317"

echo "🔥 Проверка сжигания ANT токенов"
echo "=================================="
echo ""

# Функция для получения текущей высоты блока
get_current_height() {
    curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height'
}

# Функция для проверки событий в блоке
check_block_events() {
    local height=$1
    echo "Проверка блока $height на события сжигания..."
    
    BLOCK=$(curl -s "$RPC_ENDPOINT/block?height=$height" 2>/dev/null)
    if [ $? -ne 0 ]; then
        echo "  ❌ Не удалось получить блок $height"
        return
    fi
    
    # Проверяем события в результатах блока (EndBlock events)
    # События сжигания эмитятся в EndBlocker, не в транзакциях
    BLOCK_RESULT=$(curl -s "$RPC_ENDPOINT/block_results?height=$height" 2>/dev/null)
    if [ $? -eq 0 ]; then
        # Проверяем события из EndBlock
        EVENTS=$(echo "$BLOCK_RESULT" | jq -r '.result.end_block_events[]? | select(.type | contains("burn") or contains("consensus.burn")) | "\(.type): \(.attributes[]? | select(.key == "burn_amount" or .key == "validator") | "\(.key)=\(.value)")"' 2>/dev/null)
        
        # Также проверяем события из транзакций
        TX_EVENTS=$(echo "$BLOCK" | jq -r '.result.block.data.txs[]? // empty' | while read tx; do
            if [ -n "$tx" ]; then
                TX_HASH=$(echo "$tx" | base64 -d 2>/dev/null | sha256sum | cut -d' ' -f1)
                TX_RESULT=$(curl -s "$RPC_ENDPOINT/tx?hash=0x$TX_HASH" 2>/dev/null)
                if [ $? -eq 0 ]; then
                    echo "$TX_RESULT" | jq -r '.result.tx_result.events[]? | select(.type | contains("burn") or contains("consensus.burn")) | "\(.type): \(.attributes[]? | select(.key == "burn_amount" or .key == "validator") | "\(.key)=\(.value)")"' 2>/dev/null
                fi
            fi
        done)
        
        EVENTS="$EVENTS $TX_EVENTS"
    fi
    
    if [ -n "$EVENTS" ]; then
        echo "  ✅ Найдены события:"
        echo "$EVENTS" | sort -u | while read event; do
            echo "     - $event"
        done
    else
        echo "  ℹ️  Событий сжигания не найдено"
    fi
}

# Функция для проверки валидаторов через REST API
check_validators() {
    echo "Проверка валидаторов..."
    echo ""
    
    # Пробуем разные порты для REST API
    for port in 1317 9090; do
        VALIDATORS=$(curl -s "$LCD_ENDPOINT/volnix/consensus/v1/validators" 2>/dev/null)
        if [ $? -eq 0 ] && [ -n "$VALIDATORS" ] && [ "$VALIDATORS" != "null" ]; then
            echo "✅ Валидаторы найдены:"
            echo "$VALIDATORS" | jq -r '.validators[]? | "  - \(.validator): ANT=\(.ant_balance), Статус=\(.status), Сожжено=\(.total_burn_amount // "0")"' 2>/dev/null
            return
        fi
    done
    
    echo "  ⚠️  REST API недоступен, проверяем через RPC..."
    echo "  (Для проверки валидаторов нужен запущенный REST сервер)"
}

# Функция для проверки последних блоков на наличие сжигания
check_recent_blocks_for_burn() {
    local num_blocks=${1:-10}
    echo "Проверка последних $num_blocks блоков на сжигание..."
    echo ""
    
    CURRENT_HEIGHT=$(get_current_height)
    START_HEIGHT=$((CURRENT_HEIGHT - num_blocks + 1))
    
    BURN_FOUND=0
    
    for height in $(seq $START_HEIGHT $CURRENT_HEIGHT); do
        check_block_events $height
        echo ""
    done
    
    if [ $BURN_FOUND -eq 0 ]; then
        echo "ℹ️  Сжигание не обнаружено в последних $num_blocks блоках"
        echo ""
        echo "Примечание: Сжигание происходит только когда:"
        echo "  1. Валидаторы участвуют в слепом аукционе (commit/reveal)"
        echo "  2. Выбирается победитель аукциона"
        echo "  3. Победитель имеет достаточный баланс ANT"
    fi
}

# Функция для проверки логов на наличие сообщений о сжигании
check_logs_for_burn() {
    echo "Проверка логов на наличие сообщений о сжигании..."
    echo ""
    
    # Ищем логи в testnet директории
    if [ -d "testnet/node0" ]; then
        LOG_FILES=$(find testnet/node0 -name "*.log" -type f 2>/dev/null | head -3)
        if [ -n "$LOG_FILES" ]; then
            echo "Проверка логов узла..."
            for log_file in $LOG_FILES; do
                if grep -q "ANT burned\|burned from auction" "$log_file" 2>/dev/null; then
                    echo "  ✅ Найдены записи о сжигании в $log_file:"
                    grep "ANT burned\|burned from auction" "$log_file" 2>/dev/null | tail -5
                    echo ""
                fi
            done
        fi
    fi
    
    # Проверяем вывод процесса, если он запущен
    echo "Проверка процессов volnixd..."
    if pgrep -f volnixd > /dev/null; then
        echo "  ✅ Процесс volnixd запущен"
        echo "  (Проверьте логи процесса для детальной информации)"
    else
        echo "  ⚠️  Процесс volnixd не найден"
    fi
}

# Функция для проверки состояния сети
check_network_status() {
    echo "Проверка состояния сети..."
    echo ""
    
    STATUS=$(curl -s "$RPC_ENDPOINT/status" 2>/dev/null)
    if [ $? -ne 0 ]; then
        echo "  ❌ Не удалось подключиться к RPC узлу"
        return 1
    fi
    
    HEIGHT=$(echo "$STATUS" | jq -r '.result.sync_info.latest_block_height')
    NETWORK=$(echo "$STATUS" | jq -r '.result.node_info.network')
    SYNCING=$(echo "$STATUS" | jq -r '.result.sync_info.catching_up')
    
    echo "  ✅ Сеть: $NETWORK"
    echo "  ✅ Высота блока: $HEIGHT"
    echo "  ✅ Синхронизация: $([ "$SYNCING" = "false" ] && echo "Синхронизирован" || echo "Синхронизируется")"
    echo ""
}

# Основная логика
main() {
    check_network_status
    
    if [ $? -ne 0 ]; then
        echo "❌ Узел недоступен. Убедитесь, что volnixd запущен."
        exit 1
    fi
    
    check_validators
    echo ""
    
    check_recent_blocks_for_burn "${1:-10}"
    
    check_logs_for_burn
    echo ""
    
    echo "=================================="
    echo "✅ Проверка завершена"
    echo ""
    echo "Для более детальной проверки:"
    echo "  1. Проверьте логи узла: tail -f testnet/node0/.volnix/logs/*.log"
    echo "  2. Проверьте балансы ANT через REST API"
    echo "  3. Убедитесь, что валидаторы участвуют в аукционах"
}

# Запуск
main "$@"

