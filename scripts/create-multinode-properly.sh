#!/bin/bash

# Правильное создание мультинод сети с уникальными ключами

set -e

TESTNET_DIR="testnet-proper"
CHAIN_ID="volnix-testnet"
BINARY="./build/volnixd-standalone"

echo "🚀 Правильная настройка 3-нод сети"
echo "===================================="
echo ""

# Очистка
rm -rf "$TESTNET_DIR" logs
mkdir -p "$TESTNET_DIR" logs

#═══════════════════════════════════════════════════════════
# ШАГ 1: Создание уникальных ключей для каждого узла
#═══════════════════════════════════════════════════════════

echo "🔑 Шаг 1/4: Генерация уникальных ключей..."
echo ""

for i in 0 1 2; do
    node_dir="$TESTNET_DIR/node$i"
    mkdir -p "$node_dir"
    
    echo "  Генерация ключей для node$i..."
    
    # Создаем директории вручную
    mkdir -p "$node_dir/.volnix/config"
    mkdir -p "$node_dir/.volnix/data"
    
    # Генерируем уникальные ключи запуская узел на временных портах и останавливая
    temp_rpc=$((29000 + i))
    temp_p2p=$((29100 + i))
    
    (cd "$node_dir" && \
     VOLNIX_HOME=".volnix" \
     VOLNIX_RPC_PORT=$temp_rpc \
     VOLNIX_P2P_PORT=$temp_p2p \
     "$BINARY" init "node$i" > /dev/null 2>&1) &
    
    INIT_PID=$!
    
    # Даем время на создание файлов
    sleep 5
    
    # Останавливаем
    kill $INIT_PID 2>/dev/null || true
    wait $INIT_PID 2>/dev/null || true
    
    # Проверяем что ключи созданы
    if [ -f "$node_dir/.volnix/config/priv_validator_key.json" ] && [ -f "$node_dir/.volnix/config/node_key.json" ]; then
        echo "    ✅ Ключи созданы"
    else
        echo "    ⚠️  Не все ключи созданы, попробуем еще раз..."
        
        # Запускаем узел кратко для генерации ключей
        (cd "$node_dir" && \
         VOLNIX_HOME=".volnix" \
         VOLNIX_RPC_PORT=$temp_rpc \
         VOLNIX_P2P_PORT=$temp_p2p \
         "$BINARY" start > /dev/null 2>&1) &
        
        START_PID=$!
        sleep 5
        kill $START_PID 2>/dev/null || true
        wait $START_PID 2>/dev/null || true
        
        if [ -f "$node_dir/.volnix/config/priv_validator_key.json" ]; then
            echo "    ✅ Ключи созданы через start"
        fi
    fi
done

echo ""
echo "✅ Все узлы имеют уникальные ключи"
echo ""

# Проверяем уникальность
echo "Проверка уникальности node IDs..."
python3 << 'PYEOF'
import json, hashlib, base64, sys

node_ids = []
for i in range(3):
    try:
        with open(f'testnet-proper/node{i}/.volnix/config/node_key.json', 'r') as f:
            node_key = json.load(f)
        
        priv_key_b64 = node_key['priv_key']['value']
        priv_key_bytes = base64.b64decode(priv_key_b64)
        pub_key_bytes = priv_key_bytes[32:]
        node_id = hashlib.sha256(pub_key_bytes).hexdigest()[:40]
        
        node_ids.append(node_id)
        print(f"  Node {i}: {node_id}")
    except Exception as e:
        print(f"  Node {i}: Error - {e}")
        sys.exit(1)

# Проверка уникальности
if len(node_ids) == len(set(node_ids)):
    print("\n✅ Все node IDs уникальны!")
else:
    print("\n❌ Найдены дубликаты node IDs!")
    sys.exit(1)
PYEOF

echo ""

#═══════════════════════════════════════════════════════════
# ШАГ 2: Создание общего genesis с всеми валидаторами
#═══════════════════════════════════════════════════════════

echo "📝 Шаг 2/4: Создание общего genesis..."
echo ""

python3 << 'PYEOF'
import json

TESTNET_DIR = "testnet-proper"
CHAIN_ID = "volnix-testnet"

# Читаем genesis каждого узла и собираем валидаторов
all_validators = []

for i in range(3):
    genesis_file = f'{TESTNET_DIR}/node{i}/.volnix/config/genesis.json'
    try:
        with open(genesis_file, 'r') as f:
            genesis = json.load(f)
        
        validators = genesis.get('validators', [])
        for val in validators:
            val['name'] = f'node{i}'
        all_validators.extend(validators)
        print(f"  ✅ Добавлен валидатор node{i}")
    except Exception as e:
        print(f"  ⚠️  node{i}: {e}")

# Используем genesis node0 как базу
with open(f'{TESTNET_DIR}/node0/.volnix/config/genesis.json', 'r') as f:
    shared_genesis = json.load(f)

# Обновляем
shared_genesis['chain_id'] = CHAIN_ID
shared_genesis['validators'] = all_validators

# Сохраняем
with open(f'{TESTNET_DIR}/genesis.json', 'w') as f:
    json.dump(shared_genesis, f, indent=2)

print(f"\n✅ Создан общий genesis с {len(all_validators)} валидаторами")

# Копируем на все узлы
for i in range(3):
    node_genesis = f'{TESTNET_DIR}/node{i}/.volnix/config/genesis.json'
    with open(f'{TESTNET_DIR}/genesis.json', 'r') as f:
        content = f.read()
    with open(node_genesis, 'w') as f:
        f.write(content)
    print(f"  ✅ Genesis скопирован на node{i}")
