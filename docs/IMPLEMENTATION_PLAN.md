## План реализации Helvetia Protocol (v0.53)

Этот документ описывает поэтапную реализацию проекта на базе Cosmos SDK v0.53.x с модульной архитектурой и современными сервисами (MsgServer/QueryServer). Он опирается на базовую настройку из `docs/NEW_PROJECT_SETUP.md`.

### Технические решения
- **SDK**: Cosmos SDK v0.53.x, CometBFT v0.38.x
- **Архитектура API**: gRPC-first — MsgServer и QueryServer (без legacy sdk.Handler/Route)
- **Числа**: `cosmossdk.io/math` (`Int`), десятичные параметры — `sdk.Dec`
- **Хранилище**: KVStore с префиксами (`cosmossdk.io/store/prefix`), сериализация protobuf-структур
- **Genesis**: только protobuf-типы (`genesis.proto`) и строгая валидация
- **Прото-генерация**: `buf` (lint + breaking)

### Архитектура
- `app/`: BaseApp, `ModuleBasics`, `MakeEncodingConfig`, `NewApp`, регистрация сервисов модулей
- `cmd/helvetiad`: CLI-демон (init/keys/start/export/status/version)
- `x/` модули:
  - `ident`: ZKP-идентичность, роли, активность
  - `lizenz`: LZN, активация/деактивация, MOA
  - `anteil`: рынок ANT, ордера, аукционы
- `proto/helvetia/<module>/v1`: `tx.proto`, `query.proto`, `genesis.proto`

### Дорожная карта (итерациями)

#### Итерация 0 — Каркас и инструментальная база (1–2 дня)
- Инициализация репозитория и `go mod init`
- Добавление зависимостей (SDK, CometBFT, cobra, grpc-gateway)
- `buf.yaml`, `proto/buf.gen.yaml`, Makefile (`install/build/test/proto-gen`) — выполнено
- `cmd/helvetiad/main.go`: префиксы Bech32, команды сервера
- `app/`: минимальный запуск без кастомных модулей
- Результат: `helvetiad start` поднимает пустой узел

#### Итерация 1 — Protobuf API модулей (2–3 дня)
- Для каждого модуля описать:
  - `genesis.proto`: `Params`, сущности состояния, `GenesisState` — выполнено
  - `tx.proto`: сообщения и `service Msg` — выполнено
  - `query.proto`: запросы и `service Query` — выполнено
- `buf generate`, регистрация интерфейсов в `ModuleBasics` — частично (buf generate выполнено)
- Результат: компилируемый кодоген Msg/Query/Genesis — выполнено

#### Итерация 2 — Параметры и genesis (2 дня)
- `types/params.go`: ключи параметров, `DefaultParams`, `Validate` — выполнено (включая простые строковые `Dec`)
- `genesis.go`: `DefaultGenesis`, `Validate`, `InitGenesis`, `ExportGenesis` — выполнено (protobuf-типы)
- Протянуть `Params` в `keeper` через subspace — выполнено
- Результат: корректная инициализация сети — частично (app ещё не подключён)

#### Итерация 3 — Keeper и хранилище (4–5 дней)
- `keeper/keeper.go`: subspace, базовая инициализация — выполнено
- Сериализация protobuf-структур; итераторы через `prefix` — позже
- Базовые CRUD и инварианты состояния — позже

#### Итерация 4 — MsgServer (5–7 дней)
- `ident`: заглушки `VerifyIdentity`, `MigrateRole`, `ChangeRole` — выполнено (заготовки)
- `lizenz`: заглушки `ActivateLZN`, `DeactivateLZN` — выполнено (заготовки)
- `anteil`: заглушки `PlaceOrder`, `CancelOrder`, `PlaceBid` — выполнено (заготовки)
- Результат: транзакции меняют состояние по правилам — позже (логика будет добавлена)

#### Итерация 5 — QueryServer (3–4 дня)
- Параметры и основные выборки — выполнено заглушками (пустые ответы, пагинация подключена)
  - `ident`: verified account по адресу/все/по роли, `Params`
  - `lizenz`: активированные/деактивируемые LZN, MOA-статус, `Params`
  - `anteil`: ордера/сделки/аукционы, `Params`

