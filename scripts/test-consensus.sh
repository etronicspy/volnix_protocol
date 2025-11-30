#!/bin/bash

# Комплексный скрипт проверки консенсуса PoVB
# Проверяет: создание блоков, аукционы, сжигание, валидаторы

RPC_ENDPOINT="http://localhost:26657"
LCD_ENDPOINT="http://localhost:1317"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Счетчики
PASSED=0
FAILED=0
WARNINGS=0

# Функция для вывода результатов
print_result() {
    local status=$1
    local message=$2
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ PASS${NC}: $message"
        ((PASSED++))
    elif [ "$status" = "FAIL" ]; then
        echo -e "${RED}❌ FAIL${NC}: $message"
        ((FAILED++))
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠️  WARN${NC}: $message"
        ((WARNINGS++))
    else
        echo -e "${BLUE}ℹ️  INFO${NC}: $message"
    fi
}

# Функция для проверки доступности узла
check_node_availability() {
    echo -e "\n${BLUE}=== Проверка доступности узла ===${NC}\n"
    
    STATUS=$(curl -s "$RPC_ENDPOINT/status" 2>/dev/null)
    if [ $? -ne 0 ] || [ -z "$STATUS" ]; then
        print_result "FAIL" "Узел недоступен на $RPC_ENDPOINT"
        return 1
    fi
    
    HEIGHT=$(echo "$STATUS" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    NETWORK=$(echo "$STATUS" | jq -r '.result.node_info.network' 2>/dev/null)
    SYNCING=$(echo "$STATUS" | jq -r '.result.sync_info.catching_up' 2>/dev/null)
    
    if [ -n "$HEIGHT" ] && [ "$HEIGHT" != "null" ]; then
        print_result "PASS" "Узел доступен: сеть=$NETWORK, высота=$HEIGHT, синхронизация=$SYNCING"
        return 0
    else
        print_result "FAIL" "Не удалось получить статус узла"
        return 1
    fi
}

# Функция для проверки создания блоков
check_block_creation() {
    echo -e "\n${BLUE}=== Проверка создания блоков ===${NC}\n"
    
    # Получаем текущую высоту
    CURRENT_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    
    if [ -z "$CURRENT_HEIGHT" ] || [ "$CURRENT_HEIGHT" = "null" ]; then
        print_result "FAIL" "Не удалось получить текущую высоту блока"
        return 1
    fi
    
    print_result "INFO" "Текущая высота блока: $CURRENT_HEIGHT"
    
    # Проверяем несколько последних блоков
    BLOCKS_TO_CHECK=5
    START_HEIGHT=$((CURRENT_HEIGHT - BLOCKS_TO_CHECK + 1))
    
    VALID_BLOCKS=0
    for height in $(seq $START_HEIGHT $CURRENT_HEIGHT); do
        BLOCK=$(curl -s "$RPC_ENDPOINT/block?height=$height" 2>/dev/null)
        if [ $? -eq 0 ] && [ -n "$BLOCK" ]; then
            BLOCK_HEIGHT=$(echo "$BLOCK" | jq -r '.result.block.header.height' 2>/dev/null)
            if [ "$BLOCK_HEIGHT" = "$height" ]; then
                ((VALID_BLOCKS++))
            fi
        fi
    done
    
    if [ $VALID_BLOCKS -eq $BLOCKS_TO_CHECK ]; then
        print_result "PASS" "Все проверенные блоки ($BLOCKS_TO_CHECK) существуют и корректны"
    else
        print_result "WARN" "Найдено $VALID_BLOCKS из $BLOCKS_TO_CHECK блоков"
    fi
    
    # Проверяем, что блоки создаются последовательно
    HEIGHT1=$(curl -s "$RPC_ENDPOINT/block?height=$((CURRENT_HEIGHT-1))" | jq -r '.result.block.header.height' 2>/dev/null)
    HEIGHT2=$(curl -s "$RPC_ENDPOINT/block?height=$CURRENT_HEIGHT" | jq -r '.result.block.header.height' 2>/dev/null)
    
    if [ "$HEIGHT1" = "$((CURRENT_HEIGHT-1))" ] && [ "$HEIGHT2" = "$CURRENT_HEIGHT" ]; then
        print_result "PASS" "Блоки создаются последовательно"
    else
        print_result "WARN" "Обнаружены пропуски в последовательности блоков"
    fi
}

# Функция для проверки BeginBlocker
check_begin_blocker() {
    echo -e "\n${BLUE}=== Проверка BeginBlocker ===${NC}\n"
    
    CURRENT_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    
    # Проверяем события BeginBlock в последнем блоке
    BLOCK_RESULT=$(curl -s "$RPC_ENDPOINT/block_results?height=$CURRENT_HEIGHT" 2>/dev/null)
    
    if [ $? -eq 0 ] && [ -n "$BLOCK_RESULT" ]; then
        BEGIN_EVENTS=$(echo "$BLOCK_RESULT" | jq -r '.result.begin_block_events[]?.type' 2>/dev/null | wc -l)
        if [ "$BEGIN_EVENTS" -gt 0 ]; then
            print_result "PASS" "BeginBlock события найдены ($BEGIN_EVENTS событий)"
        else
            print_result "WARN" "BeginBlock события не найдены (может быть нормально)"
        fi
    else
        print_result "WARN" "Не удалось получить результаты блока"
    fi
}

# Функция для проверки EndBlocker
check_end_blocker() {
    echo -e "\n${BLUE}=== Проверка EndBlocker ===${NC}\n"
    
    CURRENT_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    
    # Проверяем события EndBlock в последнем блоке
    BLOCK_RESULT=$(curl -s "$RPC_ENDPOINT/block_results?height=$CURRENT_HEIGHT" 2>/dev/null)
    
    if [ $? -eq 0 ] && [ -n "$BLOCK_RESULT" ]; then
        END_EVENTS=$(echo "$BLOCK_RESULT" | jq -r '.result.end_block_events[]?.type' 2>/dev/null | wc -l)
        if [ "$END_EVENTS" -gt 0 ]; then
            print_result "PASS" "EndBlock события найдены ($END_EVENTS событий)"
            
            # Проверяем наличие событий консенсуса
            CONSENSUS_EVENTS=$(echo "$BLOCK_RESULT" | jq -r '.result.end_block_events[]?.type' 2>/dev/null | grep -i "consensus\|auction\|burn" | wc -l)
            if [ "$CONSENSUS_EVENTS" -gt 0 ]; then
                print_result "PASS" "События консенсуса найдены ($CONSENSUS_EVENTS событий)"
            else
                print_result "WARN" "События консенсуса не найдены в EndBlock"
            fi
        else
            print_result "WARN" "EndBlock события не найдены"
        fi
    else
        print_result "WARN" "Не удалось получить результаты блока"
    fi
}

# Функция для проверки валидаторов
check_validators() {
    echo -e "\n${BLUE}=== Проверка валидаторов ===${NC}\n"
    
    # Пробуем получить валидаторов через REST API
    VALIDATORS=$(curl -s "$LCD_ENDPOINT/volnix/consensus/v1/validators" 2>/dev/null)
    
    if [ $? -eq 0 ] && [ -n "$VALIDATORS" ] && [ "$VALIDATORS" != "null" ]; then
        VALIDATOR_COUNT=$(echo "$VALIDATORS" | jq -r '.validators | length' 2>/dev/null)
        if [ "$VALIDATOR_COUNT" -gt 0 ]; then
            print_result "PASS" "Найдено валидаторов: $VALIDATOR_COUNT"
            
            # Проверяем активных валидаторов
            ACTIVE_COUNT=$(echo "$VALIDATORS" | jq -r '.validators[] | select(.status == "VALIDATOR_STATUS_ACTIVE")' 2>/dev/null | jq -s 'length')
            if [ "$ACTIVE_COUNT" -gt 0 ]; then
                print_result "PASS" "Активных валидаторов: $ACTIVE_COUNT"
            else
                print_result "WARN" "Активных валидаторов не найдено"
            fi
        else
            print_result "WARN" "Валидаторы не найдены"
        fi
    else
        print_result "WARN" "REST API недоступен, проверка валидаторов пропущена"
    fi
}

# Функция для проверки сжигания
check_burn_events() {
    echo -e "\n${BLUE}=== Проверка сжигания ANT ===${NC}\n"
    
    CURRENT_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    
    # Проверяем последние 10 блоков на события сжигания
    BLOCKS_TO_CHECK=10
    START_HEIGHT=$((CURRENT_HEIGHT - BLOCKS_TO_CHECK + 1))
    
    BURN_FOUND=0
    for height in $(seq $START_HEIGHT $CURRENT_HEIGHT); do
        BLOCK_RESULT=$(curl -s "$RPC_ENDPOINT/block_results?height=$height" 2>/dev/null)
        if [ $? -eq 0 ]; then
            BURN_EVENTS=$(echo "$BLOCK_RESULT" | jq -r '.result.end_block_events[]? | select(.type | contains("burn"))' 2>/dev/null | wc -l)
            if [ "$BURN_EVENTS" -gt 0 ]; then
                ((BURN_FOUND++))
                print_result "INFO" "Событие сжигания найдено в блоке $height"
            fi
        fi
    done
    
    if [ $BURN_FOUND -gt 0 ]; then
        print_result "PASS" "Найдено событий сжигания: $BURN_FOUND в последних $BLOCKS_TO_CHECK блоках"
    else
        print_result "WARN" "События сжигания не найдены (нормально, если нет активных аукционов)"
    fi
}

# Функция для проверки аукционов
check_auctions() {
    echo -e "\n${BLUE}=== Проверка слепых аукционов ===${NC}\n"
    
    # Аукционы хранятся в состоянии, доступ через query
    # Пока проверяем только наличие событий аукциона
    
    CURRENT_HEIGHT=$(curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height' 2>/dev/null)
    
    # Проверяем последние 5 блоков на события аукциона
    BLOCKS_TO_CHECK=5
    START_HEIGHT=$((CURRENT_HEIGHT - BLOCKS_TO_CHECK + 1))
    
    AUCTION_EVENTS=0
    for height in $(seq $START_HEIGHT $CURRENT_HEIGHT); do
        BLOCK_RESULT=$(curl -s "$RPC_ENDPOINT/block_results?height=$height" 2>/dev/null)
        if [ $? -eq 0 ]; then
            EVENTS=$(echo "$BLOCK_RESULT" | jq -r '.result.end_block_events[]? | select(.type | contains("auction"))' 2>/dev/null | wc -l)
            if [ "$EVENTS" -gt 0 ]; then
                ((AUCTION_EVENTS++))
            fi
        fi
    done
    
    if [ $AUCTION_EVENTS -gt 0 ]; then
        print_result "PASS" "События аукциона найдены в $AUCTION_EVENTS блоках"
    else
        print_result "WARN" "События аукциона не найдены (нормально, если нет активных участников)"
    fi
}

# Функция для вывода итогового отчета
print_summary() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}=== Итоговый отчет проверки ===${NC}"
    echo -e "${BLUE}========================================${NC}\n"
    
    echo -e "${GREEN}✅ Успешно: $PASSED${NC}"
    echo -e "${YELLOW}⚠️  Предупреждения: $WARNINGS${NC}"
    echo -e "${RED}❌ Ошибки: $FAILED${NC}"
    echo ""
    
    TOTAL=$((PASSED + WARNINGS + FAILED))
    if [ $TOTAL -gt 0 ]; then
        SUCCESS_RATE=$((PASSED * 100 / TOTAL))
        echo -e "Процент успеха: ${GREEN}$SUCCESS_RATE%${NC}"
    fi
    
    if [ $FAILED -eq 0 ]; then
        echo -e "\n${GREEN}✅ Все критические проверки пройдены!${NC}"
    else
        echo -e "\n${RED}❌ Обнаружены критические ошибки${NC}"
    fi
}

# Главная функция
main() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║   Комплексная проверка консенсуса PoVB                  ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # Проверяем доступность узла
    if ! check_node_availability; then
        print_result "FAIL" "Узел недоступен. Завершение проверки."
        exit 1
    fi
    
    # Выполняем проверки
    check_block_creation
    check_begin_blocker
    check_end_blocker
    check_validators
    check_burn_events
    check_auctions
    
    # Выводим итоговый отчет
    print_summary
}

# Запуск
main "$@"


