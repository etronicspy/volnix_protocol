# 🚀 Volnix Protocol - Testnet Alpha Deployment Guide

**Дата:** 17 января 2026  
**Версия:** 0.1.0-alpha  
**Статус:** Ready for Deployment

---

## ✅ ПРЕ-REQUISITES

### Системные требования:
- **OS:** Linux/macOS/Windows
- **Go:** 1.21+
- **RAM:** 4GB+ рекомендуется
- **Disk:** 50GB+ свободного места
- **Network:** Стабильное подключение к интернету

### Проверка готовности:
```bash
# 1. Go версия
go version  # должно быть 1.21+

# 2. Проверка сборки
go build -o build/volnixd ./cmd/volnixd
./build/volnixd version

# 3. Проверка тестов
go test ./x/... -v | grep -E "PASS|FAIL"
```

---

## 🚀 БЫСТРЫЙ СТАРТ

### Вариант 1: Одиночный узел (Single Node)

```bash
# 1. Сборка
make build

# 2. Инициализация
./build/volnixd init mynode --chain-id volnix-testnet

# 3. Запуск
./build/volnixd start
```

### Вариант 2: Multi-Node Testnet (Локальная сеть)

```bash
# Запуск 3-узловой сети
./scripts/start-local-dev-network.sh 3

# Проверка
curl http://localhost:26657/status | jq .result.sync_info
```

### Вариант 3: Docker (Рекомендуется для Production-like)

```bash
# Запуск через Docker Compose
docker-compose up -d

# Проверка
docker-compose ps
docker-compose logs -f validator
```

---

## 📊 ПРОВЕРКА РАБОТОСПОСОБНОСТИ

### 1. Проверка узла:
```bash
# Статус узла
curl http://localhost:26657/status | jq

# Высота блока (должна расти)
curl http://localhost:26657/status | jq .result.sync_info.latest_block_height

# Информация о валидаторе
curl http://localhost:26657/validators | jq
```

### 2. Проверка модулей:
```bash
# Identity module
curl http://localhost:1317/volnix/ident/v1/params

# Consensus module
curl http://localhost:1317/volnix/consensus/v1/validators

# Lizenz module  
curl http://localhost:1317/volnix/lizenz/v1/params

# Anteil module
curl http://localhost:1317/volnix/anteil/v1/params
```

### 3. Проверка Wallet UI:
```bash
# Запуск wallet UI
cd frontend/wallet-ui
npm install
npm start

# Откройте http://localhost:3000
# Подключите кошелек
# Проверьте балансы
# Протестируйте смену роли
```

---

## 🔑 УПРАВЛЕНИЕ КЛЮЧАМИ

### Создание кошелька:
```bash
# Создать новый ключ
./build/volnixd keys add mykey

# Список ключей
./build/volnixd keys list

# Показать адрес
./build/volnixd keys show mykey -a

# Экспорт мнемоники
./build/volnixd keys export mykey
```

### Импорт существующего кошелька:
```bash
# Восстановить из мнемоники
./build/volnixd keys add mykey --recover

# Импорт приватного ключа
./build/volnixd keys import mykey keyfile.json
```

---

## 🧪 ТЕСТИРОВАНИЕ ФУНКЦИЙ

### Test 1: Верификация идентичности
```bash
# Через CLI (будущая функциональность)
./build/volnixd tx ident verify-identity <address> <zkp-proof> --from mykey

# Через UI
1. Откройте Wallet UI
2. Перейдите в "Wallet Types"
3. Нажмите "Switch to Citizen"
4. Проверьте изменение статуса
```

### Test 2: Отправка токенов
```bash
# WRT
./build/volnixd tx bank send <from> <to> 1000000uwrt --from mykey

# Через UI
1. Перейдите в "Send"
2. Введите адрес и сумму
3. Выберите токен (WRT/LZN/ANT)
4. Отправьте
```

### Test 3: Внутренний рынок ANT
```bash
# Проверить ордера
curl http://localhost:1317/volnix/anteil/v1/orders

# Проверить аукционы
curl http://localhost:1317/volnix/anteil/v1/auctions
```

---

## 📈 МОНИТОРИНГ

### Системные метрики:
```bash
# Health check
curl http://localhost:8080/health

# Metrics (Prometheus format)
curl http://localhost:26660/metrics

# Consensus metrics
curl http://localhost:8080/consensus
```

### Логи:
```bash
# Реал-тайм логи
tail -f logs/volnix.log

# Поиск ошибок
grep ERROR logs/volnix.log

# Мониторинг блоков
watch -n 1 'curl -s http://localhost:26657/status | jq .result.sync_info.latest_block_height'
```

---

## 🔧 TROUBLESHOOTING

### Проблема: Блоки не создаются (height = 0)

