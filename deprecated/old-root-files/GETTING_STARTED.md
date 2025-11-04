# 🚀 Volnix Protocol - Руководство по началу работы

## 📋 Обзор

Добро пожаловать в Volnix Protocol! Это руководство поможет вам быстро начать работу с нашим инновационным блокчейн-протоколом.

## ⚡ Быстрый старт

### 1. 📦 Получение бинарных файлов

Бинарные файлы уже собраны и находятся в директории `build/`:

```
build/
├── volnixd.exe     # Windows (58.7 MB)
└── volnixd         # Linux (58.5 MB)
```

### 2. 🔍 Проверка установки

```bash
# Windows
.\build\volnixd.exe version

# Linux/macOS
./build/volnixd version
```

Ожидаемый вывод:
```
🚀 Volnix Protocol
Version: 0.1.0-alpha
Commit: development
Built: 2025-01-30

🏗️  Built with:
   • Cosmos SDK v0.53.x
   • CometBFT v0.38.x
   • Go 1.23+

🌟 Features:
   • Hybrid PoVB Consensus
   • ZKP Identity Verification
   • Three-tier Economic Model
   • High Performance Architecture
```

### 3. 🏗️ Инициализация узла

```bash
# Инициализация с именем валидатора
.\build\volnixd.exe init MyValidator --chain-id volnix-testnet-1

# Проверка статуса
.\build\volnixd.exe status
```

### 4. 🚀 Запуск узла

```bash
# Запуск узла
.\build\volnixd.exe start
```

## 🛠️ Автоматическое развертывание

### Windows

```powershell
# Полное развертывание с мониторингом
.\scripts\deploy.ps1 -Moniker "MyValidator" -EnableMonitoring

# Только сборка
.\scripts\deploy.ps1 -BuildBinary

# Помощь
.\scripts\deploy.ps1 -Help
```

### Linux/macOS

```bash
# Полное развертывание
./scripts/deploy.sh --moniker "MyValidator" --enable-monitoring

# Только сборка
./scripts/deploy.sh --skip-build

# Помощь
./scripts/deploy.sh --help
```

## 🔧 Основные команды

### Управление узлом

```bash
# Инициализация
volnixd init <moniker> --chain-id <chain-id>

# Запуск
volnixd start

# Статус
volnixd status

# Версия
volnixd version
```

### Управление ключами

```bash
# Создать новый ключ
volnixd keys add mykey

# Список ключей
volnixd keys list

# Показать ключ
volnixd keys show mykey
```

### Конфигурация

```bash
# Показать конфигурацию
volnixd config show

# Установить параметр
volnixd config set network.chain_id volnix-1

# Сбросить к умолчаниям
volnixd config reset
```

### Валидаторы

```bash
# Список валидаторов
volnixd validator list

# Информация о валидаторе
volnixd validator info <validator-address>

# Сжечь токены для веса
volnixd validator burn <amount>

# Статистика валидаторов
volnixd validator stats
```

### Экономическая система

```bash
# Список ордеров
volnixd economic orders list

# Создать ордер
volnixd economic orders create LIMIT BUY 1000 1.5

# Список аукционов
volnixd economic auctions list

# Статистика
volnixd economic stats

# Информация о токенах
volnixd economic tokens
```

### Мониторинг

```bash
# Запустить мониторинг
volnixd monitoring start

# Остановить мониторинг
volnixd monitoring stop

# Статус мониторинга
volnixd monitoring status
```

## 🌐 Эндпоинты мониторинга

После запуска узла доступны следующие эндпоинты:

### HTTP API
- **Здоровье**: http://localhost:8080/health
- **Метрики**: http://localhost:8080/metrics
- **Статус**: http://localhost:8080/status
- **Консенсус**: http://localhost:8080/consensus
- **Экономика**: http://localhost:8080/economic
- **Идентификация**: http://localhost:8080/identity

### CometBFT RPC
- **RPC**: http://localhost:26657
- **P2P**: tcp://localhost:26656
- **API**: http://localhost:1317

## 📊 Примеры использования

### Создание валидатора

```bash
# 1. Создать ключ валидатора
volnixd keys add validator

# 2. Получить токены (из faucet или биржи)
# ...

# 3. Сжечь токены для получения веса
volnixd validator burn 10000

# 4. Создать валидатора
volnixd tx staking create-validator \
  --amount=1000000ant \
  --pubkey=$(volnixd tendermint show-validator) \
  --moniker="MyValidator" \
  --chain-id=volnix-1 \
  --from=validator
```

### Торговля на внутреннем рынке

