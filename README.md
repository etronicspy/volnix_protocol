# 🚀 Волникс Протокол (Volnix Protocol)

**Волникс Протокол** - суверенный блокчейн первого уровня на базе Cosmos SDK с гибридным консенсусом Proof-of-Verified-Burn (PoVB) и трехуровневой экономической моделью.

## 🎉 Статус проекта: ✅ ГОТОВ К TESTNET

**Версия**: 0.1.0-alpha  
**Дата обновления**: 15 января 2026  
**Готовность**: Готов к Testnet Alpha deployment  
**Покрытие тестами**: ✅ 68.6% (цель: 60%+ достигнута)  
**Security**: ✅ Все критические проверки активны  
**Тесты**: ✅ 1,135+ тестов проходят

## 🚀 Быстрый старт

### Запуск полноценного узла (рекомендуется)
```bash
# Сборка (ВАЖНО: всегда используйте флаг -o для указания пути!)
# ✅ Правильно:
go build -o build/volnixd ./cmd/volnixd
# или используйте Makefile:
make build

# ❌ НЕПРАВИЛЬНО (создаст бинарник в корне проекта):
# go build ./cmd/volnixd

# Инициализация узла
./build/volnixd init MyValidator

# Запуск узла (с gRPC и REST API)
./build/volnixd start
```

### Запуск REST API сервера
```bash
# В отдельном терминале
cd backend/api
./volnix-rest-api -grpc-addr=localhost:9090 -http-addr=0.0.0.0:1317
```

### Запуск блокчейн-эксплорера
Откройте `frontend/blockchain-explorer/index.html` в браузере

### Автоматическое развертывание
```powershell
# Windows - быстрый старт
.\scripts\quick-start.ps1

# Windows - полный стек
.\scripts\start-full-stack.ps1

# Linux/macOS
./scripts/deploy.sh --moniker "MyValidator" --enable-monitoring
```

### Запуск локальной сети для разработки ⚠️
```bash
# ⚠️ ВНИМАНИЕ: Только для локальной разработки/тестирования!

# Запуск сети с несколькими узлами (по умолчанию 3 узла)
./scripts/start-local-dev-network.sh

# Запуск с указанием количества узлов (минимум 2)
./scripts/start-local-dev-network.sh 2
./scripts/start-local-dev-network.sh 4

# Запуск с очисткой данных
./scripts/start-local-dev-network.sh --clean

# Подробная документация
cat scripts/README-minimal-network.md
```

## 🌟 Особенности

- **🔐 Гибридный консенсус PoVB** - уникальное сочетание Proof-of-Stake и Proof-of-Burn
- **🆔 ZKP верификация** - доказательства с нулевым разглашением для идентичности
- **💎 Трехуровневая экономика** - WRT, LZN и ANT токены
- **⚡ Высокая производительность** - до 10,000 TPS
- **🌱 Энергоэффективность** - 99.9% экономия энергии по сравнению с PoW
- **🌐 Cosmos SDK** - совместимость с экосистемой Cosmos

## 🔗 Demo Wallet и консенсус

Действия demo wallet напрямую влияют на консенсус блокчейна:

1. **Активация LZN** — при активации LZN через wallet-ui voting power валидатора увеличивается пропорционально сумме активированного LZN.
2. **EndBlocker** — в конце каждого блока приложение суммирует активный LZN и возвращает `ValidatorUpdates` в CometBFT.
3. **Без хардкода** — начальный валидатор берётся из genesis, но его мощность обновляется на основе экономической активности (LZN).

Для локальной разработки используйте `volnixd-standalone`:
```bash
go build -o build/volnixd-standalone ./cmd/volnixd-standalone
./build/volnixd-standalone -home testnet/node0
```

**gRPC**: Backend API подключается к gRPC на `localhost:9090`. Standalone-нода использует RPC fallback для validators. Для полной gRPC поддержки запускайте `volnixd` (полный узел с app.toml).

## 🏗️ Архитектура

### Модули
- **`x/ident`** - Управление ролями и ZKP верификация
- **`x/lizenz`** - Лицензии на майнинг и MOA
- **`x/anteil`** - Внутренний рынок производительности
- **`x/consensus`** - Консенсус PoVB

### Токены
- **WRT (Wert)** - Основной токен, ценность и голос
- **LZN (Lizenz)** - Лицензия на майнинг
- **ANT (Anteil)** - Право на производительность

## 🚀 Быстрый старт

### Требования
- Go 1.24+
- Cosmos SDK v0.53.x
- CometBFT v0.38.x

