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

# Проверяем, инициализирован ли узел
if [ ! -f "$VOLNIX_HOME/config/genesis.json" ]; then
    echo "📦 Узел не инициализирован. Выполняю инициализацию..."
    echo ""
    
    # Создаем директории
    mkdir -p "$VOLNIX_HOME/config"
    mkdir -p "$VOLNIX_HOME/data"
    mkdir -p "$VOLNIX_HOME/keyring-test"
    
    # Инициализируем узел
    volnixd-standalone init "$MONIKER" || {
        echo "❌ Ошибка инициализации узла"
        exit 1
    }
    
    echo "✅ Узел успешно инициализирован"
    echo ""
    
    # Обновляем chain_id в genesis.json если указан
    if [ "$CHAIN_ID" != "volnix-standalone" ]; then
        if command -v jq >/dev/null 2>&1; then
            jq ".chain_id = \"$CHAIN_ID\"" "$VOLNIX_HOME/config/genesis.json" > "$VOLNIX_HOME/config/genesis.json.tmp" && \
            mv "$VOLNIX_HOME/config/genesis.json.tmp" "$VOLNIX_HOME/config/genesis.json"
            echo "✅ Chain ID обновлен: $CHAIN_ID"
        fi
    fi
else
    echo "✅ Узел уже инициализирован"
    echo ""
fi

# Проверяем конфигурацию
if [ ! -f "$VOLNIX_HOME/config/config.toml" ]; then
    echo "⚠️  Конфигурационный файл отсутствует"
    echo "   Это не должно происходить после инициализации"
    exit 1
fi

# Выполняем команду
echo "⚡ Запуск узла..."
echo ""

# Передаем все аргументы команде volnixd-standalone
exec volnixd-standalone "$@"

