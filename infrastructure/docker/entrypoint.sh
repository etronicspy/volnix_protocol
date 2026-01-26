#!/bin/sh
set -e

# Volnix Protocol Docker Entrypoint
# Автоматическая инициализация и запуск узла

VOLNIX_HOME="${VOLNIX_HOME:-/home/volnix/.volnix}"
MONIKER="${MONIKER:-validator}"
CHAIN_ID="${CHAIN_ID:-volnix-standalone}"

echo "🚀 Volnix Protocol Node Entrypoint"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📁 Home: $VOLNIX_HOME"
echo "🏷️  Moniker: $MONIKER"
echo "🔗 Chain ID: $CHAIN_ID"
echo ""

# Убеждаемся, что директории существуют
mkdir -p "$VOLNIX_HOME/config"
mkdir -p "$VOLNIX_HOME/data"
mkdir -p "$VOLNIX_HOME/keyring-test"

# Проверяем, инициализирован ли узел
if [ ! -f "$VOLNIX_HOME/config/genesis.json" ]; then
    echo "📦 Узел не инициализирован. Выполняю инициализацию..."
    echo ""
    
    # Создаем временную директорию для инициализации (с правами записи)
    INIT_TMP="/tmp/volnix-init-$$"
    mkdir -p "$INIT_TMP/.volnix/config"
    mkdir -p "$INIT_TMP/.volnix/data"
    
    # Инициализируем узел во временной директории
    cd "$INIT_TMP"
    HOME="$INIT_TMP" volnixd-standalone init "$MONIKER" || {
        echo "❌ Ошибка инициализации узла"
        rm -rf "$INIT_TMP"
        exit 1
    }
    
    # Копируем файлы из временной директории в основную
    if [ -f "$INIT_TMP/.volnix/config/genesis.json" ]; then
        cp "$INIT_TMP/.volnix/config/genesis.json" "$VOLNIX_HOME/config/genesis.json" && \
        echo "✅ Genesis файл создан и скопирован"
    else
        echo "⚠️  Genesis файл не был создан"
    fi
    
    if [ -f "$INIT_TMP/.volnix/config/config.toml" ]; then
        cp "$INIT_TMP/.volnix/config/config.toml" "$VOLNIX_HOME/config/config.toml"
    fi
    
    # Очищаем временную директорию
    rm -rf "$INIT_TMP"
    
    echo "✅ Узел успешно инициализирован"
    echo ""
    
    # Обновляем chain_id в genesis.json если указан
    if [ "$CHAIN_ID" != "volnix-standalone" ] && [ -f "$VOLNIX_HOME/config/genesis.json" ]; then
        if command -v jq >/dev/null 2>&1; then
            jq ".chain_id = \"$CHAIN_ID\"" "$VOLNIX_HOME/config/genesis.json" > "$VOLNIX_HOME/config/genesis.json.tmp" && \
            mv "$VOLNIX_HOME/config/genesis.json.tmp" "$VOLNIX_HOME/config/genesis.json"
            echo "✅ Chain ID обновлен: $CHAIN_ID"
        fi
    fi
else
    echo "✅ Узел уже инициализирован (genesis.json существует)"
    echo "   Используется существующий genesis.json из volume"
    echo ""
fi

# КРИТИЧЕСКИ ВАЖНО: Убеждаемся, что genesis.json существует перед запуском start
# Иначе volnixd-standalone start попытается его создать и получит permission denied
if [ ! -f "$VOLNIX_HOME/config/genesis.json" ]; then
    echo "❌ КРИТИЧЕСКАЯ ОШИБКА: genesis.json не существует и не может быть создан!"
    echo "   Проверьте права доступа к volume $VOLNIX_HOME/config"
    exit 1
fi

