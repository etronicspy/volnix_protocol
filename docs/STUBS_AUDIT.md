# Аудит заглушек, приоритетов и непрактичных функций

Дата аудита: 2026-02-21

---

## 1. Таблица заглушек и приоритетов реализации

### Критические (P0) — блокируют работу протокола

| # | Файл | Функция/Область | Проблема | Статус |
|---|------|-----------------|----------|--------|
| 1 | `x/anteil/keeper/msg_server.go` | `UpdateOrder()` | ~~Возвращает `success: true` без обновления~~ | **ИСПРАВЛЕНО**: загрузка из store, проверка владельца, обновление полей |
| 2 | `x/anteil/keeper/msg_server.go` | `RegisterMarketMaker()` | ~~Возвращает хардкод `mm-123`~~ | **ИСПРАВЛЕНО**: регистрация в store, валидация, реальный ID |
| 3 | `x/anteil/keeper/msg_server.go` | `ProvideLiquidity()` / `WithdrawLiquidity()` | ~~Хардкод shares `"1000"`~~ | **ИСПРАВЛЕНО**: расчёт долей, lock/unlock ANT, liquidity shares в store |
| 4 | `x/anteil/keeper/msg_server.go` | `StakeANT()` / `UnstakeANT()` / `ClaimRewards()` | ~~Хардкод reward rate/rewards~~ | **ИСПРАВЛЕНО**: стейкинг в store, lock/unlock, расчёт наград по rate из params |
| 5 | `app/ante.go` | `ImprovedAnteHandler` | ~~Пропущены: timeout height, подписи, memo~~ | **ИСПРАВЛЕНО**: проверки timeout height, memo length, signature presence |
| 6 | `app/snapshot.go` | `exportState()` | ~~Экспортирует только blockHeight~~ | **ИСПРАВЛЕНО**: итерация по всем KV store модулей, JSON сериализация |
| 7 | `app/snapshot.go` | `importState()` | ~~Только логирует~~ | **ИСПРАВЛЕНО**: десериализация, очистка store, запись entries |

### Высокие (P1) — ограничивают функциональность

| # | Файл | Функция/Область | Проблема | Статус |
|---|------|-----------------|----------|--------|
| 8 | `x/ident/keeper/keeper.go:566-575` | `ValidateRoleChangeProof()` | ~~Только проверка формата, нет ZKP~~ | **ИСПРАВЛЕНО**: `ZKPProof` (JSON/proto), проверка `proof_hash`, binding к `address/identity_hash/role`, anti-replay |
| 9 | `x/governance/genesis.go` | `InitGenesis()` / `ExportGenesis()` | ~~`[]interface{}{}` вместо proposals/votes~~ | **ИСПРАВЛЕНО**: ExportGenesis выгружает реальные proposals из keeper |
| 10 | `x/governance/client/cli/` | `GetQueryCmd()` / `GetTxCmd()` | ~~Пустые команды~~ | **ИСПРАВЛЕНО**: CLI для submit proposal, vote, query proposal(s) |
| 11 | `x/consensus/client/cli/` | Query/Tx commands | ~~Пустые команды~~ | **ИСПРАВЛЕНО**: CLI для params, validators, select-block-creator |
| 12 | `app/monitoring.go` | Все `get*Metrics()` | ~~Всегда 0 — нет `sdk.Context`~~ | **ИСПРАВЛЕНО**: `NewContext(true)` для реальных метрик из keepers |
| 13 | `x/integration/keeper/keeper.go` | `getAnteilUserPosition()` | ~~Mock: хардкод balance `"1000.0"`~~ | **ИСПРАВЛЕНО**: вызов `anteilKeeper.GetUserPosition()` |
| 14 | `x/integration/module.go` | Почти весь модуль | ~~Пустые: InitGenesis, ExportGenesis~~ | **ИСПРАВЛЕНО**: InitGenesis инициализирует health, ExportGenesis экспортирует состояние |

### Средние (P2) — влияют на качество

| # | Файл | Функция/Область | Проблема | Статус |
|---|------|-----------------|----------|--------|
| 15 | `app/upgrade.go` | `migrateAnteilModuleV0_3_0()` / `migrateConsensusModuleV0_3_0()` | ~~Placeholder — `return nil`~~ | **ИСПРАВЛЕНО**: re-save entities, обновление params с defaults |
| 16 | `app/upgrade.go` | `migrateIdentModuleV0_2_0()` / `migrateLizenzModuleV0_2_0()` | ~~Итерация без действий~~ | **ИСПРАВЛЕНО**: re-save через SetVerifiedAccount/SetActivatedLizenz |
| 17 | `x/ident/keeper/query_server.go` | `Params()` | ~~Хардкод params~~ | **ИСПРАВЛЕНО**: чтение из keeper.GetParams() |
| 18 | `app/ratelimit.go` | `Cleanup()` | ~~Пустое тело~~ | **ИСПРАВЛЕНО**: TTL-очистка по lastSeen, addressEntry с timestamp |
| 19 | `x/consensus/keeper/keeper.go` | `ProcessHalving()` | ~~Закомментированы timestamp поля~~ | **ИСПРАВЛЕНО**: LastHalvingDate, EstimatedNextHalvingDate раскомментированы |

### Фронтенд (P2–P3)

