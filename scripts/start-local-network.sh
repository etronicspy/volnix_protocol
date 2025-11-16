#!/bin/bash

# Volnix Protocol Local Network Startup Script
# Запускает локальную сеть из 3 узлов с кошельком и блокчейн-эксплорером

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Параметры сети
NODE_COUNT=3
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
    pkill -f "volnixd.*start" 2>/dev/null || true
    pkill -f "npm.*start" 2>/dev/null || true
    pkill -f "python3.*-m.*http.server" 2>/dev/null || true
    
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
    
    if ! command -v node &> /dev/null; then
        log_error "Node.js не установлен. Установите Node.js 18+"
        exit 1
    fi
    log_success "Node.js: $(node --version)"
    
    if ! command -v npm &> /dev/null; then
        log_error "npm не установлен"
        exit 1
    fi
    log_success "npm: $(npm --version)"
    
    echo ""
}

# Сборка проекта
build_binary() {
    log_info "Сборка volnixd..."
    
    if [ ! -f "build/volnixd" ]; then
        mkdir -p build
        go build -o build/volnixd ./cmd/volnixd
        if [ $? -ne 0 ]; then
            log_error "Ошибка сборки volnixd"
            exit 1
        fi
        log_success "volnixd собран"
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
    local api_port=$((rpc_port + 1000))
    local grpc_port=$((p2p_port + 1000))
    
    log_info "Инициализация $node_name..." >&2
    
    # Очистка существующей директории если нужно
    if [ "$CLEAN_START" = "true" ] && [ -d "$node_dir" ]; then
        rm -rf "$node_dir"
    fi
    
    # Инициализация узла
    if [ ! -d "$node_dir" ]; then
        mkdir -p "$node_dir"
        
        # Используем volnixd init если доступно, иначе создаем базовую структуру
        if [ -f "build/volnixd" ]; then
            ./build/volnixd init "$node_name" --home "$node_dir" --chain-id "$CHAIN_ID" >/dev/null 2>&1 || {
                # Если команда не поддерживается, создаем структуру вручную
                mkdir -p "$node_dir/config"
                mkdir -p "$node_dir/data"
            }
        else
            mkdir -p "$node_dir/config"
            mkdir -p "$node_dir/data"
        fi
        
        # Создание базового config.toml
        create_config_toml "$node_dir/config/config.toml" "$p2p_port" "$rpc_port"
        
        # Создание базового app.toml
        create_app_toml "$node_dir/config/app.toml" "$api_port" "$grpc_port"
        
        log_success "$node_name инициализирован (P2P: $p2p_port, RPC: $rpc_port)" >&2
    else
        log_info "$node_name уже существует" >&2
    fi
    
    # Выводим только информацию об узле в stdout
    echo "$node_name:$node_dir:$p2p_port:$rpc_port:$api_port:$grpc_port"
}

# Создание config.toml
create_config_toml() {
    local config_file=$1
    local p2p_port=$2
    local rpc_port=$3
    
    cat > "$config_file" <<EOF
# Volnix Node Configuration

# RPC Server Configuration
[rpc]
laddr = "tcp://0.0.0.0:$rpc_port"
cors_allowed_origins = ["*"]
cors_allowed_methods = ["HEAD", "GET", "POST"]
cors_allowed_headers = ["Origin", "Accept", "Content-Type", "X-Requested-With", "X-Server-Time"]

# P2P Configuration
[p2p]
laddr = "tcp://0.0.0.0:$p2p_port"
external_address = "127.0.0.1:$p2p_port"
max_num_inbound_peers = 40
max_num_outbound_peers = 10
flush_throttle_timeout = "100ms"
max_packet_msg_payload_size = 1024
send_rate = 5120000
recv_rate = 5120000

# Consensus Configuration
[consensus]
timeout_propose = "3s"
timeout_prevote = "1s"
timeout_precommit = "1s"
timeout_commit = "5s"
create_empty_blocks = true
create_empty_blocks_interval = "0s"

# Mempool Configuration
[mempool]
size = 5000
cache_size = 10000

# State Sync Configuration
[statesync]
enable = false

# Block Sync Configuration
[blocksync]
version = "v0"

# Logging
[log]
level = "info"
format = "plain"
EOF
}

