#!/bin/bash

# Скрипт для тестирования слепого аукциона
# Тестирует полный цикл: commit -> reveal -> winner selection -> burn

RPC_ENDPOINT="http://localhost:26657"
LCD_ENDPOINT="http://localhost:1317"
CHAIN_ID="volnix-testnet"

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Функция для генерации commit hash
# commit_hash = SHA256(nonce:bid_amount)
generate_commit_hash() {
    local nonce=$1
    local bid_amount=$2
    echo -n "${nonce}:${bid_amount}" | sha256sum | cut -d' ' -f1
}

# Функция для получения текущей высоты блока
get_current_height() {
    curl -s "$RPC_ENDPOINT/status" | jq -r '.result.sync_info.latest_block_height'
}

# Функция для проверки фазы аукциона
check_auction_phase() {
    local height=$1
    # Пока не можем проверить через query, проверяем через логику:
    # Аукцион создается в EndBlocker для следующего блока
    # Фаза COMMIT для блока N создается в EndBlocker блока N-1
    # Фаза REVEAL после первого commit в EndBlocker
    echo "COMMIT" # Упрощенная версия
}

# Функция для отправки CommitBid транзакции
send_commit_bid() {
    local validator=$1
    local commit_hash=$2
    local height=$3
    local key_name=$4
    
    echo -e "${BLUE}Отправка CommitBid транзакции...${NC}"
    echo "  Валидатор: $validator"
    echo "  Commit Hash: $commit_hash"
    echo "  Высота блока: $height"
    
    # Формируем JSON для транзакции
    # Используем volnixd tx для отправки
    # Формат: volnixd tx consensus commit-bid [validator] [commit-hash] [height] --from [key-name]
    
    # Пока используем упрощенный подход через gRPC/REST
    # В реальности нужно использовать volnixd tx или CosmJS
    
    echo -e "${YELLOW}⚠️  Для отправки транзакции нужен volnixd CLI или CosmJS клиент${NC}"
    echo "Пример команды:"
    echo "  volnixd tx consensus commit-bid $validator $commit_hash $height --from $key_name --chain-id $CHAIN_ID"
    
    return 0
}

# Функция для отправки RevealBid транзакции
send_reveal_bid() {
    local validator=$1
    local nonce=$2
    local bid_amount=$3
    local height=$4
    local key_name=$5
    
    echo -e "${BLUE}Отправка RevealBid транзакции...${NC}"
    echo "  Валидатор: $validator"
    echo "  Nonce: $nonce"
    echo "  Bid Amount: $bid_amount"
    echo "  Высота блока: $height"
    
    echo -e "${YELLOW}⚠️  Для отправки транзакции нужен volnixd CLI или CosmJS клиент${NC}"
    echo "Пример команды:"
    echo "  volnixd tx consensus reveal-bid $validator $nonce $bid_amount $height --from $key_name --chain-id $CHAIN_ID"
    
    return 0
}

# Функция для проверки событий сжигания
check_burn_events() {
    local height=$1
    
    echo -e "\n${BLUE}Проверка событий сжигания в блоке $height...${NC}"
    
    BLOCK_RESULT=$(curl -s "$RPC_ENDPOINT/block_results?height=$height" 2>/dev/null)
    if [ $? -eq 0 ] && [ -n "$BLOCK_RESULT" ]; then
        BURN_EVENTS=$(echo "$BLOCK_RESULT" | jq -r '.result.end_block_events[]? | select(.type | contains("burn"))' 2>/dev/null)
        
        if [ -n "$BURN_EVENTS" ] && [ "$BURN_EVENTS" != "null" ]; then
            echo -e "${GREEN}✅ События сжигания найдены:${NC}"
            echo "$BURN_EVENTS" | jq '.'
            return 0
        else
            echo -e "${YELLOW}⚠️  События сжигания не найдены${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ Не удалось получить результаты блока${NC}"
        return 1
    fi
}

