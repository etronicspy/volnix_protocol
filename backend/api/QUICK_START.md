# Быстрый старт REST API сервера

## Шаг 1: Генерация Protobuf файлов

```bash
# Из корня проекта
cd ../..
buf generate proto/volnix
```

Если `buf` не установлен:
```bash
# macOS
brew install bufbuild/buf/buf

# Linux
curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o "/usr/local/bin/buf"
chmod +x "/usr/local/bin/buf"
```

## Шаг 2: Загрузка зависимостей

```bash
cd backend/api
go mod download
```

## Шаг 3: Сборка

```bash
go build -o volnix-rest-api main.go server.go
```

## Шаг 4: Запуск

### Базовый запуск
```bash
./volnix-rest-api
```

### С параметрами
```bash
./volnix-rest-api -grpc-addr=localhost:9090 -http-addr=0.0.0.0:1317
```

### Или через скрипт
```bash
./start.sh
```

## Шаг 5: Проверка

```bash
# Health check
curl http://localhost:1317/health

# Валидаторы
curl http://localhost:1317/volnix/consensus/v1/validators

# Параметры
curl http://localhost:1317/volnix/consensus/v1/params
```

## Troubleshooting

### Ошибка: "cannot find package"
- Убедитесь, что protobuf файлы сгенерированы
- Проверьте путь: `proto/gen/go/volnix/consensus/v1/`

### Ошибка: "connection refused"
- Убедитесь, что блокчейн узел запущен
- Проверьте, что gRPC сервер доступен на порту 9090

### Ошибка: "module not found"
- Выполните: `go mod download`
- Проверьте, что вы в правильной директории


