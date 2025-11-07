# 🚀 Volnix Protocol - Руководство по быстрому запуску

> 📖 **Навигация**: Для быстрого доступа ко всем ресурсам проекта см. [NAVIGATION.md](../NAVIGATION.md)

## Обзор

Данное руководство поможет вам запустить полный функционал Volnix Protocol, включая:
- 🌐 Блокчейн узел с консенсусом PoVB
- 💰 Wallet UI для управления токенами
- 🔍 Blockchain Explorer для мониторинга сети
- 🔧 CLI инструменты для разработчиков

**Примечание**: Валидация личности отключена для демонстрации.

## Предварительные требования

### Обязательные зависимости:
- **Go 1.21+** - для сборки блокчейн узла
- **Node.js 18+** - для Wallet UI
- **npm** - для управления зависимостями
- **PowerShell** - для запуска скриптов (Windows)

### Проверка зависимостей:
```powershell
# Проверка Go
go version

# Проверка Node.js
node --version

# Проверка npm
npm --version
```

## Способы запуска

### 🚀 Способ 1: Автоматический запуск (Рекомендуется)

```powershell
# Полный запуск с проверками
powershell -ExecutionPolicy Bypass -File scripts/start-full-stack.ps1

# Быстрый запуск
powershell -ExecutionPolicy Bypass -File scripts/quick-start.ps1

# Запуск с чистой инициализацией
powershell -ExecutionPolicy Bypass -File scripts/start-full-stack.ps1 -CleanStart

# Пропустить сборку (использовать существующие бинарники)
powershell -ExecutionPolicy Bypass -File scripts/start-full-stack.ps1 -SkipBuild
```

### 🔧 Способ 2: Ручной запуск

#### Шаг 1: Сборка проекта
```powershell
# Сборка основного узла
go build -o volnixd.exe ./cmd/volnixd

# Сборка standalone версии (опционально)
go build -o volnixd-standalone.exe ./cmd/volnixd-standalone

# Или использовать Makefile
make build
```

#### Шаг 2: Инициализация узла
```powershell
# Инициализация нового узла
.\volnixd.exe init testnode --chain-id volnix-testnet

# Проверка конфигурации
.\volnixd.exe version
```

#### Шаг 3: Запуск блокчейн узла
```powershell
# Запуск узла
.\volnixd.exe start

# В отдельном терминале - проверка статуса
.\volnixd.exe status
```

#### Шаг 4: Запуск Wallet UI
```powershell
# Переход в директорию wallet-ui
cd frontend/wallet-ui

# Установка зависимостей (первый раз)
npm install

# Запуск UI
npm start
```

#### Шаг 5: Запуск Blockchain Explorer
```powershell
# Переход в директорию blockchain-explorer
cd frontend/blockchain-explorer

# Запуск explorer
powershell -ExecutionPolicy Bypass -File start-explorer.ps1
```

## Доступные сервисы

После успешного запуска будут доступны следующие сервисы:

### 🌐 Блокчейн узел
- **RPC API**: http://localhost:26657
- **P2P**: tcp://localhost:26656
- **REST API**: http://localhost:1317 (если включен)

### 💰 Wallet UI
- **URL**: http://localhost:3000
- **Функции**:
  - Создание и управление кошельками
  - Отправка и получение токенов (WRT, LZN, ANT)
  - Просмотр баланса и истории транзакций
  - Управление ролями (Гость, Гражданин, Валидатор)

### 🔍 Blockchain Explorer
- **URL**: http://localhost:8080
- **Функции**:
  - Мониторинг сети в реальном времени
  - Просмотр блоков и транзакций
  - Статистика валидаторов
  - Метрики консенсуса PoVB

## Основные команды CLI

### Управление ключами
```powershell
# Создание нового ключа
.\volnixd.exe keys add mykey

# Список всех ключей
.\volnixd.exe keys list

# Показать адрес ключа
.\volnixd.exe keys show mykey --address
```

### Запросы к блокчейну
```powershell
# Баланс аккаунта
.\volnixd.exe query bank balances volnix1address...

# Информация об аккаунте Identity
.\volnixd.exe query ident account volnix1address...

# Активированные лицензии
.\volnixd.exe query lizenz all-activated-lizenz

# Ордера на рынке ANT
.\volnixd.exe query anteil all-orders

# Статус валидаторов
.\volnixd.exe query consensus all-validators
```

### Отправка транзакций
```powershell
# Отправка токенов
.\volnixd.exe tx bank send mykey volnix1recipient... 1000000uvx --chain-id volnix-testnet

# Создание аккаунта (без валидации личности)
.\volnixd.exe tx ident create-account citizen test_hash --from mykey --chain-id volnix-testnet

# Активация лицензии
.\volnixd.exe tx lizenz activate-lizenz 1000000ulzn test_hash --from mykey --chain-id volnix-testnet

# Создание ордера на рынке ANT
.\volnixd.exe tx anteil create-order limit buy 100 1.5 test_hash --from mykey --chain-id volnix-testnet
```

