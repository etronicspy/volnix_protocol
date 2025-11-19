#!/bin/bash

# Скрипт инициализации продакшн сети Volnix Protocol
# Создает мульти-нод сеть с валидаторами

set -e

# Конфигурация
CHAIN_ID="volnix-mainnet-1"
NUM_VALIDATORS=4
BINARY="./build/volnixd"
NETWORK_DIR="mainnet"
DENOM="uwrt"
GENESIS_TOKENS="1000000000000${DENOM}" # 1,000,000 WRT

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== Инициализация продакшн сети Volnix Protocol ===${NC}"
echo ""
echo "Chain ID: $CHAIN_ID"
echo "Validators: $NUM_VALIDATORS"
echo ""

# Проверка бинарника
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}❌ Бинарник не найден: $BINARY${NC}"
    echo "Собираю бинарник..."
    make build
fi

echo -e "${BLUE}✅ Бинарник готов: $BINARY${NC}"
echo ""

# Очистка старых данных
if [ -d "$NETWORK_DIR" ]; then
    echo -e "${YELLOW}⚠️  Удаляю старые данные...${NC}"
    rm -rf "$NETWORK_DIR"
fi

mkdir -p "$NETWORK_DIR"

# Создание узлов
echo -e "${GREEN}📦 Создание узлов...${NC}"
echo ""

VALIDATORS_INFO=""

for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    NODE_NAME="validator-$i"
    NODE_DIR="$NETWORK_DIR/$NODE_NAME"
    
    echo -e "${BLUE}🔧 Инициализация $NODE_NAME...${NC}"
    
    # Инициализация узла
    $BINARY init "$NODE_NAME" --chain-id "$CHAIN_ID" --home "$NODE_DIR" > /dev/null 2>&1
    
    # Создание ключа валидатора
    echo "password" | $BINARY keys add "$NODE_NAME" --keyring-backend test --home "$NODE_DIR" > "$NODE_DIR/key_info.txt" 2>&1
    
    # Получение адреса
    VALIDATOR_ADDR=$($BINARY keys show "$NODE_NAME" -a --keyring-backend test --home "$NODE_DIR" 2>/dev/null)
    
    # Получение валидаторного ключа
    VALIDATOR_PUBKEY=$($BINARY tendermint show-validator --home "$NODE_DIR" 2>/dev/null)
    
    # Получение node ID
    NODE_ID=$($BINARY tendermint show-node-id --home "$NODE_DIR" 2>/dev/null)
    
    # Настройка портов
    P2P_PORT=$((26656 + i * 100))
    RPC_PORT=$((26657 + i * 100))
    API_PORT=$((1317 + i * 10))
    GRPC_PORT=$((9090 + i * 10))
    
    # Обновление конфигурации
    sed -i '' "s|laddr = \"tcp://127.0.0.1:26657\"|laddr = \"tcp://0.0.0.0:$RPC_PORT\"|g" "$NODE_DIR/config/config.toml" 2>/dev/null || true
    sed -i '' "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:$P2P_PORT\"|g" "$NODE_DIR/config/config.toml" 2>/dev/null || true
    
    # Включение API в app.toml
    sed -i '' 's|enable = false|enable = true|g' "$NODE_DIR/config/app.toml" 2>/dev/null || true
    sed -i '' "s|address = \"tcp://localhost:1317\"|address = \"tcp://0.0.0.0:$API_PORT\"|g" "$NODE_DIR/config/app.toml" 2>/dev/null || true
    
    # Сохранение информации об узле
    VALIDATORS_INFO="${VALIDATORS_INFO}${NODE_ID}@127.0.0.1:${P2P_PORT},"
    
    echo "   ✅ $NODE_NAME"
    echo "      Address: $VALIDATOR_ADDR"
    echo "      Node ID: $NODE_ID"
    echo "      P2P: $P2P_PORT, RPC: $RPC_PORT, API: $API_PORT"
    echo ""
    
    # Сохранение информации
    cat > "$NODE_DIR/node_info.json" <<EOF
{
  "name": "$NODE_NAME",
  "address": "$VALIDATOR_ADDR",
  "node_id": "$NODE_ID",
  "validator_pubkey": $VALIDATOR_PUBKEY,
  "ports": {
    "p2p": $P2P_PORT,
    "rpc": $RPC_PORT,
    "api": $API_PORT,
    "grpc": $GRPC_PORT
  }
}
EOF
done

# Удаление последней запятой
VALIDATORS_INFO=${VALIDATORS_INFO%,}

echo -e "${GREEN}📝 Создание genesis файла...${NC}"

# Использование genesis первого узла как базового
GENESIS_FILE="$NETWORK_DIR/validator-0/config/genesis.json"

