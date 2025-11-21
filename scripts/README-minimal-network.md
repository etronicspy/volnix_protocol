# Запуск локальной сети для разработки Volnix Protocol

⚠️ **ВНИМАНИЕ:** Этот скрипт предназначен **ТОЛЬКО для локальной разработки и тестирования**.

**Для production используйте Docker** - каждый валидатор должен быть в отдельном Docker контейнере. Сеть формируется из множества независимых контейнеров, каждый на своем сервере.

Этот скрипт запускает несколько узлов на одной машине для разработки/тестирования.

## Требования

- Go 1.21+
- jq (для работы с JSON)
- curl (для проверки статуса)

### Установка jq

**macOS:**
```bash
brew install jq
```

**Linux (Ubuntu/Debian):**
```bash
sudo apt-get install jq
```

**Linux (Fedora/RHEL):**
```bash
sudo dnf install jq
```

## Использование

### Базовый запуск (3 узла по умолчанию)

```bash
./scripts/start-local-dev-network.sh
```

### Запуск с указанием количества узлов

```bash
# Запуск с 2 узлами (минимум)
./scripts/start-local-dev-network.sh 2

# Запуск с 4 узлами
./scripts/start-local-dev-network.sh 4

# Запуск с 5 узлами
./scripts/start-local-dev-network.sh 5
```

### Запуск с очисткой существующих данных

```bash
./scripts/start-local-dev-network.sh --clean
```

### Комбинированные опции

```bash
# Запуск с 4 узлами и очисткой данных
./scripts/start-local-dev-network.sh 4 --clean
```

## Что запускается

1. **Несколько узлов блокчейна:**
   - node0: P2P порт 26656, RPC порт 26657
   - node1: P2P порт 26666, RPC порт 26667
   - node2: P2P порт 26676, RPC порт 26677
   - nodeN: P2P порт 26656 + N*10, RPC порт 26657 + N*10

2. **Общий genesis файл:**
   - Все узлы используют один и тот же genesis файл
   - Все валидаторы включены в genesis
   - Chain ID: `volnix-testnet`

3. **Peer connections:**
   - Все узлы настроены для подключения друг к другу
   - Автоматическая настройка persistent_peers

## Структура

После запуска создаются следующие директории:

- `testnet/` - данные узлов (конфигурация, ключи, база данных)
  - `testnet/node0/` - данные первого узла
  - `testnet/node1/` - данные второго узла
  - `testnet/nodeN/` - данные N-го узла
  - `testnet/genesis.json` - общий genesis файл
- `logs/` - логи всех узлов
  - `logs/node0.log` - логи первого узла
  - `logs/node1.log` - логи второго узла
  - `logs/nodeN.log` - логи N-го узла
- `build/` - скомпилированный бинарник volnixd-standalone

## Остановка

Нажмите `Ctrl+C` для остановки всех процессов. Скрипт автоматически остановит:
- Все узлы блокчейна
- Все связанные процессы

## Просмотр логов

```bash
# Логи конкретного узла
tail -f logs/node0.log
tail -f logs/node1.log

# Логи всех узлов
tail -f logs/*.log
```

## Проверка статуса

### Проверка статуса через RPC

```bash
# Статус первого узла
curl http://localhost:26657/status | jq

# Статус второго узла
curl http://localhost:26667/status | jq

# Получение информации о блоке
curl http://localhost:26657/block?height=1 | jq

# Получение последнего блока
curl http://localhost:26657/block | jq
```

### Проверка синхронизации

```bash
# Проверка высоты блоков на всех узлах
for port in 26657 26667 26677; do
  echo "Node on port $port:"
  curl -s http://localhost:$port/status | jq -r '.result.sync_info.latest_block_height'
done
```

### Проверка валидаторов

```bash
# Список валидаторов
curl http://localhost:26657/validators | jq

# Информация о консенсусе
curl http://localhost:26657/consensus_state | jq
```

## Настройка параметров консенсуса

Скрипт автоматически настраивает быстрые параметры консенсуса для локальной сети:

- `timeout_propose = "1s"` (вместо 3s)
- `timeout_prevote = "500ms"` (вместо 1s)
- `timeout_precommit = "500ms"` (вместо 1s)
- `timeout_commit = "1s"` (вместо 5s)

Это позволяет блокам создаваться быстрее для тестирования.

## Минимальное количество узлов

Для работы консенсуса требуется минимум **2 узла**. Однако для более реалистичной симуляции рекомендуется использовать **3-4 узла**.

- **2 узла**: Минимальная конфигурация, работает, но менее устойчива к сбоям
- **3 узла**: Рекомендуемый минимум для тестирования
- **4 узла**: Оптимально для симуляции реальной сети (BFT консенсус требует >2/3 голосов)

## Устранение неполадок

### Узлы не запускаются

1. Проверьте, что порты не заняты:
```bash
lsof -i :26657
lsof -i :26667
```

2. Очистите данные и перезапустите:
```bash
./scripts/start-local-dev-network.sh --clean
```

### Узлы не синхронизируются

1. Проверьте логи на наличие ошибок:
```bash
tail -f logs/node0.log
```

2. Убедитесь, что все узлы используют один и тот же genesis файл:
```bash
diff testnet/node0/config/genesis.json testnet/node1/config/genesis.json
```

3. Проверьте peer connections в config.toml:
```bash
grep persistent_peers testnet/node0/config/config.toml
```

### Ошибки при создании genesis

1. Убедитесь, что jq установлен:
```bash
which jq
jq --version
```

2. Проверьте права доступа к файлам:
```bash
ls -la testnet/node0/config/
```

## Примеры использования

### Тестирование транзакций

После запуска сети вы можете отправлять транзакции через RPC:

```bash
# Пример отправки транзакции (требует настройки)
curl -X POST http://localhost:26657 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "broadcast_tx_sync",
    "params": {
      "tx": "..."
    },
    "id": 1
  }'
```

### Мониторинг сети

```bash
# Следить за созданием новых блоков
watch -n 1 'curl -s http://localhost:26657/status | jq -r ".result.sync_info.latest_block_height"'

# Мониторинг всех узлов
for port in 26657 26667 26677; do
  echo "Port $port: $(curl -s http://localhost:$port/status | jq -r '.result.sync_info.latest_block_height')"
done
```

## Дополнительная информация

- Все узлы используют один и тот же chain ID: `volnix-testnet`
- Каждый узел имеет свой уникальный валидатор ключ
- Все валидаторы имеют одинаковую мощность (power = 10)
- Peer connections настроены автоматически
- База данных очищается при использовании флага `--clean`