PYEOF

echo ""

#═══════════════════════════════════════════════════════════
# ШАГ 3: Настройка persistent_peers
#═══════════════════════════════════════════════════════════

echo "🔗 Шаг 3/4: Настройка P2P соединений..."
echo ""

python3 << 'PYEOF'
import json, hashlib, base64, re

# Получаем node IDs
node_ids = []
for i in range(3):
    with open(f'testnet-proper/node{i}/.volnix/config/node_key.json', 'r') as f:
        node_key = json.load(f)
    
    priv_key_b64 = node_key['priv_key']['value']
    priv_key_bytes = base64.b64decode(priv_key_b64)
    pub_key_bytes = priv_key_bytes[32:]
    node_id = hashlib.sha256(pub_key_bytes).hexdigest()[:40]
    node_ids.append(node_id)

print(f"Node IDs получены:")
for i, nid in enumerate(node_ids):
    print(f"  node{i}: {nid}")
print()

# Настраиваем peers для каждого узла
peers_config = {
    0: f"{node_ids[1]}@127.0.0.1:26666,{node_ids[2]}@127.0.0.1:26676",
    1: f"{node_ids[0]}@127.0.0.1:26656,{node_ids[2]}@127.0.0.1:26676",
    2: f"{node_ids[0]}@127.0.0.1:26656,{node_ids[1]}@127.0.0.1:26666",
}

for i, peers in peers_config.items():
    config_file = f'testnet-proper/node{i}/.volnix/config/config.toml'
    
    with open(config_file, 'r') as f:
        content = f.read()
    
    # Заменяем persistent_peers
    content = re.sub(r'persistent_peers = ".*"', f'persistent_peers = "{peers}"', content)
    
    with open(config_file, 'w') as f:
        f.write(content)
    
    print(f"✅ node{i}: peers настроены ({len(peers.split(','))} peers)")

print()
PYEOF

echo "✅ P2P соединения настроены"
echo ""

#═══════════════════════════════════════════════════════════
# ШАГ 4: Запуск узлов
#═══════════════════════════════════════════════════════════

echo "🚀 Шаг 4/4: Запуск узлов..."
echo ""

declare -a PIDS

for i in 0 1 2; do
    node_dir="$TESTNET_DIR/node$i"
    rpc_port=$((26657 + i * 10))
    p2p_port=$((26656 + i * 10))
    
    echo "  Запуск node$i (RPC: $rpc_port, P2P: $p2p_port)..."
    
    (cd "$node_dir" && \
     VOLNIX_HOME=".volnix" \
     VOLNIX_RPC_PORT=$rpc_port \
     VOLNIX_P2P_PORT=$p2p_port \
     "$BINARY" start > "../../logs/node$i.log" 2>&1 &)
    
    PIDS[$i]=$!
    sleep 4
done

echo ""
echo "✅ Узлы запущены!"
echo "PIDs: ${PIDS[@]}"
echo ""

# Сохранение
echo "${PIDS[@]}" > "$TESTNET_DIR/pids.txt"

# Ожидание
echo "⏳ Ожидание P2P соединений (25 секунд)..."
sleep 25

#═══════════════════════════════════════════════════════════
# ПРОВЕРКА
#═══════════════════════════════════════════════════════════

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🔍 ПРОВЕРКА МУЛЬТИНОД СЕТИ"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

for i in 0 1 2; do
    rpc_port=$((26657 + i * 10))
    
    if curl -s "http://localhost:$rpc_port/status" > /dev/null 2>&1; then
        HEIGHT=$(curl -s "http://localhost:$rpc_port/status" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null)
        PEERS=$(curl -s "http://localhost:$rpc_port/net_info" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('n_peers',0))" 2>/dev/null)
        
        echo "✅ Node $i (http://localhost:$rpc_port):"
        echo "   Блок: $HEIGHT"
        echo "   Peers: $PEERS"
        
        if [ "$PEERS" != "0" ]; then
            echo "   Соединения:"
            curl -s "http://localhost:$rpc_port/net_info" | python3 -c "import sys,json; peers=json.load(sys.stdin).get('result',{}).get('peers',[]); [print(f'     - {p.get(\"node_info\",{}).get(\"moniker\",\"unknown\")}') for p in peers]" 2>/dev/null
        fi
    else
        echo "❌ Node $i: не отвечает"
    fi
    echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Проверка синхронизации
echo "📊 Синхронизация блоков:"
H0=$(curl -s "http://localhost:26657/status" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null)
H1=$(curl -s "http://localhost:26667/status" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null)
H2=$(curl -s "http://localhost:26677/status" | python3 -c "import sys,json; print(json.load(sys.stdin).get('result',{}).get('sync_info',{}).get('latest_block_height',0))" 2>/dev/null)

echo "  Node 0: $H0"
echo "  Node 1: $H1"
echo "  Node 2: $H2"

if [ "$H0" = "$H1" ] && [ "$H1" = "$H2" ] && [ "$H0" != "0" ]; then
    echo ""
    echo "🎉 ВСЕ УЗЛЫ СИНХРОНИЗИРОВАНЫ!"
elif [ "$H0" != "0" ] && [ "$H1" != "0" ] && [ "$H2" != "0" ]; then
    echo ""
    echo "⚠️  Узлы создают блоки, но не синхронизированы (может потребоваться больше времени)"
else
    echo ""
    echo "⚠️  Узлы не создают блоки или не все запустились"
fi

echo ""
echo "📋 Управление:"
echo "  Логи: tail -f logs/node*.log"
echo "  Остановка: kill ${PIDS[@]}"
echo ""