# Добавление аккаунтов в genesis
for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    NODE_NAME="validator-$i"
    NODE_DIR="$NETWORK_DIR/$NODE_NAME"
    
    VALIDATOR_ADDR=$($BINARY keys show "$NODE_NAME" -a --keyring-backend test --home "$NODE_DIR" 2>/dev/null)
    
    # Добавление аккаунта в genesis
    $BINARY genesis add-genesis-account "$VALIDATOR_ADDR" "$GENESIS_TOKENS" --home "$NETWORK_DIR/validator-0" --keyring-backend test > /dev/null 2>&1 || true
    
    echo "   ✅ Добавлен аккаунт $NODE_NAME: $VALIDATOR_ADDR"
done

echo ""
echo -e "${GREEN}👥 Создание genesis транзакций валидаторов...${NC}"

# Создание gentx для каждого валидатора
for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    NODE_NAME="validator-$i"
    NODE_DIR="$NETWORK_DIR/$NODE_NAME"
    
    # Копирование genesis файла
    cp "$GENESIS_FILE" "$NODE_DIR/config/genesis.json"
    
    # Создание gentx
    VALIDATOR_ADDR=$($BINARY keys show "$NODE_NAME" -a --keyring-backend test --home "$NODE_DIR" 2>/dev/null)
    
    $BINARY genesis gentx "$NODE_NAME" "100000000000${DENOM}" \
        --chain-id "$CHAIN_ID" \
        --keyring-backend test \
        --home "$NODE_DIR" \
        --moniker "$NODE_NAME" \
        --commission-rate "0.10" \
        --commission-max-rate "0.20" \
        --commission-max-change-rate "0.01" \
        --min-self-delegation "1" > /dev/null 2>&1 || true
    
    echo "   ✅ Создан gentx для $NODE_NAME"
    
    # Копирование gentx в директорию validator-0
    if [ $i -ne 0 ]; then
        cp "$NODE_DIR/config/gentx/"*.json "$NETWORK_DIR/validator-0/config/gentx/" 2>/dev/null || true
    fi
done

echo ""
echo -e "${GREEN}🔗 Сборка финального genesis...${NC}"

# Собрать все gentx в финальный genesis
$BINARY genesis collect-gentxs --home "$NETWORK_DIR/validator-0" > /dev/null 2>&1 || true

# Копирование финального genesis на все узлы
for i in $(seq 1 $((NUM_VALIDATORS-1))); do
    cp "$NETWORK_DIR/validator-0/config/genesis.json" "$NETWORK_DIR/validator-$i/config/genesis.json"
done

echo "   ✅ Genesis файл создан и распространен"
echo ""

# Настройка persistent peers
echo -e "${GREEN}🌐 Настройка P2P соединений...${NC}"

for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    NODE_DIR="$NETWORK_DIR/validator-$i"
    
    # Получить список пиров (все узлы кроме текущего)
    PEERS=""
    for j in $(seq 0 $((NUM_VALIDATORS-1))); do
        if [ $i -ne $j ]; then
            PEER_NODE_DIR="$NETWORK_DIR/validator-$j"
            PEER_ID=$($BINARY tendermint show-node-id --home "$PEER_NODE_DIR" 2>/dev/null)
            PEER_PORT=$((26656 + j * 100))
            PEERS="${PEERS}${PEER_ID}@127.0.0.1:${PEER_PORT},"
        fi
    done
    PEERS=${PEERS%,}
    
    # Обновить конфигурацию
    sed -i '' "s|persistent_peers = \"\"|persistent_peers = \"$PEERS\"|g" "$NODE_DIR/config/config.toml" 2>/dev/null || true
    
    echo "   ✅ validator-$i настроен на соединение с $((NUM_VALIDATORS-1)) пирами"
done

echo ""
echo -e "${GREEN}✅ Инициализация завершена!${NC}"
echo ""
echo -e "${BLUE}📊 Информация о сети:${NC}"
echo "   Chain ID: $CHAIN_ID"
echo "   Validators: $NUM_VALIDATORS"
echo "   Network Directory: $NETWORK_DIR"
echo ""
echo -e "${BLUE}🚀 Для запуска сети:${NC}"
echo "   ./scripts/start-production-network.sh"
echo ""
echo -e "${BLUE}📋 Endpoint'ы узлов:${NC}"
for i in $(seq 0 $((NUM_VALIDATORS-1))); do
    RPC_PORT=$((26657 + i * 100))
    API_PORT=$((1317 + i * 10))
    echo "   validator-$i: RPC http://localhost:$RPC_PORT, API http://localhost:$API_PORT"
done
echo ""

