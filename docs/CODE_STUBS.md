# Заглушки и временные реализации (Code Stubs)

Отчёт о заглушках, TODO и временных реализациях в коде (исключая сгенерированный proto и тесты).

## Критичные (реальная логика отсутствует)

| Файл | Строка | Описание |
|------|--------|----------|
| ~~`backend/api/handlers.go`~~ | — | **Исправлено**: `anteilOrdersHandler` и `anteilAuctionsHandler` вызывают gRPC; anteil query server возвращает данные из keeper |
| ~~`x/ident/keeper/keeper.go`~~ | — | **Исправлено**: `processRoleMigrations` — обрабатывает незавершённые миграции в BeginBlocker |
| `x/ident/keeper/keeper.go` | 548–550 | **TODO (ZKP)**: `ValidateRoleChangeProof` — только проверка формата, нет интеграции с ZKP (gnark/circom) |
| ~~`x/ident/keeper/msg_server.go`~~ | — | **Исправлено**: `RegisterVerificationProvider` — сохраняет провайдера в keeper, возвращает реальный accreditation hash |

## Мониторинг и контекст

| Файл | Строка | Описание |
|------|--------|----------|
| `app/monitoring.go` | 179–185 | **TODO**: метрики consensus (validators, burned, weight) — всегда 0, нет запроса к keeper |
| `app/monitoring.go` | 208, 270, 318 | **TODO**: контекст для мониторинга — закомментирован код запросов к keeper, возвращаются нули |
| `app/monitoring.go` | 319, 329 | `getValidatorCount` / метрики — возврат 0 до появления контекста |

## Ante и проверки транзакций

| Файл | Строка | Описание |
|------|--------|----------|
| `app/ante.go` | 41, 59, 63 | **Skip**: проверки timeout height, подписей и memo отключены — «For now, we skip» (требуются типы tx) |

## Типы и регистрация

| Файл | Строка | Описание |
|------|--------|----------|
| `x/governance/module.go` | 33 | **TODO**: Register types when needed |
| `x/governance/types/types.go` | 4 | **Temporary**: GenesisState использует `[]interface{}` до proto-генерации |

## Нормальные/ожидаемые

- **proto/gen/** — `Unimplemented*Server` генерируются protoc, встраиваются в реальные серверы; это не заглушки в смысле «не сделано».
- **Тесты** (`tests/*.go`, `*_test.go`) — множество TODO и `t.Skip` оставлены как план проверок; при необходимости вынести в отдельный отчёт.

## Рекомендации

1. ~~**backend/api**~~: реализовано — orders и auctions идут через gRPC в anteil query server (keeper).
2. ~~**ident/keeper processRoleMigrations**~~: реализовано — в BeginBlocker обрабатываются незавершённые миграции.
3. ~~**ident/msg_server RegisterVerificationProvider**~~: реализовано — сохранение в keeper и реальный accreditation hash.
4. **ident/keeper ZKP**: оставить до интеграции с gnark/circom; см. `ValidateRoleChangeProof`.
5. **app/monitoring**: ввести способ получения `sdk.Context` для мониторинга (например, last committed) и включить реальные запросы к keeper.
6. **app/ante**: при появлении полной типизации tx включить проверки timeout height, подписей и memo.