# Создание app.toml
create_app_toml() {
    local app_file=$1
    local api_port=$2
    local grpc_port=$3
    
    cat > "$app_file" <<EOF
# Volnix Application Configuration

# API Configuration
[api]
enable = true
swagger = true
address = "tcp://0.0.0.0:$api_port"
max-open-connections = 1000
rpc-read-timeout = 10
rpc-write-timeout = 0
rpc-max-body-bytes = 1000000
enabled-unsafe-cors = true

# gRPC Configuration
[grpc]
enable = true
address = "0.0.0.0:$grpc_port"

# State Sync Configuration
[state-sync]
snapshot-interval = 0
snapshot-keep-recent = 2
EOF
}

# Создание genesis файла с валидаторами
create_genesis_file() {
    log_info "Создание genesis файла..." >&2
    
    local nodes_info=("$@")
    local genesis_file="$TESTNET_DIR/genesis.json"
    
    # Базовый genesis файл
    cat > "$genesis_file" <<EOF
{
  "genesis_time": "$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")",
  "chain_id": "$CHAIN_ID",
  "initial_height": "1",
  "consensus_params": {
    "block": {
      "max_bytes": "22020096",
      "max_gas": "-1",
      "time_iota_ms": "1000"
    },
    "evidence": {
      "max_age_num_blocks": "100000",
      "max_age_duration": "172800000000000",
      "max_bytes": "1048576"
    },
    "validator": {
      "pub_key_types": ["ed25519"]
    },
    "version": {}
  },
  "validators": [],
  "app_hash": "",
  "app_state": {}
}
EOF
    
    # Копируем genesis файл во все узлы (кроме node0, который использует standalone)
    for node_info in "${nodes_info[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port api_port grpc_port <<< "$node_info"
        # Пропускаем node0, так как standalone создает свой genesis
        if [ "$name" = "node0" ]; then
            continue
        fi
        # Копируем только если директория config существует
        if [ -d "$dir/config" ]; then
            cp "$genesis_file" "$dir/config/genesis.json"
        fi
    done
    
    log_success "Genesis файл создан" >&2
    echo "" >&2
}

# Настройка peer connections
setup_peers() {
    log_info "Настройка peer connections..." >&2
    
    local nodes_info=("$@")
    
    # Обновляем persistent_peers для каждого узла
    for node_info in "${nodes_info[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port api_port grpc_port <<< "$node_info"
        local config_file="$dir/config/config.toml"
        
        if [ -f "$config_file" ]; then
            # Создаем строку пиров (исключая текущий узел)
            local peers_for_node=()
            for peer_info in "${nodes_info[@]}"; do
                IFS=':' read -r peer_name peer_dir peer_p2p peer_rpc peer_api peer_grpc <<< "$peer_info"
                if [ "$peer_name" != "$name" ]; then
                    # Используем простой формат для подключения
                    # В реальной сети node ID будет получен при запуске
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
        fi
    done
    
    log_success "Peer connections настроены" >&2
    echo "" >&2
}

