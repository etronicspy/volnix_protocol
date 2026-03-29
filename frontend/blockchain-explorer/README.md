# Volnix Blockchain Explorer (React)

Современный React-приложение для мониторинга блокчейна Volnix Protocol в реальном времени.

## Особенности

- 🔍 **Мониторинг сети** - Статус сети, высота блока, активные валидаторы
- 📦 **Просмотр блоков** - Последние блоки с детальной информацией
- 💸 **Транзакции** - История транзакций в реальном времени
- 👑 **Валидаторы** - Список валидаторов с статистикой
- ⚖️ **Консенсус** - Информация о PoVB консенсусе и сжигании ANT
- 🔧 **Модули** - Статус всех модулей протокола

## Технологии

- **React 18** с TypeScript
- **React Hooks** для управления состоянием
- **CSS3** с адаптивным дизайном
- **RPC API** для получения данных блокчейна
- **REST API** для валидаторов и параметров консенсуса

## Установка и запуск

```bash
# Установка зависимостей
npm install

# Запуск в режиме разработки
npm start

# Сборка для продакшена
npm run build
```

## Конфигурация

Эндпоинты настраиваются через переменные окружения:

```bash
REACT_APP_RPC_ENDPOINT=http://localhost:26657
REACT_APP_REST_API_ENDPOINT=http://localhost:1317
```

По умолчанию используются:
- RPC: `http://localhost:26657`
- REST API: `http://localhost:1317`

## Структура проекта

```
frontend/blockchain-explorer/
├── src/
│   ├── components/          # React компоненты
│   │   ├── NetworkOverview.tsx
│   │   ├── StatsGrid.tsx
│   │   ├── Tabs.tsx
│   │   ├── BlocksList.tsx
│   │   ├── TransactionsList.tsx
│   │   ├── ValidatorsList.tsx
│   │   ├── ConsensusInfo.tsx
│   │   └── ModulesStatus.tsx
│   ├── hooks/               # Custom React hooks
│   │   └── useNetworkData.ts
│   ├── services/            # API сервисы
│   │   └── api.ts
│   ├── types/               # TypeScript типы
│   │   └── index.ts
│   ├── App.tsx              # Главный компонент
│   ├── App.css              # Стили приложения
│   ├── index.tsx            # Точка входа
│   └── index.css            # Глобальные стили
├── public/
│   └── index.html
├── package.json
├── tsconfig.json
└── config-overrides.js
```

## Автоматическое обновление

Данные автоматически обновляются каждые 30 секунд. Можно также обновить вручную, нажав кнопку обновления в правом нижнем углу.

## Требования

Для полной функциональности необходимо:
1. Запущенный блокчейн узел (RPC на порту 26657, gRPC на 9090)
2. Запущенный REST API сервер (порт 1317)

## Проверка отображения валидаторов

Валидаторы синхронизируются из genesis в consensus-модуль при InitChain и обновляются в EndBlock. Для проверки:

1. **Сбросить и запустить ноду** (после сброса InitChain выполнится заново):
   ```bash
   ./scripts/testnet-reset-and-start.sh
   # В отдельном терминале:
   ./build/volnixd start --home testnet/node0
   ```

2. **Запустить REST API** (подключение к gRPC 9090):
   ```bash
   cd backend/api && go run main.go server.go -grpc-addr=localhost:9090 -http-addr=0.0.0.0:1317
   ```

3. **Проверить API**:
   ```bash
   curl -s http://localhost:1317/volnix/consensus/v1/validators | jq '.validators | length'
   ```
   Ожидается: `1` (один genesis-валидатор)

4. **Запустить Explorer** и открыть вкладку Validators — должен отображаться активный валидатор.

## Миграция с HTML версии

Эта React версия полностью заменяет предыдущую HTML/JavaScript версию. Все функции сохранены, но код теперь:
- Модульный и переиспользуемый
- Типобезопасный (TypeScript)
- Легче поддерживать и расширять
- Следует лучшим практикам React