#### Итерация 6 — Интеграции и события (3–4 дня)
- События (`sdk.Event`), подписки
- Межмодульные проверки: роль/активность (`ident`) в `lizenz` и `anteil`
- Базовые инварианты (кризис-модуль)

#### Итерация 7 — Тесты и E2E (1–2 недели)
- Unit: `keeper`, `types`, `msg_server`, `query_server`
- Интеграция: сценарии `ident↔lizenz`, `lizenz↔anteil`
- E2E: локальная сеть, транзакции, проверка балансов/состояний

#### Итерация 8 — Производительность и безопасность (1–2 недели)
- Оптимизация KV-доступа, кеширование
- Улучшение matching engine (структуры книги ордеров)
- ZKP: интеграция реальной верификации
- Аудит MOA, аукционов и санкций

#### Итерация 9 — DevEx и релиз (3–5 дней)
- Улучшения CLI и UX
- Документация (README, API модулей, деплой, экономика, безопасность)
- Dockerfile, CI (lint/test/build), релизные артефакты

### Детализация по модулям

#### x/ident
- Сущности: `VerifiedAccount { address, role, last_active, identity_hash }`
- Params: сроки неактивности по ролям, лимиты
- Msg: `VerifyIdentity`, `MigrateRole`, `ChangeRole`
- Query: по адресу, список, по роли, `Params`
- Keeper: связь `identity_hash → address`, обновление активности, деактивация

#### x/lizenz
- Сущности: `ActivatedLizenz`, `DeactivatingLizenz`, `MOAStatus`
- Params: `MaxActivatedLZNPerValidator`, `ActivityCoefficient (Dec)`, `DeactivationPeriod`, `InactivityPeriod`
- Msg: `ActivateLZN`, `DeactivateLZN`
- Логика: лимиты, перевод LZN модулю/обратно, вычисление MOA, неактивность валидаторов
- Query: по валидатору/все, MOA, `Params`

#### x/anteil
- Сущности: `Order`, `Trade`, `Auction`, `Bid`
- Params: `MaxOrderAmount`, `MinOrderAmount`, `TradingFee (Dec)`, `AuctionPeriod`
- Msg: `PlaceOrder`, `CancelOrder`, `PlaceBid`
- Логика: валидация, перевод средств, комиссия, аукционы
- Query: ордера/сделки/аукционы, `Params`

### Регистрация в приложении
- `ModuleBasics`: все `AppModuleBasic`
- `NewApp`: keepers, subspaces, порядок `Begin/End/InitGenesis`
- `RegisterServices(cfg)`: регистрация Msg/Query сервисов
- Bech32-префиксы (`hp`) — в `cmd/helvetiad/main.go`

### Тестирование
- Unit: мок-keepers, `sdk.Context`, property-based тесты расчётов
- Интеграция: `InitGenesis` + последовательности Msg
- E2E: локальный узел, транзакции, события и инварианты

### Риски и меры
- Несоответствие API SDK: строго следовать v0.53 (без legacy)
- Числовые риски: везде `math.Int`, `Dec` — только для параметров/тарифов
- Производительность: простая модель matching engine → последующие оптимизации
- ZKP: контракт интерфейса, затем подключение реальной верификации

### Definition of Done
- Узел запускается (`helvetiad start`) с подключёнными модулями
- Msg/Query покрыты unit-тестами, интеграция проходит
- E2E сценарии экономики успешны
- Документация актуальна

### Ближайшие задачи
1. Итерация 0: каркас `app/`, `cmd/helvetiad`, Makefile, buf-конфиг
2. Итерация 1: protobuf API для `ident`, `lizenz`, `anteil` + кодоген
3. Итерация 2–3: `params`, `genesis`, `keeper` (CRUD), провязка в `app/`
4. Итерация 4–5: `MsgServer`/`QueryServer` и CLI-команды
5. Итерация 7: unit/integration/e2e + базовая CI