# Запуск узла
start_node() {
    local node_info=$1
    IFS=':' read -r name dir p2p_port rpc_port api_port grpc_port <<< "$node_info"
    
    log_info "Запуск $name (P2P: $p2p_port, RPC: $rpc_port)..." >&2
    
    local abs_dir=$(cd "$dir" && pwd)
    local abs_build=$(cd build && pwd)
    local abs_logs=$(cd "$LOGS_DIR" && pwd)
    local log_file="$abs_logs/${name}.log"
    
    # Используем volnixd-standalone для реального RPC сервера
    # Для node0 используем standalone с реальным CometBFT
    if [ "$name" = "node0" ] && [ -f "$abs_build/volnixd-standalone" ]; then
        log_info "Используется volnixd-standalone для $name" >&2
        # Создаем директорию .volnix внутри директории узла для standalone
        local standalone_home="$abs_dir/.volnix"
        mkdir -p "$standalone_home"
        
        # Инициализируем standalone узел если нужно
        if [ ! -f "$standalone_home/config/config.toml" ]; then
            log_info "Инициализация standalone узла $name..." >&2
            (cd "$abs_dir" && VOLNIX_HOME="$standalone_home" "$abs_build/volnixd-standalone" init "$name" >/dev/null 2>&1) || true
        fi
        
        # Для standalone узла используем chain-id "volnix-standalone" (дефолтное значение в коде)
        # Это необходимо, так как standalone узел жестко закодирован с этим chain-id
        local standalone_chain_id="volnix-standalone"
        if [ -f "$standalone_home/config/genesis.json" ]; then
            log_info "Установка chain-id для standalone узла: $standalone_chain_id..." >&2
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' "s|\"chain_id\": \"[^\"]*\"|\"chain_id\": \"$standalone_chain_id\"|g" "$standalone_home/config/genesis.json" 2>/dev/null || true
            else
                sed -i "s|\"chain_id\": \"[^\"]*\"|\"chain_id\": \"$standalone_chain_id\"|g" "$standalone_home/config/genesis.json" 2>/dev/null || true
            fi
        fi
        
        # Обновляем конфигурацию портов для standalone
        if [ -f "$standalone_home/config/config.toml" ]; then
            # Обновляем RPC порт
            if [[ "$OSTYPE" == "darwin"* ]]; then
                sed -i '' "s|laddr = \"tcp://0.0.0.0:26657\"|laddr = \"tcp://0.0.0.0:$rpc_port\"|g" "$standalone_home/config/config.toml" 2>/dev/null || true
                # Обновляем P2P порт
                sed -i '' "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:$p2p_port\"|g" "$standalone_home/config/config.toml" 2>/dev/null || true
            else
                sed -i "s|laddr = \"tcp://0.0.0.0:26657\"|laddr = \"tcp://0.0.0.0:$rpc_port\"|g" "$standalone_home/config/config.toml" 2>/dev/null || true
                # Обновляем P2P порт
                sed -i "s|laddr = \"tcp://0.0.0.0:26656\"|laddr = \"tcp://0.0.0.0:$p2p_port\"|g" "$standalone_home/config/config.toml" 2>/dev/null || true
            fi
        fi
        
        # ВАЖНО: Полностью очищаем базу данных перед каждым запуском standalone узла
        # Это необходимо, так как база данных может содержать старый chain-id
        if [ -d "$standalone_home/data" ]; then
            log_info "Полная очистка базы данных для standalone узла..." >&2
            # Сохраняем priv_validator_state.json если он существует
            if [ -f "$standalone_home/data/priv_validator_state.json" ]; then
                cp "$standalone_home/data/priv_validator_state.json" "$standalone_home/data/priv_validator_state.json.bak" 2>/dev/null || true
            fi
            # Удаляем ВСЕ файлы базы данных (включая все .db файлы)
            find "$standalone_home/data" -type f \( -name "*.db" -o -name "*.db-shm" -o -name "*.db-wal" \) -delete 2>/dev/null || true
            # Восстанавливаем priv_validator_state.json
            if [ -f "$standalone_home/data/priv_validator_state.json.bak" ]; then
                mv "$standalone_home/data/priv_validator_state.json.bak" "$standalone_home/data/priv_validator_state.json"
            else
                echo '{"height":"0","round":0,"step":0}' > "$standalone_home/data/priv_validator_state.json"
            fi
        fi
        
        # Создаем необходимые файлы для standalone перед запуском
        mkdir -p "$standalone_home/data"
        if [ ! -f "$standalone_home/data/priv_validator_state.json" ]; then
            echo '{"height":"0","round":0,"step":0}' > "$standalone_home/data/priv_validator_state.json"
        fi
        
        # Запускаем standalone узел с правильной домашней директорией
        (cd "$abs_dir" && VOLNIX_HOME="$standalone_home" "$abs_build/volnixd-standalone" start > "$log_file" 2>&1) &
        local pid=$!
    else
        # Для остальных узлов используем обычный volnixd (демо)
        (cd "$abs_dir" && VOLNIX_HOME="$abs_dir" "$abs_build/volnixd" start > "$log_file" 2>&1) &
        local pid=$!
    fi
    
    echo "$pid" >> "$PIDS_FILE"
    # Выводим только информацию в stdout
    echo "$name:$pid"
    
    sleep 2
}

