#!/bin/sh
# Утилита для получения информации о узле

VOLNIX_HOME="${VOLNIX_HOME:-/home/volnix/.volnix}"
RPC_PORT="${VOLNIX_RPC_PORT:-26657}"

echo "📊 Информация о узле"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Информация из конфигурации
if [ -f "$VOLNIX_HOME/config/config.toml" ]; then
    echo "🔧 Конфигурация:"
    grep -E "^moniker|^chain_id" "$VOLNIX_HOME/config/config.toml" 2>/dev/null || true
    echo ""
fi

# Информация из genesis
if [ -f "$VOLNIX_HOME/config/genesis.json" ]; then
    if command -v jq >/dev/null 2>&1; then
        echo "🔗 Genesis:"
        echo "   Chain ID: $(jq -r '.chain_id' "$VOLNIX_HOME/config/genesis.json")"
        echo "   Validators: $(jq '.validators | length' "$VOLNIX_HOME/config/genesis.json")"
        echo ""
    fi
fi

# Статус через RPC
if curl -f -s "http://localhost:${RPC_PORT}/status" > /dev/null 2>&1; then
    echo "⚡ Статус узла (RPC):"
    curl -s "http://localhost:${RPC_PORT}/status" | jq -r '.result.node_info | "   ID: \(.id)\n   Moniker: \(.moniker)\n   Network: \(.network)"' 2>/dev/null || echo "   Узел работает"
    echo ""
    
    echo "📦 Блокчейн:"
    curl -s "http://localhost:${RPC_PORT}/status" | jq -r '.result.sync_info | "   Latest Block: \(.latest_block_height)\n   Latest Block Time: \(.latest_block_time)"' 2>/dev/null || echo "   Информация недоступна"
else
    echo "⚠️  RPC недоступен (узел не запущен или порт неверный)"
fi

echo ""