# Проверяем конфигурацию
if [ ! -f "$VOLNIX_HOME/config/config.toml" ]; then
    echo "⚠️  Конфигурационный файл отсутствует"
    echo "   Создаем его через volnixd-standalone init..."
    
    # Создаем временную директорию для init
    INIT_TMP=$(mktemp -d)
    chmod 777 "$INIT_TMP"
    
    # Запускаем init в временной директории
    volnixd-standalone init "$MONIKER" --home "$INIT_TMP" > /dev/null 2>&1
    
    # Копируем config.toml если он был создан
    if [ -f "$INIT_TMP/.volnix/config/config.toml" ]; then
        cp "$INIT_TMP/.volnix/config/config.toml" "$VOLNIX_HOME/config/config.toml"
        chmod 666 "$VOLNIX_HOME/config/config.toml"
        echo "✅ config.toml создан"
    else
        echo "❌ Не удалось создать config.toml"
        rm -rf "$INIT_TMP"
        exit 1
    fi
    
    rm -rf "$INIT_TMP"
fi

# Настройка persistent_peers если указано
# ВАЖНО: Не перезаписываем persistent_peers если они уже настроены в config.toml
# Это позволяет использовать update-persistent-peers.sh для правильной настройки
if [ -n "$PERSISTENT_PEERS" ]; then
    CONFIG_FILE="$VOLNIX_HOME/config/config.toml"
    
    # Проверяем, есть ли уже persistent_peers в config.toml
    if grep -q "^persistent_peers" "$CONFIG_FILE"; then
        # Если persistent_peers уже настроены, не перезаписываем их
        # Это позволяет использовать update-persistent-peers.sh для правильной настройки
        echo "ℹ️  Persistent peers уже настроены в config.toml, пропускаем обновление из переменной окружения"
    else
        # Только если persistent_peers отсутствуют, добавляем их
        echo "🔗 Настройка persistent peers из переменной окружения..."
        sed -i "/\[p2p\]/a persistent_peers = \"$PERSISTENT_PEERS\"" "$CONFIG_FILE"
        echo "✅ Persistent peers настроены: $PERSISTENT_PEERS"
    fi
    echo ""
fi