# Установка зависимостей wallet-ui
install_wallet_dependencies() {
    log_info "Проверка зависимостей wallet-ui..."
    
    if [ ! -d "frontend/wallet-ui/node_modules" ]; then
        log_info "Установка зависимостей wallet-ui..."
        cd frontend/wallet-ui
        npm install
        cd ../..
        log_success "Зависимости wallet-ui установлены"
    else
        log_info "Зависимости wallet-ui уже установлены"
    fi
    
    echo ""
}

# Запуск wallet-ui
start_wallet_ui() {
    log_info "Запуск Wallet UI..."
    
    cd frontend/wallet-ui
    npm start > "../../$LOGS_DIR/wallet-ui.log" 2>&1 &
    local pid=$!
    cd ../..
    
    echo "$pid" >> "$PIDS_FILE"
    log_success "Wallet UI запущен (http://localhost:3000)"
    echo ""
    
    sleep 3
}

# Запуск blockchain-explorer
start_explorer() {
    log_info "Запуск Blockchain Explorer..."
    
    cd frontend/blockchain-explorer
    
    # Запуск простого HTTP сервера
    python3 -m http.server 8080 > "../../$LOGS_DIR/explorer.log" 2>&1 &
    local pid=$!
    cd ../..
    
    echo "$pid" >> "$PIDS_FILE"
    log_success "Blockchain Explorer запущен (http://localhost:8080)"
    echo ""
    
    sleep 2
}

# Отображение статуса
show_status() {
    echo ""
    echo -e "${GREEN}🎉 Volnix Protocol Local Network запущена!${NC}"
    echo -e "${GREEN}===========================================${NC}"
    echo ""
    echo -e "${CYAN}📊 Информация о сети:${NC}"
    echo -e "  Chain ID: ${CHAIN_ID}"
    echo -e "  Узлов: ${NODE_COUNT}"
    echo ""
    echo -e "${CYAN}🌐 Эндпоинты узлов:${NC}"
    
    for node_info in "${NODES_INFO[@]}"; do
        IFS=':' read -r name dir p2p_port rpc_port api_port grpc_port <<< "$node_info"
        echo -e "  ${YELLOW}$name:${NC}"
        echo -e "    RPC:  http://localhost:$rpc_port"
        echo -e "    API:  http://localhost:$api_port"
        echo -e "    P2P:  tcp://localhost:$p2p_port"
        echo -e "    gRPC: localhost:$grpc_port"
    done
    
    echo ""
    echo -e "${CYAN}💰 Wallet UI:${NC}"
    echo -e "  http://localhost:3000"
    echo ""
    echo -e "${CYAN}🔍 Blockchain Explorer:${NC}"
    echo -e "  http://localhost:8080"
    echo ""
    echo -e "${CYAN}📋 Полезные команды:${NC}"
    echo -e "  # Проверка статуса узла"
    echo -e "  ./build/volnixd status --home $TESTNET_DIR/node0"
    echo ""
    echo -e "  # Просмотр логов"
    echo -e "  tail -f $LOGS_DIR/node0.log"
    echo ""
    echo -e "${YELLOW}⚠️  Для остановки нажмите Ctrl+C${NC}"
    echo ""
}

# Основная функция
main() {
    echo -e "${CYAN}🚀 Запуск Volnix Protocol Local Network${NC}"
    echo -e "${CYAN}===========================================${NC}"
    echo ""
    
    # Парсинг аргументов
    CLEAN_START="false"
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean)
                CLEAN_START="true"
                shift
                ;;
            *)
                log_error "Неизвестный аргумент: $1"
                echo "Использование: $0 [--clean]"
                exit 1
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
    
    # Создание genesis файла
    create_genesis_file "${NODES_INFO[@]}"
    
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
    
    # Установка и запуск wallet-ui
    install_wallet_dependencies
    start_wallet_ui
    
    # Запуск explorer
    start_explorer
    
    # Отображение статуса
    show_status
    
    # Ожидание завершения
    wait
}

# Запуск
main "$@"

