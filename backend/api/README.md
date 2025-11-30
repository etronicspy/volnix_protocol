# Volnix REST API Server

REST API сервер для доступа к данным блокчейна Volnix Protocol через HTTP.

## Описание

Этот сервер предоставляет HTTP-интерфейс для запросов к блокчейну, подключаясь к gRPC серверу (порт 9090) и преобразуя gRPC ответы в JSON.

## Возможности

- ✅ Health check эндпоинт
- ✅ Consensus модуль:
  - `/volnix/consensus/v1/params` - параметры модуля
  - `/volnix/consensus/v1/validators` - список валидаторов
- ✅ Graceful shutdown
- ✅ Обработка ошибок gRPC → HTTP

## Установка

```bash
cd backend/api
go mod download
```

## Запуск

### Базовый запуск

```bash
go run main.go server.go
```

### С параметрами

```bash
go run main.go server.go \
  -grpc-addr=localhost:9090 \
  -http-addr=0.0.0.0:1317
```

### Сборка бинарника

```bash
go build -o volnix-rest-api main.go server.go
./volnix-rest-api
```

## Конфигурация

Параметры командной строки:

- `-grpc-addr` - адрес gRPC сервера (по умолчанию: `localhost:9090`)
- `-http-addr` - адрес HTTP сервера (по умолчанию: `0.0.0.0:1317`)

## Эндпоинты

### Health Check

```bash
curl http://localhost:1317/health
```

Ответ:
```json
{
  "status": "healthy",
  "service": "volnix-rest-api"
}
```

### Root

```bash
curl http://localhost:1317/
```

Ответ:
```json
{
  "service": "Volnix REST API",
  "version": "1.0.0",
  "endpoints": {
    "health": "/health",
    "consensus_params": "/volnix/consensus/v1/params",
    "consensus_validators": "/volnix/consensus/v1/validators"
  }
}
```

### Consensus Params

```bash
curl http://localhost:1317/volnix/consensus/v1/params
```

Ответ:
```json
{
  "params": {
    "base_block_time": "5s",
    "high_activity_threshold": 1000000,
    "low_activity_threshold": 100000,
    "min_burn_amount": "1000",
    "max_burn_amount": "1000000000"
  }
}
```

### Consensus Validators

```bash
curl http://localhost:1317/volnix/consensus/v1/validators
```

Ответ:
```json
{
  "validators": [
    {
      "validator": "cosmos1...",
      "ant_balance": "1000000",
      "status": "VALIDATOR_STATUS_ACTIVE",
      "activity_score": "500",
      "total_blocks_created": 10,
      "total_burn_amount": "5000000"
    }
  ]
}
```

## Требования

- Go 1.21+
- Запущенный блокчейн узел с gRPC сервером на порту 9090
- Сгенерированные protobuf файлы в `proto/gen/go/`

### Генерация Protobuf файлов

Перед запуском REST API сервера необходимо сгенерировать protobuf файлы:

```bash
# Из корня проекта
cd ../..  # Если вы в backend/api
buf generate proto/volnix
```

Или используйте команду из корня проекта (если настроена):
```bash
make proto-gen  # Если команда есть в Makefile
```

## Разработка

### Добавление новых эндпоинтов

1. Добавьте обработчик в `server.go`:
```go
func (s *Server) newHandler(w http.ResponseWriter, r *http.Request) {
    // Ваша логика
}
```

2. Зарегистрируйте маршрут в `SetupRoutes`:
```go
mux.HandleFunc("/new/endpoint", s.newHandler)
```

### Тестирование

```bash
# Проверка health
curl http://localhost:1317/health

# Проверка валидаторов
curl http://localhost:1317/volnix/consensus/v1/validators | jq

# Проверка параметров
curl http://localhost:1317/volnix/consensus/v1/params | jq
```

## Troubleshooting

### Ошибка подключения к gRPC

```
Failed to connect to gRPC server: connection refused
```

**Решение**: Убедитесь, что блокчейн узел запущен и gRPC сервер доступен на порту 9090.

### Ошибка импорта protobuf

```
cannot find package "github.com/volnix-protocol/volnix-protocol/proto/gen/go/volnix/consensus/v1"
```

**Решение**: Убедитесь, что protobuf файлы сгенерированы:
```bash
# Из корня проекта
cd ../..  # Если вы в backend/api
buf generate proto/volnix

# Или если установлен buf
cd ../..
buf generate
```

Проверьте, что файлы созданы:
```bash
ls proto/gen/go/volnix/consensus/v1/
```

## Интеграция с Docker

Пример Dockerfile:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY backend/api/ .
RUN go mod download
RUN go build -o volnix-rest-api main.go server.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/volnix-rest-api .
CMD ["./volnix-rest-api"]
```

## Лицензия

Часть проекта Volnix Protocol.