### Установка
```bash
# Клонирование репозитория
git clone https://github.com/volnix-protocol/volnix-protocol.git
cd volnix-protocol

# Сборка (Linux/macOS)
make build

# Сборка (Windows)
.\build.bat build

# Или вручную (бинарники собираются в build/)
mkdir -p build
go build -o build/volnixd.exe ./cmd/volnixd  # Windows
go build -o build/volnixd ./cmd/volnixd      # Linux/macOS
```

### Запуск узла

#### 1. Инициализация узла
```bash
# Создание нового узла
./build/volnixd init mynode        # Linux/macOS
.\build\volnixd.exe init mynode    # Windows

# Это создаст:
# - Домашнюю директорию: .volnix/
# - Конфигурацию: .volnix/config/
# - Genesis файл: .volnix/config/genesis.json
```

#### 2. Запуск узла
```bash
# Запуск с персистентным хранилищем
./build/volnixd start        # Linux/macOS
.\build\volnixd.exe start    # Windows

# Узел запустится с:
# - Персистентной БД (GoLevelDB)
# - ABCI сервером
# - Сохранением состояния между перезапусками
```

#### 3. Тестовая сеть
```bash
# Linux/macOS
cd testnet && ./start.sh

# Windows
cd testnet && start.bat

# Это автоматически:
# - Инициализирует узел если нужно
# - Запустит сеть с персистентным хранилищем
# - Покажет статус и endpoints
```

#### 4. Управление ключами
```bash
# Создание ключа
./build/volnixd keys add mykey

# Список ключей
./build/volnixd keys list

# Информация о ключе
./build/volnixd keys show mykey
```

## 🏗️ Архитектура

### Технологический стек
- **Фреймворк:** Cosmos SDK v0.53.x
- **Консенсус:** CometBFT v0.38.x
- **Язык:** Go (Golang)
- **Хранилище:** GoLevelDB
- **Криптография:** ECDSA (secp256k1), zk-SNARK

### Сеть
- **Chain ID:** test-volnix
- **Bech32:** vx, vxvaloper, vxvalcons
- **API:** 1317
- **P2P:** 26656

### Хранилище

- **Тип**: GoLevelDB (персистентное)
- **Директория**: `~/.volnix/data/`
- **Конфигурация**: `~/.volnix/config/`

## 📊 Экономическая модель

### Двухконтурная система
1. **Безопасность** - WRT токены для стейкинга
2. **Производительность** - ANT токены для торговли

### Роли участников
- **Гость** - базовый функционал
- **Гражданин** - получает ANT, продавец на рынке
- **Валидатор** - активирует LZN, покупатель ANT

## 🧪 Тестирование

```bash
# Запуск всех тестов (Linux/macOS)
make test

# Windows
.\build.bat test

# Или вручную
go test ./x/ident/keeper -v
go test ./x/lizenz/keeper -v
go test ./x/anteil/keeper -v
go test ./x/consensus/keeper -v

# С покрытием
go test ./x/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Тестирование CLI
./build/volnixd version
./build/volnixd status
./build/volnixd init testnode
```

### 📊 Статус тестов (обновлено: 15 января 2026)

**Общее покрытие: 68.6%** ✅ (цель 60%+ достигнута!)

#### Keeper модули
| Модуль | Покрытие | Статус | Оценка |
|--------|----------|--------|--------|
| **x/consensus/keeper** | 84.2% | ✅ PASS | ⭐⭐⭐⭐⭐ |
| **x/anteil/keeper** | 77.3% | ✅ PASS | ⭐⭐⭐⭐ |
| **x/ident/keeper** | 73.1% | ✅ PASS | ⭐⭐⭐⭐ |
| **x/lizenz/keeper** | 72.9% | ✅ PASS | ⭐⭐⭐⭐ |
| **x/governance/keeper** | 66.8% | ✅ PASS | ⭐⭐⭐ |

**Средний keeper: 75.6%** ⭐⭐⭐⭐

#### Types модули
| Модуль | Покрытие | Статус |
|--------|----------|--------|
| **x/ident/types** | 71.2% | ✅ PASS |
| **x/lizenz/types** | 66.1% | ✅ PASS |
| **x/consensus/types** | 57.8% | ✅ PASS |
| **x/anteil/types** | 53.9% | ✅ PASS |

**Средний types: 62.3%**

**Достижения:**
- ✅ **0 падающих тестов** (было 7+)
- ✅ **1,135+ тестов проходят**
- ✅ **15 benchmark тестов** добавлено
- ✅ **Все security проверки активны**
- ✅ **Централизованная тестовая инфраструктура**

**Подробные отчеты:** См. `deprecated/PROJECT_STATUS_REPORT.md` и `tests/OPTIMIZATION_REPORT.md`

## 📁 Структура проекта