| # | Файл | Компонент | Проблема | Что нужно |
|---|------|-----------|----------|-----------|
| 20 | AntMarket.tsx | AntMarket | ~~Хардкод~~ | **ИСПРАВЛЕНО**: fetch /volnix/anteil/v1/orders |
| 21 | ModulesStatus.tsx | ModulesStatus | ~~Хардкод~~ | **ИСПРАВЛЕНО**: fetch /status |
| 22 | blockchainService.ts | ZKP proof | ~~`zkpProof: ''`~~ | **ИСПРАВЛЕНО**: клиент генерирует структурированный `ZKPProof` с `proofHash/publicInputs/proofData/createdAt` |
| 23 | App.tsx | handleViewBlock | ~~alert()~~ | **ИСПРАВЛЕНО**: BlockDetail |
| 24 | SendTokens.tsx | Fee display | ~~Хардкод~~ | **ИСПРАВЛЕНО**: getEstimatedFee() |

---

## 2. Функции, непрактичные для реальных блокчейн-систем

| # | Файл | Функция | Проблема | Рекомендация |
|---|------|---------|----------|-------------|
| **A** | `x/anteil/keeper/economic_engine.go` целиком | `ProcessOrderMatching`, `executeTrade`, `CalculateMarketMetrics` | ~~**`float64` для финансовых расчётов**~~ | **ИСПРАВЛЕНО**: заменено на `cosmossdk.io/math.LegacyDec` для детерминизма |
| **B** | `x/anteil/keeper/economic_engine.go` | `createMarketMakingOrders()` | ~~**Системный маркет-мейкер**~~ | **ИСПРАВЛЕНО**: Owner = `authtypes.NewModuleAddress(types.ModuleName)` — реальный module account anteil |
| **C** | `app/snapshot.go` | `SnapshotManager` | ~~**In-memory хранение**~~ | **ИСПРАВЛЕНО**: SetSnapshotDir() — персистенция в home/data/snapshots, переживает перезапуск |
| **D** | `x/ident/keeper/security_enhancements.go:244-303` | `VerifyZKProofIntegrity()` | ~~**Фейковая ZKP верификация**~~ | **ИСПРАВЛЕНО**: structured proof integrity (`proof_hash`), TTL, replay-protection, optional provider signature verification |
| **E** | `x/integration/keeper/keeper.go` | `handleIdentityVerified`, `handleLizenzActivated`, `handleConsensusParticipation` | ~~**Cross-module events только логируют**~~ | **ИСПРАВЛЕНО**: реальные cross-module проверки, SDK events, валидация eligibility |
| **F** | `x/integration/types/integration.go` | `calculateOverallScore()` | ~~**Возвращает 0.0**~~ | **ИСПРАВЛЕНО**: формула с учётом баланса ANT и статуса валидатора |
| **G** | `app/ratelimit.go` | `RateLimiter` | ~~**In-memory на уровне app**~~ | **ИСПРАВЛЕНО**: Enabled: false по умолчанию — не влияет на консенсус; включить только для RPC-защиты |
| **H** | `app/upgrade.go` / `app/app.go` | Upgrade flow | ~~Кастомный upgrade manager~~ | **ИСПРАВЛЕНО**: runtime переведён на стандартный `x/upgrade` keeper/module + pre-block hook + SDK upgrade handlers |
| **I** | `x/anteil/keeper/economic_engine.go` | Sort в `ProcessOrderMatching` | ~~**`strconv.ParseFloat` в sorting**~~ | **ИСПРАВЛЕНО**: заменено на `LegacyDec` вместе с A |
| **J** | `x/ident/keeper/security_enhancements.go` | JSON для store | ~~**`json.Marshal`/`Unmarshal` для KV store**~~ | **ИСПРАВЛЕНО**: VerificationProvider использует `proto.Marshal`/`proto.Unmarshal` |

---

## 3. Сводная матрица приоритетов

```
Критичность \ Сложность  │  Низкая (1–2 дня)  │  Средняя (3–5 дней)  │  Высокая (1–2 нед)
──────────────────────────┼────────────────────┼─────────────────────┼───────────────────
  P0 — Блокер             │ #5 ante handler    │ #1–4 anteil stubs   │ #6–7 snapshot
                          │ #17 ident params   │                     │
──────────────────────────┼────────────────────┼─────────────────────┼───────────────────
  P1 — Высокий            │ #12 monitoring ctx │ #9–11 governance    │ —
                          │ #13 integration    │ #14 integration mod │
                          │ #19 halving fields │                     │
──────────────────────────┼────────────────────┼─────────────────────┼───────────────────
  P2 — Средний            │ #15–16 migrations  │ #20–21 frontend API │ A — float64→sdk.Dec
                          │ #18 ratelimit      │ #24 dynamic fees    │ H — x/upgrade
                          │ #23 block detail   │ J — json→protobuf   │ C — snapshot mgr
──────────────────────────┼────────────────────┼─────────────────────┼───────────────────
  P3 — Низкий / Удалить   │ F — score=0        │ B — system MM       │ G — rate limiter
                          │ E — event logging  │                     │
```

---

## 4. Рекомендуемый порядок действий

1. **Немедленно (неделя 1):** Заменить `float64` на `sdk.Dec` в economic engine (A) — это консенсус-критичный баг. Исправить anteil msg_server стабы (#1–4).

2. **Неделя 2:** Подключить мониторинг к реальному context (#12), исправить ante handler (#5), подключить integration keeper к реальному anteil (#13).

3. **Неделя 3–4:** Governance genesis/CLI (#9–11), миграционные скелеты (#15–16), ~~заменить кастомный UpgradeManager на `x/upgrade` (H)~~.

4. **Неделя 5+:** ZKP интеграция (#8), snapshot полная реализация (#6–7), фронтенд подключения (#20–21).

5. **Решить — оставить или удалить:** системный маркет-мейкер (B), in-memory rate limiter (G), пустой integration module event system (E).

---

## Статистика

- **Заглушек в backend:** 19
- **Заглушек в frontend:** 5
- **Архитектурно-проблемных функций:** 10 (A–J)
- **Самый опасный для консенсуса:** A (`float64` в economic engine)
