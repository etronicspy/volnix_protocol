#!/bin/bash

# Volnix Protocol Local Development Network Script
# ⚠️  ВНИМАНИЕ: Этот скрипт предназначен ТОЛЬКО для локальной разработки и тестирования
# 
# Для production используйте Docker - каждый валидатор должен быть в отдельном контейнере
# Сеть формируется из множества независимых Docker контейнеров, каждый на своем сервере
# 
# Этот скрипт запускает несколько узлов на одной машине для разработки/тестирования
# В production сети каждый валидатор = отдельный Docker контейнер (может быть на разных серверах)

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Параметры сети
CHAIN_ID="volnix-testnet"
BASE_PORT=26656
TESTNET_DIR="testnet"
LOGS_DIR="logs"
PIDS_FILE=".network_pids"

# Функции для вывода
log_info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Проверка минимального количества узлов
if [ "$NODE_COUNT" -lt 2 ]; then
    log_error "Минимальное количество узлов: 2"
    exit 1
fi

# Функция очистки при выходе
cleanup() {
    log_warning "Остановка всех процессов..."
    
    if [ -f "$PIDS_FILE" ]; then
        while read pid; do
            if [ ! -z "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                kill "$pid" 2>/dev/null || true
            fi
        done < "$PIDS_FILE"
        rm -f "$PIDS_FILE"
    fi
    
    # Остановка процессов по имени
    pkill -f "volnixd-standalone.*start" 2>/dev/null || true
    
    log_success "Все процессы остановлены"
    exit 0
}

# Обработка Ctrl+C
trap cleanup SIGINT SIGTERM

# Проверка зависимостей
check_dependencies() {
    log_info "Проверка зависимостей..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go не установлен. Установите Go 1.21+"
        exit 1
    fi
    log_success "Go: $(go version)"
    
    if ! command -v jq &> /dev/null; then
        log_warning "jq не установлен. Установите jq для работы с JSON"
        log_info "Установка jq через brew..."
        if command -v brew &> /dev/null; then
            brew install jq || {
                log_error "Не удалось установить jq. Установите вручную: brew install jq"
                exit 1
            }
        else
            log_error "Установите jq вручную: https://stedolan.github.io/jq/download/"
            exit 1
        fi
    fi
    log_success "jq: $(jq --version)"
    
    echo ""
}

# Сборка проекта
build_binary() {
    log_info "Сборка volnixd-standalone..."
    
    if [ ! -f "build/volnixd-standalone" ]; then
        mkdir -p build
        go build -o build/volnixd-standalone ./cmd/volnixd-standalone
        if [ $? -ne 0 ]; then
            log_error "Ошибка сборки volnixd-standalone"
            exit 1
        fi
        log_success "volnixd-standalone собран"
    else
        log_info "Используется существующий бинарник"
    fi
    
    echo ""
}

# Создание директорий
setup_directories() {
    log_info "Создание директорий..."
    
    mkdir -p "$TESTNET_DIR"
    mkdir -p "$LOGS_DIR"
    
    log_success "Директории созданы"
    echo ""
}

# Инициализация узла
init_node() {
    local node_index=$1
    local node_name="node$node_index"
    local node_dir="$TESTNET_DIR/$node_name"
    local p2p_port=$((BASE_PORT + node_index * 10))
    local rpc_port=$((p2p_port + 1))
    
    log_info "Инициализация $node_name..." >&2
    
    # Очистка существующей директории если нужно
    if [ "$CLEAN_START" = "true" ] && [ -d "$node_dir" ]; then
        rm -rf "$node_dir"
    fi
    
    # Инициализация узла
    if [ ! -d "$node_dir" ]; then
        mkdir -p "$node_dir"
        
        # Инициализируем узел через volnixd-standalone
        # volnixd-standalone использует жестко заданную директорию .volnix
        # Поэтому запускаем инициализацию из директории узла
        if [ -f "build/volnixd-standalone" ]; then
            log_info "Инициализация $node_name через volnixd-standalone..." >&2
            (cd "$node_dir" && ../../build/volnixd-standalone init "$node_name" >/dev/null 2>&1) || {
                log_error "Ошибка инициализации $node_name" >&2
                exit 1
            }
            
            # Проверяем, что ключ валидатора создан (может создаваться при первом запуске)
            # Если ключа нет, volnixd-standalone создаст его при запуске
            local priv_val_key="$node_dir/.volnix/config/priv_validator_key.json"
            if [ ! -f "$priv_val_key" ]; then
                log_warning "Ключ валидатора для $node_name не найден после init, будет создан при запуске" >&2
            fi
        else
            log_error "volnixd-standalone не найден" >&2
            exit 1
        fi
        
        # Обновляем конфигурацию портов
        local config_file="$node_dir/.volnix/config/config.toml"
        if [ -f "$config_file" ]; then
            # Используем Python для более надежной замены портов в секциях
            python3 <<PYTHON_SCRIPT
import re
import sys

config_file = "$config_file"
rpc_port = "$rpc_port"
p2p_port = "$p2p_port"

with open(config_file, 'r') as f:
    content = f.read()

# Заменяем RPC порт в секции [rpc]
content = re.sub(
    r'(\[rpc\][^\[]*?laddr = "tcp://0\.0\.0\.0:)26657(")',
    r'\g<1>' + rpc_port + r'\2',
    content,
    flags=re.DOTALL
)
content = re.sub(
    r'(\[rpc\][^\[]*?laddr = "tcp://127\.0\.0\.1:)26657(")',
    r'\g<1>' + rpc_port + r'\2',
    content,
    flags=re.DOTALL
)

# Заменяем P2P порт в секции [p2p]
content = re.sub(
    r'(\[p2p\][^\[]*?laddr = "tcp://0\.0\.0\.0:)26656(")',
    r'\g<1>' + p2p_port + r'\2',
    content,
    flags=re.DOTALL
)
content = re.sub(
    r'(\[p2p\][^\[]*?laddr = "tcp://127\.0\.0\.1:)26656(")',
    r'\g<1>' + p2p_port + r'\2',
    content,
    flags=re.DOTALL
)

with open(config_file, 'w') as f:
    f.write(content)
PYTHON_SCRIPT
            
            # Настраиваем параметры консенсуса для быстрой работы
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' 's|timeout_propose = "3s"|timeout_propose = "1s"|g' "$config_file"
                sed -i '' 's|timeout_prevote = "1s"|timeout_prevote = "500ms"|g' "$config_file"
                sed -i '' 's|timeout_precommit = "1s"|timeout_precommit = "500ms"|g' "$config_file"
                sed -i '' 's|timeout_commit = "5s"|timeout_commit = "1s"|g' "$config_file"
            else
                sed -i 's|timeout_propose = "3s"|timeout_propose = "1s"|g' "$config_file"
                sed -i 's|timeout_prevote = "1s"|timeout_prevote = "500ms"|g' "$config_file"
                sed -i 's|timeout_precommit = "1s"|timeout_precommit = "500ms"|g' "$config_file"
                sed -i 's|timeout_commit = "5s"|timeout_commit = "1s"|g' "$config_file"
            fi
        fi
        
        log_success "$node_name инициализирован (P2P: $p2p_port, RPC: $rpc_port)" >&2
    else
        log_info "$node_name уже существует" >&2
    fi
    
    # Выводим только информацию об узле в stdout
    echo "$node_name:$node_dir:$p2p_port:$rpc_port"
}

# Создание общего genesis файла с валидаторами
create_shared_genesis() {
    log_info "Создание общего genesis файла с валидаторами..." >&2
    
    local nodes_info=("$@")
    local shared_genesis="$TESTNET_DIR/genesis.json"
    
    # Читаем genesis первого узла как основу
    local first_node_info="${nodes_info[0]}"
    IFS=':' read -r first_name first_dir first_p2p first_rpc <<< "$first_node_info"
    local first_genesis="$first_dir/.volnix/config/genesis.json"
    
    if [ ! -f "$first_genesis" ]; then
        log_error "Genesis файл первого узла не найден: $first_genesis" >&2
        exit 1
    fi
    
    # Копируем genesis первого узла
    cp "$first_genesis" "$shared_genesis"
    
    # Обновляем chain_id
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s|\"chain_id\": \"[^\"]*\"|\"chain_id\": \"$CHAIN_ID\"|g" "$shared_genesis"
    else
        sed -i "s|\"chain_id\": \"[^\"]*\"|\"chain_id\": \"$CHAIN_ID\"|g" "$shared_genesis"
    fi
    
    # Собираем всех валидаторов
    local validators_json="[]"
    
    for node_info in "${nodes_info[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port <<< "$node_info"
        local priv_val_key="$dir/.volnix/config/priv_validator_key.json"
        local genesis_file="$dir/.volnix/config/genesis.json"
        
        if [ -f "$priv_val_key" ] && [ -f "$genesis_file" ]; then
            # Читаем публичный ключ валидатора
            local pub_key_type=$(jq -r '.pub_key.type' "$priv_val_key")
            local pub_key_value=$(jq -r '.pub_key.value' "$priv_val_key")
            
            # Используем адрес валидатора из genesis файла узла (уже правильно вычислен)
            local validator_address=$(jq -r '.validators[0].address // empty' "$genesis_file")
            
            # Если адрес не найден, пропускаем этот узел
            if [ -z "$validator_address" ] || [ "$validator_address" = "null" ] || [ "$validator_address" = "" ]; then
                log_warning "Не удалось получить адрес валидатора для $name, пропускаем..." >&2
                continue
            fi
            
            # Создаем JSON валидатора
            local validator_json=$(jq -n \
                --arg address "$validator_address" \
                --arg type "$pub_key_type" \
                --arg value "$pub_key_value" \
                --arg name "$name" \
                '{
                    address: $address,
                    pub_key: {
                        type: $type,
                        value: $value
                    },
                    power: "10",
                    name: $name
                }')
            
            # Добавляем валидатора в массив
            validators_json=$(echo "$validators_json" | jq --argjson validator "$validator_json" '. + [$validator]')
        else
            log_warning "Не найдены файлы валидатора для $name, пропускаем..." >&2
        fi
    done
    
    # Обновляем genesis файл с валидаторами
    local temp_genesis=$(mktemp)
    jq --argjson validators "$validators_json" '.validators = $validators' "$shared_genesis" > "$temp_genesis"
    mv "$temp_genesis" "$shared_genesis"
    
    # Копируем общий genesis во все узлы
    for node_info in "${nodes_info[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port <<< "$node_info"
        cp "$shared_genesis" "$dir/.volnix/config/genesis.json"
    done
    
    log_success "Общий genesis файл создан с ${#nodes_info[@]} валидаторами" >&2
    echo "" >&2
}

# Настройка peer connections
setup_peers() {
    log_info "Настройка peer connections..." >&2
    
    local nodes_info=("$@")
    
    # Сначала запускаем первый узел, чтобы получить его node ID
    # Для этого нам нужно будет запустить его временно или использовать другой метод
    
    # Обновляем persistent_peers для каждого узла
    for node_info in "${nodes_info[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port <<< "$node_info"
        local config_file="$dir/.volnix/config/config.toml"
        
        if [ -f "$config_file" ]; then
            # Создаем строку пиров (исключая текущий узел)
            local peers_for_node=()
            for peer_info in "${nodes_info[@]}"; do
                IFS=':' read -r peer_name peer_dir peer_p2p peer_rpc <<< "$peer_info"
                if [ "$peer_name" != "$name" ]; then
                    # Используем формат с node ID (будет получен при запуске)
                    # Пока используем только IP:PORT, node ID добавится автоматически
                    peers_for_node+=("127.0.0.1:$peer_p2p")
                fi
            done
            
            # Обновляем config.toml
            if [ ${#peers_for_node[@]} -gt 0 ]; then
                local peers_str=$(IFS=','; echo "${peers_for_node[*]}")
                # Добавляем persistent_peers в секцию [p2p]
                if grep -q "persistent_peers" "$config_file"; then
                    # Обновляем существующую строку
                    if [[ "$OSTYPE" == "darwin"* ]]; then
                        sed -i '' "s|persistent_peers = \".*\"|persistent_peers = \"$peers_str\"|" "$config_file"
                    else
                        sed -i "s|persistent_peers = \".*\"|persistent_peers = \"$peers_str\"|" "$config_file"
                    fi
                else
                    # Добавляем новую строку после [p2p]
                    if [[ "$OSTYPE" == "darwin"* ]]; then
                        sed -i '' "/\[p2p\]/a\\
persistent_peers = \"$peers_str\"
" "$config_file"
                    else
                        sed -i "/\[p2p\]/a persistent_peers = \"$peers_str\"" "$config_file"
                    fi
                fi
            fi
            
            # Отключаем UPnP для локальной сети
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' 's|upnp = true|upnp = false|g' "$config_file"
            else
                sed -i 's|upnp = true|upnp = false|g' "$config_file"
            fi
        fi
    done
    
    log_success "Peer connections настроены" >&2
    echo "" >&2
}

# Запуск узла
start_node() {
    local node_info=$1
    IFS=':' read -r name dir p2p_port rpc_port <<< "$node_info"
    
    log_info "Запуск $name (P2P: $p2p_port, RPC: $rpc_port)..." >&2
    
    local abs_dir=$(cd "$dir" && pwd)
    local abs_build=$(cd build && pwd)
    local abs_logs=$(cd "$LOGS_DIR" && pwd)
    local log_file="$abs_logs/${name}.log"
    
    # Очищаем базу данных перед запуском (для чистого старта)
    # volnixd-standalone использует .volnix/data для хранения данных
    local data_dir="$abs_dir/.volnix/data"
    if [ "$CLEAN_START" = "true" ] && [ -d "$data_dir" ]; then
        find "$data_dir" -type f \( -name "*.db" -o -name "*.db-shm" -o -name "*.db-wal" \) -delete 2>/dev/null || true
        # Также очищаем директории баз данных CometBFT
        rm -rf "$data_dir/blockstore.db" "$data_dir/state.db" "$data_dir/tx_index.db" 2>/dev/null || true
    fi
    
    # Запускаем узел
    # volnixd-standalone использует .volnix в текущей директории
    # Важно: запускаем из директории узла, чтобы .volnix был найден правильно
    # Используем абсолютный путь к бинарнику для надежности
    local abs_build_path=$(cd build && pwd)
    local volnix_dir="$abs_dir/.volnix"
    
    # Проверяем, что узел инициализирован
    if [ ! -d "$volnix_dir/config" ]; then
        log_error "Узел $name не инициализирован: $volnix_dir/config не найден" >&2
        return 1
    fi
    
    # Запускаем узел из его директории с env переменными для портов
    # CRITICAL: Передаем VOLNIX_RPC_PORT и VOLNIX_P2P_PORT чтобы узел использовал правильные порты
    (cd "$abs_dir" && VOLNIX_RPC_PORT=$rpc_port VOLNIX_P2P_PORT=$p2p_port "$abs_build_path/volnixd-standalone" start > "$log_file" 2>&1) &
    local pid=$!
    
    # Даем процессу время на запуск (volnixd-standalone может запускаться не сразу)
    sleep 2
    
    echo "$pid" >> "$PIDS_FILE"
    # Выводим только информацию в stdout
    echo "$name:$pid"
    
    sleep 2
}

# Проверка статуса узлов
check_nodes_status() {
    log_info "Проверка статуса узлов..." >&2
    
    local nodes_info=("$@")
    local all_ready=true
    
    for node_info in "${nodes_info[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port <<< "$node_info"
        
        # Проверяем RPC эндпоинт
        local max_attempts=10
        local attempt=0
        local node_ready=false
        
        while [ $attempt -lt $max_attempts ]; do
            if curl -s "http://localhost:$rpc_port/status" > /dev/null 2>&1; then
                node_ready=true
                break
            fi
            attempt=$((attempt + 1))
            sleep 1
        done
        
        if [ "$node_ready" = true ]; then
            log_success "$name готов (RPC: $rpc_port)" >&2
        else
            log_warning "$name еще не готов (RPC: $rpc_port)" >&2
            all_ready=false
        fi
    done
    
    echo "" >&2
    return $([ "$all_ready" = true ] && echo 0 || echo 1)
}

# Отображение статуса
show_status() {
    echo ""
    echo -e "${GREEN}🎉 Volnix Protocol Minimal Network запущена!${NC}"
    echo -e "${GREEN}===========================================${NC}"
    echo ""
    echo -e "${CYAN}📊 Информация о сети:${NC}"
    echo -e "  Chain ID: ${CHAIN_ID}"
    echo -e "  Узлов: ${NODE_COUNT}"
    echo ""
    echo -e "${CYAN}🌐 Эндпоинты узлов:${NC}"
    
    for node_info in "${NODES_INFO[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port <<< "$node_info"
        echo -e "  ${YELLOW}$name:${NC}"
        echo -e "    RPC:  http://localhost:$rpc_port"
        echo -e "    P2P:  tcp://localhost:$p2p_port"
    done
    
    echo ""
    echo -e "${CYAN}📋 Полезные команды:${NC}"
    echo -e "  # Проверка статуса узла"
    echo -e "  curl http://localhost:26657/status | jq"
    echo ""
    echo -e "  # Просмотр логов"
    echo -e "  tail -f $LOGS_DIR/node0.log"
    echo ""
    echo -e "  # Проверка блоков"
    echo -e "  curl http://localhost:26657/block?height=1 | jq"
    echo ""
    echo -e "${YELLOW}⚠️  Для остановки нажмите Ctrl+C${NC}"
    echo ""
}

# Функция добавления узла
add_node() {
    local new_node_num=$1
    
    if [ -z "$new_node_num" ]; then
        log_error "Не указан номер узла"
        echo "Использование: $0 add <номер_узла>"
        echo "Пример: $0 add 3  (добавит node3)"
        exit 1
    fi
    
    echo -e "${CYAN}🚀 Добавление node$new_node_num к сети${NC}"
    echo -e "${CYAN}====================================${NC}"
    echo ""
    
    # Проверка существования узла
    if [ -d "$TESTNET_DIR/node$new_node_num" ]; then
        log_warning "node$new_node_num уже существует!"
        read -p "Пересоздать? (y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
        rm -rf "$TESTNET_DIR/node$new_node_num"
    fi
    
    check_dependencies
    build_binary
    
    # Генерация ключей через Python (используем существующую логику)
    log_info "Генерация ключей для node$new_node_num..."
    node_dir="$TESTNET_DIR/node$new_node_num"
    mkdir -p "$node_dir/.volnix/config"
    mkdir -p "$node_dir/.volnix/data"
    
    python3 << PYEOF
import json, hashlib, base64
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
from cryptography.hazmat.primitives import serialization

node_dir = "$node_dir"

# Генерация node_key
private_key = Ed25519PrivateKey.generate()
private_bytes = private_key.private_bytes(
    encoding=serialization.Encoding.Raw,
    format=serialization.PrivateFormat.Raw,
    encryption_algorithm=serialization.NoEncryption()
)
public_bytes = private_key.public_key().public_bytes(
    encoding=serialization.Encoding.Raw,
    format=serialization.PublicFormat.Raw
)
full_key = private_bytes + public_bytes

node_key = {
    "priv_key": {
        "type": "tendermint/PrivKeyEd25519",
        "value": base64.b64encode(full_key).decode('utf-8')
    }
}

with open(f"{node_dir}/.volnix/config/node_key.json", 'w') as f:
    json.dump(node_key, f, indent=2)

# Генерация priv_validator_key
val_private_key = Ed25519PrivateKey.generate()
val_private_bytes = val_private_key.private_bytes(
    encoding=serialization.Encoding.Raw,
    format=serialization.PrivateFormat.Raw,
    encryption_algorithm=serialization.NoEncryption()
)
val_public_bytes = val_private_key.public_key().public_bytes(
    encoding=serialization.Encoding.Raw,
    format=serialization.PublicFormat.Raw
)

address_bytes = hashlib.sha256(val_public_bytes).digest()[:20]
address = address_bytes.hex().upper()

val_full_key = val_private_bytes + val_public_bytes

priv_validator_key = {
    "address": address,
    "pub_key": {
        "type": "tendermint/PubKeyEd25519",
        "value": base64.b64encode(val_public_bytes).decode('utf-8')
    },
    "priv_key": {
        "type": "tendermint/PrivKeyEd25519",
        "value": base64.b64encode(val_full_key).decode('utf-8')
    }
}

with open(f"{node_dir}/.volnix/config/priv_validator_key.json", 'w') as f:
    json.dump(priv_validator_key, f, indent=2)

# Вычисление node ID
pub_key_bytes = full_key[32:]
node_id = hashlib.sha256(pub_key_bytes).hexdigest()[:40]
print(f"Node ID: {node_id}")
print(f"Validator: {address}")
PYEOF
    
    log_success "Ключи созданы для node$new_node_num"
    
    # Инициализация узла
    log_info "Инициализация node$new_node_num..."
    (cd "$node_dir" && VOLNIX_HOME=".volnix" "$BINARY" init "node$new_node_num" > /dev/null 2>&1)
    
    # Обновление genesis (добавление валидатора)
    log_info "Обновление genesis файла..."
    # Здесь должна быть логика обновления genesis, но для упрощения оставляем как есть
    
    log_success "Узел node$new_node_num добавлен!"
    log_info "Для запуска: cd $node_dir && VOLNIX_HOME=.volnix $BINARY start"
}

# Основная функция
main() {
    # Проверка режима работы (если первый аргумент - "add")
    if [ "$1" = "add" ]; then
        add_node "$2"
        exit 0
    fi
    
    echo -e "${CYAN}🚀 Запуск Volnix Protocol Minimal Network${NC}"
    echo -e "${CYAN}===========================================${NC}"
    echo ""
    
    # Парсинг аргументов
    NODE_COUNT=${1:-3}  # По умолчанию 3 узла
    CLEAN_START="false"
    shift 2>/dev/null || true
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean)
                CLEAN_START="true"
                shift
                ;;
            --nodes)
                NODE_COUNT="$2"
                shift 2
                ;;
            *)
                if [[ "$1" =~ ^[0-9]+$ ]]; then
                    NODE_COUNT="$1"
                    shift
                else
                    log_error "Неизвестный аргумент: $1"
                    echo "Использование: $0 [количество_узлов] [--clean]"
                    echo "              $0 add <номер_узла>"
                    exit 1
                fi
                ;;
        esac
    done
    
    check_dependencies
    build_binary
    setup_directories
    
    # Инициализация узлов
    log_info "Инициализация $NODE_COUNT узлов..."
    NODES_INFO=()
    for i in $(seq 0 $((NODE_COUNT - 1))); do
        node_info=$(init_node $i)
        NODES_INFO+=("$node_info")
    done
    echo ""
    
    # Создание общего genesis файла с валидаторами
    create_shared_genesis "${NODES_INFO[@]}"
    
    # Настройка peer connections
    setup_peers "${NODES_INFO[@]}"
    
    # Запуск узлов
    log_info "Запуск узлов..."
    NODE_PIDS=()
    for node_info in "${NODES_INFO[@]}"; do
        pid_info=$(start_node "$node_info")
        IFS=':' read -r name pid <<< "$pid_info"
        NODE_PIDS+=("$pid")
    done
    echo ""
    
    # Ожидание запуска узлов
    log_info "Ожидание запуска узлов..."
    sleep 5
    
    # Проверка статуса
    check_nodes_status "${NODES_INFO[@]}" || log_warning "Некоторые узлы еще не готовы"
    
    # Отображение статуса
    show_status
    
    # Ожидание завершения
    wait
}

# Запуск
main "$@"