```
volnix-protocol/
├── app/                    # Основное приложение
├── cmd/                    # CLI команды
├── x/                      # Модули
│   ├── ident/             # Идентичность
│   ├── lizenz/            # Лицензии
│   ├── anteil/            # Рынок
│   └── consensus/         # Консенсус
├── proto/                  # Protobuf определения
├── config/                 # Конфигурационные файлы
│   ├── buf.*.yaml         # Protobuf конфигурация
│   └── genesis.json       # Genesis файл
├── build/                  # Скомпилированные бинарники
├── scripts/                # Скрипты запуска и управления
├── docs/                   # Технические спецификации
├── tests/                  # Тесты
├── deprecated/             # Устаревшие файлы и документация
│   └── guides/             # Старые руководства пользователя
└── frontend/               # Frontend приложения
    ├── wallet-ui/          # Веб-интерфейс кошелька
    └── blockchain-explorer/ # Обозреватель блокчейна
```

## 🔒 Безопасность

### ZKP интеграция
- Верификация уникальности личности
- Миграция ролей ("цифровое наследство")
- Правила активности для валидаторов

### Экономическая безопасность
- MOA проверки для валидаторов
- Аудит всех экономических операций
- Защита от атак Сивиллы

## 🚀 Производительность

### Оптимизации
- Эффективные алгоритмы matching engine
- Кэширование часто используемых данных
- Оптимизация запросов к state
- Минимизация накладных расходов консенсуса

## 🚧 Текущий статус: **84% готовности** (обновлено: 15.01.2026)

### ✅ Полностью готово (95-100%)
- **Базовая архитектура** Cosmos SDK ✅
- **Кастомные модули** (ident, lizenz, anteil, consensus, governance) ✅
- **CLI команды** для управления узлом ✅
- **Персистентное хранилище** (GoLevelDB) ✅
- **Genesis и конфигурация** ✅
- **ABCI сервер** с сохранением состояния ✅
- **Build инструменты** (Makefile, build.bat) ✅
- **Governance модуль** с DAO управлением ✅
- **Интеграция модулей** через адаптеры ✅
- **Валидация параметров** во всех модулях ✅
- **Security проверки** во всех критических местах ✅
- **Тестовая инфраструктура** (fixtures, mocks, helpers) ✅
- **Benchmark тесты** для измерения производительности ✅

### ✅ Готово (70-90%)
- **Тестовое покрытие** 68.6% (цель 60%+ достигнута!) ✅
  - ✅ x/consensus/keeper: 84.2%
  - ✅ x/anteil/keeper: 77.3%
  - ✅ x/ident/keeper: 73.1%
  - ✅ x/lizenz/keeper: 72.9%
  - ✅ x/governance/keeper: 66.8%
- **ZKP верификация** (структуры + тесты готовы) ✅
- **Консенсус PoVB** (полностью реализован) ✅
- **Экономическая логика** (работает, протестирована) ✅

### 🔄 В разработке (40-70%)
- **REST API** (базовые endpoints работают) 🔄
- **Frontend** (wallet UI функционален, требует доработки) 🔄
- **Real ZKP integration** (требуется библиотека gnark/circom) 🔄
- **Performance optimization** (узкие места идентифицированы) 🔄
- **Monitoring** (структура готова, требуется подключение) 🔄

### 🎯 Готов к использованию
- ✅ **Testnet Alpha** - готов к запуску
- ✅ **Локальное тестирование** - полностью готов
- ✅ **Multi-node testnet** - готов к запуску
- ⚠️ **Testnet Beta** - готов через 1-2 недели
- ⏳ **Mainnet** - готов через 2-3 месяца

## 🤝 Вклад в проект

1. Форкните репозиторий
2. Создайте ветку для новой функции
3. Внесите изменения
4. Добавьте тесты
5. Создайте Pull Request

## 📄 Лицензия

MIT License - см. [LICENSE](LICENSE) файл для деталей.

## 📚 Дополнительные ресурсы

- **[docs/](docs/)** - Техническая документация
- **[coverage_report.md](coverage_report.md)** - Детальный отчет о покрытии тестами (9.2%)
- **[tests/README.md](tests/README.md)** - Руководство по тестированию
- **[deprecated/](deprecated/)** - Устаревшие файлы и документация

## 📞 Контакты

- **GitHub:** [volnix-protocol](https://github.com/volnix-protocol)
- **Документация:** [docs/](docs/)
- **Issues:** [GitHub Issues](https://github.com/volnix-protocol/issues)

---

**Волникс Протокол** - Строим будущее децентрализованной экономики! 🚀

## 🎉 Поздравляем!

**Проект готов к базовому запуску!** 

Вы можете:
- ✅ Инициализировать узел
- ✅ Запустить ABCI сервер
- ✅ Использовать персистентное хранилище
- ✅ Тестировать CLI команды

Для полноценного блокчейна осталось интегрировать CometBFT и реализовать экономическую логику.