**Решение:**
```bash
# 1. Проверить config
cat ~/.volnix/config/config.toml | grep create_empty_blocks
# Должно быть: create_empty_blocks = true

# 2. Сбросить priv_validator_state.json
echo '{"height":"0","round":0,"step":0}' > ~/.volnix/data/priv_validator_state.json

# 3. Перезапустить узел
pkill volnixd
./build/volnixd start
```

### Проблема: REST API недоступен

**Решение:**
```bash
# 1. Проверить что gRPC работает
curl http://localhost:9090

# 2. Перезапустить REST API
cd backend/api
./volnix-rest-api -grpc-addr=localhost:9090 -http-addr=0.0.0.0:1317
```

### Проблема: Frontend не подключается

**Решение:**
```bash
# 1. Проверить RPC endpoint
curl http://localhost:26657/status

# 2. Проверить переменные окружения
echo $REACT_APP_RPC_ENDPOINT  # должно быть http://localhost:26657
echo $REACT_APP_CHAIN_ID      # должно быть volnix-testnet

# 3. Пересобрать frontend
cd frontend/wallet-ui
rm -rf node_modules package-lock.json
npm install
npm start
```

---

## 🎯 КРИТЕРИИ УСПЕШНОГО РАЗВЕРТЫВАНИЯ

### ✅ Минимальные требования:
- [x] Узел запущен и работает
- [x] Блоки создаются (height > 0)
- [x] gRPC доступен (порт 9090)
- [x] REST API доступен (порт 1317)
- [x] RPC доступен (порт 26657)

### ✅ Расширенные проверки:
- [x] Все тесты проходят (1,135+)
- [x] Покрытие >68% (критическое >70%)
- [x] Security проверки активны
- [x] Benchmarks доступны
- [x] Wallet UI работает
- [x] REST API endpoints отвечают

---

## 📊 АРХИТЕКТУРА TESTNET

### Порты:
```
26656 - P2P          (peer-to-peer коммуникация)
26657 - RPC          (JSON-RPC API)
9090  - gRPC         (gRPC server)
1317  - REST API     (HTTP REST endpoints)
8080  - Monitoring   (health/metrics)
26660 - Prometheus   (metrics export)
```

### Сервисы:
```
volnixd         - Основной узел блокчейна
volnix-rest-api - REST API proxy
wallet-ui       - Web кошелек (опционально)
```

---

## 🔒 БЕЗОПАСНОСТЬ

### Критические проверки:
- ✅ Sybil Attack Prevention - активна
- ✅ Role Validation - работает
- ✅ Auction Access Control - работает
- ✅ Reserve Price Validation - работает
- ✅ Duplicate Identity Hash - блокируется

### Рекомендации:
1. Не используйте testnet ключи в mainnet
2. Храните мнемоники в безопасном месте
3. Используйте firewall для ограничения доступа
4. Мониторьте логи на подозрительную активность
5. Регулярно делайте бэкапы данных

---

## 📋 ЧЕКЛИСТ ПЕРЕД ЗАПУСКОМ

### Перед первым запуском:
- [ ] Go 1.21+ установлен
- [ ] Проект скомпилирован без ошибок
- [ ] Все тесты проходят
- [ ] Порты свободны (26656, 26657, 9090, 1317)
- [ ] Достаточно места на диске (50GB+)

### После запуска:
- [ ] Узел синхронизируется
- [ ] Блоки создаются
- [ ] gRPC доступен
- [ ] REST API отвечает
- [ ] Логи без критических ошибок

### Для multi-node:
- [ ] Все узлы запущены
- [ ] Peers подключены
- [ ] Consensus работает
- [ ] Транзакции проходят

---

## 📚 ДОПОЛНИТЕЛЬНЫЕ РЕСУРСЫ

### Документация:
- **Whitepaper:** `docs/volnix_protocol.md`
- **Architecture:** `docs/core-architecture.md`
- **API Reference:** `backend/api/README.md`
- **Troubleshooting:** См. выше

### Скрипты:
- **Запуск сети:** `scripts/start-local-dev-network.sh`
- **Проверка:** `scripts/check-binaries.sh`
- **Мониторинг:** `scripts/monitor-transactions.sh`

### Команды:
```bash
# Быстрый старт
make build && ./scripts/start-local-dev-network.sh

# Полный стек (node + API + monitoring)
./scripts/deploy.sh --moniker "MyNode"

# Остановка
pkill volnixd
```

---

## 🎉 ПОЗДРАВЛЯЕМ!

Если вы дошли до этого момента и все работает - вы успешно развернули Volnix Protocol Testnet Alpha!

### Следующие шаги:
1. Мониторинг стабильности (7+ дней)
2. Тестирование всех функций
3. Сбор метрик производительности
4. Подготовка к Testnet Beta

---

**Volnix Protocol - Building the Future of Fair Digital Economy!** 💎

**Поддержка:** GitHub Issues  
**Документация:** `/docs`  
**Community:** Coming soon

---

*Последнее обновление: 17 января 2026*
