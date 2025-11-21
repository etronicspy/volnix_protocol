#!/bin/sh
# Health check скрипт для узла

RPC_PORT="${VOLNIX_RPC_PORT:-26657}"
RPC_URL="http://localhost:${RPC_PORT}"

# Проверяем доступность RPC endpoint
if curl -f -s "${RPC_URL}/status" > /dev/null 2>&1; then
    exit 0
fi

exit 1