```bash
# 1. Создать лимитный ордер на покупку
volnixd economic orders create LIMIT BUY 1000 1.50

# 2. Создать рыночный ордер на продажу
volnixd economic orders create MARKET SELL 500 0

# 3. Проверить статус ордеров
volnixd economic orders list

# 4. Посмотреть статистику торгов
volnixd economic stats
```

### Верификация личности

```bash
# 1. Создать аккаунт для верификации
volnixd tx ident create-account \
  --verification-hash="hash123" \
  --zk-proof="proof456" \
  --from=mykey

# 2. Проверить статус верификации
volnixd query ident account $(volnixd keys show mykey -a)

# 3. Мигрировать роль (если необходимо)
volnixd tx ident migrate-role \
  --new-role="trader" \
  --from=mykey
```

## 🔧 Конфигурация

### Основные параметры

Конфигурация находится в `~/.volnix/config/config.json`:

```json
{
  "network": {
    "chain_id": "volnix-1",
    "listen_address": "tcp://0.0.0.0:26656",
    "max_peers": 50
  },
  "consensus": {
    "algorithm": "PoVB",
    "block_time": "5s",
    "halving_interval": 210000
  },
  "economic": {
    "base_currency": "ANT",
    "trading_fee": 0.001,
    "min_order_amount": "0.001"
  },
  "monitoring": {
    "enabled": true,
    "port": "8080"
  }
}
```

### Настройка для testnet

```bash
# Установить chain ID для testnet
volnixd config set network.chain_id volnix-testnet-1

# Уменьшить время блока для тестирования
volnixd config set consensus.block_time 3s

# Включить отладочное логирование
volnixd config set logging.level debug
```

### Настройка для mainnet

```bash
# Установить chain ID для mainnet
volnixd config set network.chain_id volnix-1

# Установить производственные параметры
volnixd config set consensus.block_time 6s
volnixd config set economic.trading_fee 0.002
volnixd config set logging.level warn
```

## 🧪 Тестирование

### Запуск тестов

```bash
# Все тесты
go test ./...

# Тесты производительности
go test ./tests -v -run TestSimple

# Benchmark тесты
go test ./tests -v -run BenchmarkTestSuite
```

### Локальная сеть

```bash
# Запуск локальной тестовой сети
volnixd testnet --v 4 --output-dir ./testnet

# Запуск узлов
cd testnet/node0 && volnixd start --home .
cd testnet/node1 && volnixd start --home .
# ... и так далее
```

## 📚 Дополнительные ресурсы

### Документация
- **Техническая документация**: `docs/`
- **Протокол**: `docs/volnix_protocol.md`
- **Актуальные отчеты**: `docs/reports/`
- **Структура проекта**: `PROJECT_STRUCTURE.md`

### Примеры
- **Конфигурации**: `examples/configs/`
- **Скрипты**: `scripts/`
- **Docker**: `Dockerfile`

### Сообщество
- **GitHub**: https://github.com/volnix-protocol/volnix-protocol
- **Discord**: https://discord.gg/volnix
- **Telegram**: https://t.me/volnixprotocol
- **Twitter**: https://twitter.com/volnixprotocol

## 🆘 Поддержка

### Часто задаваемые вопросы

**Q: Как получить токены для тестирования?**
A: Используйте faucet для testnet или обратитесь в сообщество.

**Q: Почему узел не синхронизируется?**
A: Проверьте подключение к интернету и настройки peers в конфигурации.

**Q: Как стать валидатором?**
A: Сожгите токены ANT для получения веса и создайте валидатора.

**Q: Где посмотреть логи?**
A: Логи находятся в `~/.volnix/volnix.log` или используйте `journalctl -u volnixd -f`.

### Получение помощи

1. **Документация**: Сначала проверьте документацию
2. **GitHub Issues**: Создайте issue для багов
3. **Discord**: Задайте вопрос в сообществе
4. **Email**: support@volnix.network

## 🎯 Следующие шаги

После успешного запуска узла:

1. **Присоединитесь к testnet** - подключитесь к тестовой сети
2. **Станьте валидатором** - помогите обеспечить безопасность сети
3. **Изучите торговлю** - попробуйте внутренний рынок ANT
4. **Участвуйте в управлении** - голосуйте с помощью LZN токенов
5. **Разрабатывайте** - создавайте приложения на Volnix Protocol

---

**🚀 Добро пожаловать в будущее децентрализованных финансов с Volnix Protocol!**

*Если у вас есть вопросы, не стесняйтесь обращаться к сообществу.*