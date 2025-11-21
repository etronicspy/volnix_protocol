# Скрипты Volnix Protocol

## Основные скрипты запуска

### Linux/macOS

#### `start-local-dev-network.sh` ⚠️ ТОЛЬКО для локальной разработки
**ВНИМАНИЕ:** Этот скрипт предназначен **ТОЛЬКО для локальной разработки и тестирования**.

Для production используйте **Docker** - каждый валидатор должен быть в отдельном контейнере.

Скрипт запускает несколько узлов на одной машине для разработки/тестирования.

```bash
# Запуск с 3 узлами (по умолчанию)
./scripts/start-local-dev-network.sh

# Запуск с указанным количеством узлов
./scripts/start-local-dev-network.sh 2
./scripts/start-local-dev-network.sh 4

# Запуск с очисткой данных
./scripts/start-local-dev-network.sh --clean

# Добавление нового узла к существующей сети
./scripts/start-local-dev-network.sh add 3
```

**Документация:** [README-minimal-network.md](README-minimal-network.md)

#### `deploy.sh`
Скрипт развертывания для Linux/macOS.

```bash
./scripts/deploy.sh --moniker "MyValidator" --enable-monitoring
```

### Windows

#### `quick-start.ps1`
Быстрый старт для Windows.

```powershell
.\scripts\quick-start.ps1
```

#### `start-full-stack.ps1`
Запуск полного стека для Windows.

```powershell
.\scripts\start-full-stack.ps1
```

## Утилиты

### `generate-validator-keys.py`
Генерация ключей валидатора.

```bash
python3 scripts/generate-validator-keys.py [testnet_dir] [num_nodes]
```

### Добавление узла к сети
Добавление нового узла к существующей сети (интегрировано в start-local-dev-network.sh).

```bash
./scripts/start-local-dev-network.sh add <номер_узла>
```

### `monitor-transactions.sh`
Мониторинг транзакций в сети.

```bash
./scripts/monitor-transactions.sh
```

## Тестирование

### `test-e2e-transactions.sh`
End-to-end тесты транзакций.

```bash
./scripts/test-e2e-transactions.sh
```

### `test-multinode.sh`
Тесты мультинод сети.

```bash
./scripts/test-multinode.sh
```

### `testing/volnix_tests.go`
Go тесты для скриптов.

## Документация

- [QUICK_START.md](QUICK_START.md) - Быстрый старт
- [README-minimal-network.md](README-minimal-network.md) - Документация по минимальной сети
- [README-local-network.md](README-local-network.md) - Документация по локальной сети

## ⚠️ Важное различие: Локальная разработка vs Production

### Локальная разработка (скрипты)
- `start-local-dev-network.sh` - запускает несколько узлов на одной машине (только для разработки)
- Используется для **разработки и тестирования**
- Все узлы на одной машине
- Не подходит для production

### Production (Docker)
- **Каждый валидатор = отдельный Docker контейнер**
- Контейнеры могут быть на **разных серверах**
- Сеть формируется из **множества независимых контейнеров**
- Используйте `docker-compose.yml` для тестирования или развертывайте контейнеры отдельно

## Примечания

- ⚠️ **Для production используйте Docker** - каждый валидатор в отдельном контейнере
- `start-local-dev-network.sh` - **только для локальной разработки**
- `docker-compose.yml` - пример минимальной конфигурации (3 валидатора для тестирования)
- В реальной production сети может быть десятки/сотни валидаторов, каждый в своем контейнере