# Настройка create_empty_blocks для CosmJS совместимости
CONFIG_FILE="$VOLNIX_HOME/config/config.toml"
if [ -f "$CONFIG_FILE" ]; then
    # Убеждаемся что create_empty_blocks включен
    if grep -q "^create_empty_blocks" "$CONFIG_FILE"; then
        sed -i 's|^create_empty_blocks = .*|create_empty_blocks = true|' "$CONFIG_FILE"
    else
        sed -i "/\[consensus\]/a create_empty_blocks = true" "$CONFIG_FILE"
    fi
    
    # Устанавливаем create_empty_blocks_interval
    if grep -q "^create_empty_blocks_interval" "$CONFIG_FILE"; then
        sed -i 's|^create_empty_blocks_interval = .*|create_empty_blocks_interval = "0s"|' "$CONFIG_FILE"
    else
        sed -i "/create_empty_blocks = true/a create_empty_blocks_interval = \"0s\"" "$CONFIG_FILE"
    fi
    
    # КРИТИЧЕСКИ ВАЖНО: Настройка параметров для подключения ко всем пирам
    # persistent_peers_max_dial_period = "0s" - узлы пытаются подключиться сразу
    if grep -q "^persistent_peers_max_dial_period" "$CONFIG_FILE"; then
        sed -i 's|^persistent_peers_max_dial_period = .*|persistent_peers_max_dial_period = "0s"|' "$CONFIG_FILE"
    else
        sed -i "/\[p2p\]/a persistent_peers_max_dial_period = \"0s\"" "$CONFIG_FILE"
    fi
    
    # allow_duplicate_ip = true - разрешает несколько узлов на одном IP
    if grep -q "^allow_duplicate_ip" "$CONFIG_FILE"; then
        sed -i 's|^allow_duplicate_ip = .*|allow_duplicate_ip = true|' "$CONFIG_FILE"
    else
        sed -i "/\[p2p\]/a allow_duplicate_ip = true" "$CONFIG_FILE"
    fi
    
    # addr_book_strict = false - важно для локального тестнета (разрешает приватные IP)
    if grep -q "^addr_book_strict" "$CONFIG_FILE"; then
        sed -i 's|^addr_book_strict = .*|addr_book_strict = false|' "$CONFIG_FILE"
    else
        sed -i "/\[p2p\]/a addr_book_strict = false" "$CONFIG_FILE"
    fi
    echo "✅ addr_book_strict = false (важно для локального тестнета)"
    
    # max_num_outbound_peers = 20 - достаточно для подключения ко всем пирам
    if grep -q "^max_num_outbound_peers" "$CONFIG_FILE"; then
        sed -i 's|^max_num_outbound_peers = .*|max_num_outbound_peers = 20|' "$CONFIG_FILE"
    else
        sed -i "/\[p2p\]/a max_num_outbound_peers = 20" "$CONFIG_FILE"
    fi
    
    # Настройка unconditional_peer_ids из persistent_peers (если они есть)
    if grep -q "^persistent_peers" "$CONFIG_FILE"; then
        PERSISTENT_PEERS=$(grep "^persistent_peers" "$CONFIG_FILE" | sed 's/.*= "\(.*\)"/\1/')
        if [ -n "$PERSISTENT_PEERS" ]; then
            # Извлекаем Node IDs из persistent_peers (формат: node_id@address:port)
            UNCONDITIONAL_IDS=$(echo "$PERSISTENT_PEERS" | tr ',' '\n' | cut -d'@' -f1 | tr '\n' ',' | sed 's/,$//')
            if [ -n "$UNCONDITIONAL_IDS" ]; then
                if grep -q "^unconditional_peer_ids" "$CONFIG_FILE"; then
                    sed -i "s|^unconditional_peer_ids = .*|unconditional_peer_ids = \"$UNCONDITIONAL_IDS\"|" "$CONFIG_FILE"
                else
                    sed -i "/\[p2p\]/a unconditional_peer_ids = \"$UNCONDITIONAL_IDS\"" "$CONFIG_FILE"
                fi
                echo "✅ unconditional_peer_ids настроен из persistent_peers"
            fi
        fi
    fi
    
    # Настройка external_address
    # В host network mode, external_address должен быть localhost с правильным портом
    # В bridge network mode, external_address должен быть IP контейнера с правильным портом
    if [ -n "$VOLNIX_P2P_PORT" ]; then
        if [ "$MODE" = "decentralized" ]; then
            # Host network mode
            EXTERNAL_ADDR="127.0.0.1:$VOLNIX_P2P_PORT"
        else
            # Bridge network mode - получаем IP адрес контейнера
            CONTAINER_IP=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "")
            if [ -n "$CONTAINER_IP" ] && [ "$CONTAINER_IP" != "127.0.0.1" ]; then
                EXTERNAL_ADDR="$CONTAINER_IP:$VOLNIX_P2P_PORT"
            else
                # Fallback: пытаемся получить IP из Docker network
                CONTAINER_IP=$(ip route get 8.8.8.8 2>/dev/null | awk '{print $7; exit}' || echo "")
                if [ -n "$CONTAINER_IP" ]; then
                    EXTERNAL_ADDR="$CONTAINER_IP:$VOLNIX_P2P_PORT"
                else
                    # Последний fallback: используем localhost (не идеально, но лучше чем ничего)
                    EXTERNAL_ADDR="127.0.0.1:$VOLNIX_P2P_PORT"
                fi
            fi
        fi
        
        if grep -q "^external_address" "$CONFIG_FILE"; then
            sed -i "s|^external_address = .*|external_address = \"$EXTERNAL_ADDR\"|" "$CONFIG_FILE"
        else
            sed -i "/\[p2p\]/a external_address = \"$EXTERNAL_ADDR\"" "$CONFIG_FILE"
        fi
        echo "✅ external_address настроен: $EXTERNAL_ADDR"
    fi
fi

# Выполняем команду
echo "⚡ Запуск узла..."
echo ""

# Передаем все аргументы команде volnixd-standalone
exec volnixd-standalone "$@"

