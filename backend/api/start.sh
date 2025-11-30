#!/bin/bash

# Скрипт запуска REST API сервера

set -e

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}🚀 Запуск Volnix REST API сервера${NC}"
echo ""

# Проверка protobuf файлов
if [ ! -d "../../proto/gen/go/volnix/consensus/v1" ]; then
    echo -e "${YELLOW}⚠️  Protobuf файлы не найдены${NC}"
    echo "Генерация protobuf файлов..."
    cd ../..
    if command -v buf &> /dev/null; then
        buf generate proto/volnix || echo -e "${RED}❌ Ошибка генерации protobuf${NC}"
    else
        echo -e "${RED}❌ buf не установлен. Установите: https://buf.build/docs/installation${NC}"
        exit 1
    fi
    cd backend/api
fi

# Проверка Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go не установлен${NC}"
    exit 1
fi

# Загрузка зависимостей
echo "Загрузка зависимостей..."
go mod download

# Сборка
echo "Сборка сервера..."
go build -o volnix-rest-api main.go server.go

# Запуск
echo -e "${GREEN}✅ Сервер готов к запуску${NC}"
echo ""
echo "Запуск с параметрами по умолчанию:"
echo "  gRPC: localhost:9090"
echo "  HTTP: 0.0.0.0:1317"
echo ""

./volnix-rest-api "$@"


