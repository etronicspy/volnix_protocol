# Helvetia Protocol — инструкции по созданию проекта с нуля

Ниже — практичная пошаговая инструкция, чтобы развернуть новый репозиторий Helvetia Protocol на базе Cosmos SDK (v0.53.x) с модульной архитектурой и готовым к локальному запуску демоном.

## 1) Предпосылки

- Go 1.23+ (рекомендуется актуальный 1.23/1.24)
- Git
- buf CLI для protobuf (`curl -sSL https://buf.build/install.sh | bash -s -- -b /usr/local/bin`)
- make (желательно) и gcc/clang (для сборок)
- Docker (опционально)

## 2) Новая директория и репозиторий

```bash
mkdir helvetia-protocol && cd helvetia-protocol
git init
```

## 3) Инициализация Go-модуля и зависимости

```bash
go mod init <your.module/path>/helvetia-protocol

# Базовые зависимости
go get github.com/cosmos/cosmos-sdk@v0.53.4
go get github.com/spf13/cobra@v1.9.1
go get github.com/gorilla/mux@v1.8.1
go get github.com/grpc-ecosystem/grpc-gateway@v1.16.0
go get github.com/tendermint/tm-db@v0.6.6
go get github.com/cometbft/cometbft@v0.38.17

go mod tidy
```

Рекомендуемые пины версий:
- Cosmos SDK v0.53.x
- CometBFT v0.38.x

## 4) Структура проекта

```text
helvetia-protocol/
├── app/                    # Основное приложение (BaseApp, NewApp, ModuleBasics)
├── cmd/                    # CLI-демон (helvetiad)
│   └── helvetiad/
│       └── main.go
├── x/                      # Модули
│   ├── ident/             # Идентичность (ZKP роли, активность)
│   ├── lizenz/            # LZN, MOA, активация/деактивация
│   └── anteil/            # Рынок ANT, аукционы
├── proto/                  # Protobuf определения
│   └── helvetia/
│       ├── ident/v1/  (tx.proto, query.proto, genesis.proto)
│       ├── lizenz/v1/ (tx.proto, query.proto, genesis.proto)
│       └── anteil/v1/ (tx.proto, query.proto, genesis.proto)
├── docs/                   # Документация
├── tests/                  # Тесты
├── Makefile
├── buf.yaml
├── proto/buf.gen.yaml
└── README.md
```

## 5) Buf (protobuf) настройка

Файлы:

`buf.yaml` (в корне proto или проекта):

```yaml
version: v1
name: buf.build/helvetia/protocol
deps:
  - buf.build/cosmos/cosmos-sdk
  - buf.build/cosmos/cosmos-proto
  - buf.build/googleapis/googleapis
lint:
  use:
    - DEFAULT
breaking:
  use:
    - FILE
```

`proto/buf.gen.yaml`:

```yaml
version: v1
plugins:
  - name: go
    out: ../
    opt:
      - paths=source_relative
```

Команды:

```bash
buf mod update
buf generate --template proto/buf.gen.yaml
```

## 6) Скелет приложения

- `app/`: реализовать `ModuleBasics`, `MakeEncodingConfig()`, `NewApp(...)` на базе `baseapp.BaseApp`. Зарегистрировать стандартные модули (auth, bank, staking, gov, mint, slashing, params, crisis, vesting) и ваши кастомные `ident`, `lizenz`, `anteil`.
- Бех32-префикс адресов (например `hp`): настроить в `cmd/helvetiad/main.go` до запуска командной инициализации.

CLI-демон `cmd/helvetiad/main.go` должен:
- создать корневую команду (cobra)
- добавить команды сервера (start/stop/export, status, version, keys)
- прокинуть `AppCreator`, `DefaultGenesis`, регистрацию gRPC маршрутов

Имена по умолчанию:
- Домашняя директория: `~/.helvetia`
- Имя демона: `helvetiad`

## 7) Модули (v0.53, MsgServer/QueryServer)

Для каждого модуля в `x/<module>`:
- `types/`: 
  - `keys.go` (имена, ключи префиксов),
  - `errors.go`,
  - `genesis.proto` + `genesis.go` (protobuf-сообщения GenesisState и валидация),
  - `tx.proto` (Msg сервисы), `query.proto` (Query сервисы).
- `keeper/`: бизнес-логика, доступ к KVStore; используйте `cosmossdk.io/store/prefix` для итераторов; числа — `cosmossdk.io/math` (`Int`, `Dec`/`LegacyDec`).
- `module.go`: `AppModuleBasic` + `AppModule`, регистрация сервисов: `RegisterServices(cfg)` должен регистрировать `MsgServer` и `QueryServer` из сгенерированных protobuf-интерфейсов.

Важно:
- В v0.53 не используется legacy `sdk.Handler`/`Route`. Определяйте `MsgServer` (имплементация методов из `tx.proto`) и `QueryServer` (из `query.proto`).
- Для больших чисел используйте `cosmossdk.io/math` (`math.Int`, `math.LegacyDec`).
- Genesis типы — protobuf-сообщения; их кодогеном генерирует `buf generate`.

## 8) Makefile (упрощённый пример)

```Makefile
.PHONY: install build test lint proto-gen

BINARY=helvetiad

install:
	go install ./cmd/helvetiad

build:
	go build -o bin/$(BINARY) ./cmd/helvetiad

test:
	go test ./...

proto-gen:
	buf generate --template proto/buf.gen.yaml
```

## 9) Сборка и локальный запуск

```bash
# Установка бинаря
make install

# Инициализация сети
helvetiad init mynode --chain-id helvetia-local-1

# Создание ключа
helvetiad keys add mykey

# Запуск узла
helvetiad start
```

Порты по умолчанию: RPC 26657, API 1317. Домашняя директория: `~/.helvetia`.

## 10) Тесты и качество

- Unit-тесты для всех публичных функций модулей (`keeper`, `types`)
- Интеграционные тесты межмодульных сценариев (`x/ident` ↔ `x/lizenz` ↔ `x/anteil`)
- E2E сценарии экономики (MOA, аукционы, торговля)
- Линтер: `golangci-lint` (опционально `make lint`)

## 11) Документация

- `README.md` — обзор архитектуры
- `docs/` — API модулей, экономическая модель, развертывание, безопасность

## 12) Docker (опционально)

```bash
docker build -t helvetia-protocol .
docker run -p 26657:26657 -p 1317:1317 helvetia-protocol
```

## 13) Советы по миграции c legacy-кода

- Перейти с `sdk.Handler`/`Route` на `MsgServer`
- Заменить прямую JSON-сериализацию в KV на protobuf-структуры
- Использовать `prefix.NewStore` вместо устаревших итераторов
- Числа: `cosmossdk.io/math` (`Int`), `LegacyDec`/`Dec` согласно выбранной версии SDK

---

После создания каркаса добавляйте функционал модулей по спецификации Helvetia (идентичность и ZKP, лицензии и MOA, рынок ANT и аукционы), генерируйте protobuf и регистрируйте сервисы в `app/`.


