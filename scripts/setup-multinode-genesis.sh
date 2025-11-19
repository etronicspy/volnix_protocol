#!/bin/bash

# Скрипт создания общего genesis для мультинод сети
# Правильно собирает validator keys и создает общий genesis

set -e

NUM_NODES=3
CHAIN_ID="volnix-testnet"
TESTNET_DIR="testnet-multinode"
BINARY="./build/volnixd-standalone"

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}=== Создание мультинод genesis ===${NC}"
echo ""

# Проверка бинарника
if [ ! -f "$BINARY" ]; then
    echo "Сборка бинарника..."
    make build-standalone
fi

# Создание директорий
mkdir -p "$TESTNET_DIR"

# Массивы для хранения информации
declare -a NODE_IDS
declare -a VALIDATOR_PUBKEYS
declare -a VALIDATOR_ADDRESSES
declare -a RPC_PORTS
declare -a P2P_PORTS

echo -e "${BLUE}📦 Шаг 1: Инициализация узлов${NC}"
echo ""

for i in 0 1 2; do
    node_name="node$i"
    node_dir="$TESTNET_DIR/$node_name"
    rpc_port=$((26657 + i * 10))
    p2p_port=$((26656 + i * 10))
    
    RPC_PORTS[$i]=$rpc_port
    P2P_PORTS[$i]=$p2p_port
    
    echo "Инициализация $node_name..."
    
    mkdir -p "$node_dir"
    
    # Инициализация
    (cd "$node_dir" && VOLNIX_HOME=".volnix" "$BINARY" init "$node_name" > /dev/null 2>&1)
    
    # Получение node ID
    node_id_file="$node_dir/.volnix/config/node_key.json"
    if [ -f "$node_id_file" ]; then
        NODE_ID=$(cat "$node_id_file" | python3 -c "import sys,json,hashlib; d=json.load(sys.stdin); key=d['priv_key']['value']; import base64; key_bytes=base64.b64decode(key); pub=key_bytes[32:]; print(hashlib.sha256(pub).hexdigest()[:40])" 2>/dev/null || echo "unknown")
        NODE_IDS[$i]=$NODE_ID
    else
        # Запускаем узел кратко чтобы создались ключи
        (cd "$node_dir" && VOLNIX_HOME=".volnix" VOLNIX_RPC_PORT=$rpc_port VOLNIX_P2P_PORT=$p2p_port "$BINARY" start > /dev/null 2>&1) &
        local pid=$!
        sleep 3
        kill $pid 2>/dev/null || true
        
        if [ -f "$node_id_file" ]; then
            NODE_ID=$(cat "$node_id_file" | python3 -c "import sys,json,hashlib; d=json.load(sys.stdin); key=d['priv_key']['value']; import base64; key_bytes=base64.b64decode(key); pub=key_bytes[32:]; print(hashlib.sha256(pub).hexdigest()[:40])" 2>/dev/null || echo "unknown")
            NODE_IDS[$i]=$NODE_ID
        fi
    fi
    
    # Получение validator pubkey
    priv_val_key="$node_dir/.volnix/config/priv_validator_key.json"
    if [ -f "$priv_val_key" ]; then
        VAL_PUBKEY=$(cat "$priv_val_key" | jq -r '.pub_key')
        VALIDATOR_PUBKEYS[$i]=$VAL_PUBKEY
        
        # Вычисление validator address
        VAL_ADDR=$(echo "$VAL_PUBKEY" | python3 << 'PYEOF'
import sys, json, hashlib, base64
pubkey = json.load(sys.stdin)
pub_value = base64.b64decode(pubkey['value'])
addr_bytes = hashlib.sha256(pub_value).digest()[:20]
print(addr_bytes.hex().upper())
PYEOF
)
        VALIDATOR_ADDRESSES[$i]=$VAL_ADDR
    fi
    
    echo "  ✅ $node_name"
    echo "     Node ID: ${NODE_IDS[$i]}"
    echo "     Validator: ${VALIDATOR_ADDRESSES[$i]}"
    echo ""
done

echo -e "${BLUE}📝 Шаг 2: Создание общего genesis${NC}"
echo ""

# Используем genesis первого узла как базу
BASE_GENESIS="$TESTNET_DIR/node0/.volnix/config/genesis.json"
SHARED_GENESIS="$TESTNET_DIR/genesis.json"

cp "$BASE_GENESIS" "$SHARED_GENESIS"

# Обновляем chain_id
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s|\"chain_id\": \"[^\"]*\"|\"chain_id\": \"$CHAIN_ID\"|g" "$SHARED_GENESIS"
else
    sed -i "s|\"chain_id\": \"[^\"]*\"|\"chain_id\": \"$CHAIN_ID\"|g" "$SHARED_GENESIS"
fi

# Создаем массив валидаторов
VALIDATORS_JSON="["

for i in 0 1 2; do
    if [ ! -z "${VALIDATOR_ADDRESSES[$i]}" ] && [ "${VALIDATOR_ADDRESSES[$i]}" != "null" ]; then
        if [ $i -gt 0 ]; then
            VALIDATORS_JSON="${VALIDATORS_JSON},"
        fi
        
        VALIDATORS_JSON="${VALIDATORS_JSON}
    {
      \"address\": \"${VALIDATOR_ADDRESSES[$i]}\",
      \"pub_key\": ${VALIDATOR_PUBKEYS[$i]},
      \"power\": \"10\",
      \"name\": \"node$i\"
    }"
    fi
done

VALIDATORS_JSON="${VALIDATORS_JSON}
  ]"

# Обновляем validators в genesis используя python
python3 << PYEOF
import json

with open('$SHARED_GENESIS', 'r') as f:
    genesis = json.load(f)

# Парсим validators JSON
validators = json.loads('''$VALIDATORS_JSON''')

genesis['validators'] = validators

with open('$SHARED_GENESIS', 'w') as f:
    json.dump(genesis, f, indent=2)

print(f"✅ Genesis обновлен с {len(validators)} валидаторами")
PYEOF

echo ""

# Копируем общий genesis на все узлы
echo -e "${BLUE}📋 Шаг 3: Распространение genesis${NC}"
echo ""

for i in 0 1 2; do
    node_dir="$TESTNET_DIR/node$i"
    cp "$SHARED_GENESIS" "$node_dir/.volnix/config/genesis.json"
    echo "  ✅ node$i: genesis обновлен"
done

echo ""
echo -e "${BLUE}🔗 Шаг 4: Настройка persistent_peers${NC}"
echo ""

for i in 0 1 2; do
    node_dir="$TESTNET_DIR/node$i"
    config_file="$node_dir/.volnix/config/config.toml"
    
    # Создаем список пиров (все узлы кроме текущего)
    PEERS=""
    for j in 0 1 2; do
        if [ $i -ne $j ]; then
            if [ ! -z "$PEERS" ]; then
                PEERS="${PEERS},"
            fi
            PEERS="${PEERS}${NODE_IDS[$j]}@127.0.0.1:${P2P_PORTS[$j]}"
        fi
    done
    
    # Обновляем config.toml
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS\"|g" "$config_file"
    else
        sed -i "s|persistent_peers = \".*\"|persistent_peers = \"$PEERS\"|g" "$config_file"
    fi
    
    echo "  ✅ node$i: peers настроены"
done

echo ""
echo -e "${GREEN}✅ Мультинод genesis готов!${NC}"
echo ""
echo "Chain ID: $CHAIN_ID"
echo "Validators: ${#VALIDATOR_ADDRESSES[@]}"
echo ""
echo "Для запуска:"
echo "  ./scripts/start-multinode-network.sh"
echo ""


