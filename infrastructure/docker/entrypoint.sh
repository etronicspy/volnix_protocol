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
if [ -n "$PERSISTENT_PEERS" ]; then
    echo "🔗 Настройка persistent peers..."
    CONFIG_FILE="$VOLNIX_HOME/config/config.toml"
    
    # Обновляем persistent_peers в config.toml
    if grep -q "^persistent_peers" "$CONFIG_FILE"; then
        # Обновляем существующую строку
        sed -i "s|^persistent_peers = \".*\"|persistent_peers = \"$PERSISTENT_PEERS\"|" "$CONFIG_FILE"
    else
        # Добавляем новую строку после [p2p]
        sed -i "/\[p2p\]/a persistent_peers = \"$PERSISTENT_PEERS\"" "$CONFIG_FILE"
    fi
    
    echo "✅ Persistent peers настроены: $PERSISTENT_PEERS"
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
fi

# Выполняем команду
echo "⚡ Запуск узла..."
echo ""

# Передаем все аргументы команде volnixd-standalone
exec volnixd-standalone "$@"