## Тестирование функциональности

### 1. Создание тестового аккаунта
```powershell
# Создание ключа
.\volnixd.exe keys add testuser

# Получение адреса
$address = .\volnixd.exe keys show testuser --address

# Создание аккаунта в системе
.\volnixd.exe tx ident create-account citizen "test_identity_hash" --from testuser --chain-id volnix-testnet
```

### 2. Работа с токенами
```powershell
# Проверка баланса
.\volnixd.exe query bank balances $address

# Отправка токенов (если есть баланс)
.\volnixd.exe tx bank send testuser volnix1recipient... 1000uvx --chain-id volnix-testnet
```

### 3. Тестирование модулей
```powershell
# Lizenz модуль - активация лицензии
.\volnixd.exe tx lizenz activate-lizenz 1000000ulzn "test_hash" --from testuser --chain-id volnix-testnet

# Anteil модуль - создание ордера
.\volnixd.exe tx anteil create-order limit buy 100 1.5 "test_hash" --from testuser --chain-id volnix-testnet

# Consensus модуль - регистрация валидатора
.\volnixd.exe tx consensus register-validator "validator_info" --from testuser --chain-id volnix-testnet
```

## Мониторинг и отладка

### Логи узла
```powershell
# Запуск с debug логами
.\volnixd.exe start --log_level debug

# Сохранение логов в файл
.\volnixd.exe start > volnix.log 2>&1
```

### Проверка состояния сети
```powershell
# Статус узла
.\volnixd.exe status

# Информация о сети
curl http://localhost:26657/net_info

# Последний блок
curl http://localhost:26657/block

# Статус консенсуса
curl http://localhost:26657/consensus_state
```

### Метрики производительности
```powershell
# Использование памяти
Get-Process volnixd | Select-Object ProcessName, WorkingSet

# Размер базы данных
Get-ChildItem .volnix/data -Recurse | Measure-Object -Property Length -Sum
```

## Остановка сервисов

### Автоматическая остановка
Если использовали автоматический запуск, нажмите **Ctrl+C** в окне PowerShell.

### Ручная остановка
```powershell
# Остановка всех процессов Volnix
Get-Process | Where-Object {$_.ProcessName -like "*volnixd*"} | Stop-Process -Force

# Остановка Node.js процессов (Wallet UI)
Get-Process | Where-Object {$_.ProcessName -like "*node*"} | Stop-Process -Force

# Остановка PowerShell процессов (Explorer)
Get-Process | Where-Object {$_.ProcessName -like "*powershell*" -and $_.MainWindowTitle -like "*explorer*"} | Stop-Process -Force
```

## Устранение неполадок

### Проблема: Порт уже используется
```powershell
# Проверка занятых портов
netstat -ano | findstr :26657
netstat -ano | findstr :3000
netstat -ano | findstr :8080

# Остановка процесса по PID
taskkill /PID <PID> /F
```

### Проблема: Ошибки сборки
```powershell
# Очистка модулей Go
go clean -modcache

# Обновление зависимостей
go mod tidy
go mod download

# Пересборка
go build -o volnixd.exe ./cmd/volnixd
```

### Проблема: Ошибки npm
```powershell
# Очистка кэша npm
npm cache clean --force

# Удаление node_modules и переустановка
cd frontend/wallet-ui
Remove-Item -Recurse -Force node_modules
npm install
```

### Проблема: Узел не запускается
```powershell
# Проверка конфигурации
.\volnixd.exe validate-genesis

# Сброс данных узла
Remove-Item -Recurse -Force .volnix
.\volnixd.exe init testnode --chain-id volnix-testnet
```

## Дополнительные возможности

### Запуск тестовой сети
```powershell
# Использование готовых скриптов testnet
cd testnet
.\start.bat  # Windows
# или
./start.sh   # Linux/macOS
```

### Standalone режим
```powershell
# Запуск standalone версии (без модулей)
.\volnixd-standalone.exe init testnode
.\volnixd-standalone.exe start
```

### Интеграция с внешними системами
```powershell
# REST API запросы
curl http://localhost:1317/cosmos/bank/v1beta1/balances/volnix1address...

# WebSocket подключение
# Используйте ws://localhost:26657/websocket для real-time событий
```

## Следующие шаги

После успешного запуска вы можете:

1. **Изучить Wallet UI** - создать кошелек и протестировать операции
2. **Исследовать Explorer** - мониторить блоки и транзакции
3. **Тестировать API** - использовать CLI команды и REST API
4. **Разрабатывать dApps** - создать приложения на базе Volnix
5. **Настроить мониторинг** - добавить Prometheus/Grafana

## Поддержка

Для получения помощи:
- Проверьте логи узла: `volnix.log`
- Изучите документацию в `docs/`
- Используйте команду `.\volnixd.exe --help`

---

**🎉 Поздравляем! Volnix Protocol успешно запущен и готов к использованию!**