# Функция для тестирования полного цикла аукциона
test_auction_cycle() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════════╗"
    echo "║   Тестирование слепого аукциона                         ║"
    echo "╚══════════════════════════════════════════════════════════╝"
    echo -e "${NC}\n"
    
    # Параметры теста (в реальности должны быть получены из конфигурации)
    VALIDATOR1="cosmos1validator1"  # Заменить на реальный адрес
    VALIDATOR2="cosmos1validator2"  # Заменить на реальный адрес
    KEY_NAME1="validator1"          # Заменить на реальное имя ключа
    KEY_NAME2="validator2"         # Заменить на реальное имя ключа
    
    # Получаем текущую высоту
    CURRENT_HEIGHT=$(get_current_height)
    TARGET_HEIGHT=$((CURRENT_HEIGHT + 2))  # Аукцион для блока через 2
    
    echo -e "${BLUE}Текущая высота блока: $CURRENT_HEIGHT${NC}"
    echo -e "${BLUE}Целевая высота для аукциона: $TARGET_HEIGHT${NC}\n"
    
    # Генерируем commit данные для валидатора 1
    NONCE1=$(openssl rand -hex 16)
    BID_AMOUNT1="1000000"  # 1M ANT
    COMMIT_HASH1=$(generate_commit_hash "$NONCE1" "$BID_AMOUNT1")
    
    echo -e "${GREEN}=== Валидатор 1 ===${NC}"
    echo "Nonce: $NONCE1"
    echo "Bid Amount: $BID_AMOUNT1 ANT"
    echo "Commit Hash: $COMMIT_HASH1"
    echo ""
    
    # Генерируем commit данные для валидатора 2
    NONCE2=$(openssl rand -hex 16)
    BID_AMOUNT2="2000000"  # 2M ANT
    COMMIT_HASH2=$(generate_commit_hash "$NONCE2" "$BID_AMOUNT2")
    
    echo -e "${GREEN}=== Валидатор 2 ===${NC}"
    echo "Nonce: $NONCE2"
    echo "Bid Amount: $BID_AMOUNT2 ANT"
    echo "Commit Hash: $COMMIT_HASH2"
    echo ""
    
    # Фаза 1: COMMIT
    echo -e "${BLUE}=== Фаза 1: COMMIT ===${NC}\n"
    
    echo "Отправка CommitBid для валидатора 1..."
    send_commit_bid "$VALIDATOR1" "$COMMIT_HASH1" "$TARGET_HEIGHT" "$KEY_NAME1"
    echo ""
    
    echo "Отправка CommitBid для валидатора 2..."
    send_commit_bid "$VALIDATOR2" "$COMMIT_HASH2" "$TARGET_HEIGHT" "$KEY_NAME2"
    echo ""
    
    echo -e "${YELLOW}⚠️  Ожидание перехода в фазу REVEAL...${NC}"
    echo "Переход происходит автоматически в EndBlocker при наличии commits"
    echo ""
    
    # Фаза 2: REVEAL
    echo -e "${BLUE}=== Фаза 2: REVEAL ===${NC}\n"
    
    echo "Отправка RevealBid для валидатора 1..."
    send_reveal_bid "$VALIDATOR1" "$NONCE1" "$BID_AMOUNT1" "$TARGET_HEIGHT" "$KEY_NAME1"
    echo ""
    
    echo "Отправка RevealBid для валидатора 2..."
    send_reveal_bid "$VALIDATOR2" "$NONCE2" "$BID_AMOUNT2" "$TARGET_HEIGHT" "$KEY_NAME2"
    echo ""
    
    # Фаза 3: Проверка результатов
    echo -e "${BLUE}=== Фаза 3: Проверка результатов ===${NC}\n"
    
    echo "Ожидание выбора победителя и сжигания..."
    sleep 5
    
    # Проверяем события сжигания в целевом блоке
    check_burn_events "$TARGET_HEIGHT"
    
    echo -e "\n${GREEN}=== Итоги теста ===${NC}"
    echo "1. Commit фаза: подготовлены commit hash для обоих валидаторов"
    echo "2. Reveal фаза: подготовлены reveal данные"
    echo "3. Для полного теста необходимо:"
    echo "   - Настроить валидаторов с реальными адресами"
    echo "   - Отправить транзакции через volnixd CLI или CosmJS"
    echo "   - Проверить выбор победителя и сжигание ANT"
}

# Функция для проверки логики HashCommit
test_hash_commit() {
    echo -e "${BLUE}=== Тест HashCommit логики ===${NC}\n"
    
    NONCE="test_nonce_123"
    BID_AMOUNT="1000000"
    
    COMMIT_HASH=$(generate_commit_hash "$NONCE" "$BID_AMOUNT")
    
    echo "Nonce: $NONCE"
    echo "Bid Amount: $BID_AMOUNT"
    echo "Commit Hash: $COMMIT_HASH"
    echo "Длина hash: ${#COMMIT_HASH} символов"
    
    if [ ${#COMMIT_HASH} -eq 64 ]; then
        echo -e "${GREEN}✅ Commit hash имеет правильную длину (64 hex символа)${NC}"
    else
        echo -e "${RED}❌ Commit hash имеет неправильную длину${NC}"
    fi
    
    # Проверяем, что тот же nonce и bid_amount дают тот же hash
    COMMIT_HASH2=$(generate_commit_hash "$NONCE" "$BID_AMOUNT")
    if [ "$COMMIT_HASH" = "$COMMIT_HASH2" ]; then
        echo -e "${GREEN}✅ Hash детерминирован (одинаковые входные данные дают одинаковый hash)${NC}"
    else
        echo -e "${RED}❌ Hash не детерминирован${NC}"
    fi
    
    # Проверяем, что разные данные дают разные hash
    COMMIT_HASH3=$(generate_commit_hash "different_nonce" "$BID_AMOUNT")
    if [ "$COMMIT_HASH" != "$COMMIT_HASH3" ]; then
        echo -e "${GREEN}✅ Разные входные данные дают разные hash${NC}"
    else
        echo -e "${RED}❌ Hash коллизия обнаружена${NC}"
    fi
}

# Главная функция
main() {
    case "${1:-all}" in
        hash)
            test_hash_commit
            ;;
        auction)
            test_auction_cycle
            ;;
        all)
            test_hash_commit
            echo ""
            test_auction_cycle
            ;;
        *)
            echo "Использование: $0 [hash|auction|all]"
            echo ""
            echo "Команды:"
            echo "  hash     - Тест логики HashCommit"
            echo "  auction  - Тест полного цикла аукциона"
            echo "  all      - Выполнить все тесты (по умолчанию)"
            exit 1
            ;;
    esac
}

# Запуск
main "$@"


