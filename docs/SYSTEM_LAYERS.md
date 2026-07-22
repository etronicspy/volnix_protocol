# Volnix / Helvetia Protocol — логическое деление системы

Документ делит систему на **логические части (подсистемы)**, чтобы о каждой можно
было рассуждать отдельно: за что отвечает, какие сущности хранит, какими
сообщениями управляется, с кем связана. Деление основано на реальном коде
(`x/`, `proto/volnix/`, `app/`, `cmd/`, `simulation/`), а не на абстракции.

> Обозначения: **Msg** — транзакция (изменяет состояние), **Query** — чтение,
> §X.Y — раздел канона `docs/volnix_protocol.md`.

---

## Карта подсистем

```mermaid
flowchart TB
  subgraph L0 [Слой 0 · Инфраструктура узла]
    cometbft[CometBFT consensus engine]
    cosmos[Cosmos SDK baseapp]
    abci[ABCI++ / app wiring]
  end
  subgraph L1 [Слой 1 · Протокольное ядро (x/)]
    ident[ident · идентичность ZKP]
    lizenz[lizenz · лицензии LZN]
    anteil[anteil · рынок ANT]
    consensus[consensus · PoVB вес/награды]
    governance[governance · DAO параметры]
    integration[integration · кросс-модульная связность]
  end
  subgraph L2 [Слой 2 · Активы и роли]
    wrt[(WRT · деньги + голос)]
    lzn[(LZN · лицензия/мощность)]
    ant[(ANT · топливо внутр. рынка)]
    roles[Гражданин / Поставщик / Валидатор]
  end
  subgraph L3 [Слой 3 · Доступ и клиенты]
    grpc[gRPC / REST gateway]
    cli[CLI volnixd]
    wallet[Кошелёк Volnix]
    backend[backend/ сервисы-обвязка]
  end
  subgraph L4 [Слой 4 · Наблюдаемость и эксплуатация]
    monitoring[monitoring / metrics]
    infra[infrastructure/ Grafana]
  end
  subgraph SIM [Off-chain · Симуляция R&D]
    simback[simulation/backend FastAPI]
    simfront[simulation/frontend React]
  end

  cometbft --> cosmos --> abci --> L1
  L1 --> L2
  L1 --> grpc --> wallet
  grpc --> cli
  grpc --> backend
  abci --> monitoring --> infra
  L1 -. зеркалит правила .-> simback --> simfront
```

---

## Слой 0 — Инфраструктура узла (consensus engine + SDK)

**Назначение:** упорядочивание транзакций, репликация состояния, жизненный цикл блока.

- **CometBFT** — BFT-консенсус (propose → pre-vote → pre-commit → commit), p2p,
  mempool, порядок tx в блоке. Ядро **не** кастомизируется под Volnix (§6.1).
- **Cosmos SDK baseapp** — маршрутизация Msg/Query, фазы блока
  (BeginBlock → DeliverTx → EndBlock), управление KV-store, ante-handler.
- **App wiring** — `app/`: сборка модулей, ABCI-обёртки, ante (`app/ante.go`),
  rate-limit (`app/ratelimit.go`), снапшоты (`app/snapshot.go`),
  апгрейды (`app/upgrade.go`), gRPC/REST серверы (`app/grpc_server.go`,
  `app/server.go`), мониторинг (`app/monitoring.go`).
- **Бинарь ноды** — `cmd/volnixd/` (полноценный узел) и
  `cmd/volnixd-standalone/` (автономный режим для dev/testnet; см. правило
  CometBFT/CosmJS про `CreateEmptyBlocks` и сброс `priv_validator_state.json`).

**Что относится сюда:** всё, что не про экономику Volnix, а про «как блокчейн
вообще работает». Деление блоков по высоте, детерминизм исполнения, genesis-state
(`InitGenesis`).

---

## Слой 1 — Протокольное ядро (модули `x/`)

Шесть модулей Cosmos SDK. Каждый — отдельная подсистема со своим store,
Msg-сервисом, Query-сервисом, genesis и параметрами.

### 1.1 `ident` — Идентичность (§3.1, §3.2, §4.2)
- **Отвечает за:** ZKP-верификацию уникальности, выбор взаимоисключающей роли
  (Поставщик/Валидатор), миграцию роли («цифровое наследство»), реестр
  аккредитованных провайдеров верификации.
- **Msg:** `VerifyIdentity`, `MigrateRole`, `ChangeRole`,
  `RegisterVerificationProvider`.
- **Хранит:** верифицированные аккаунты, привязки ZKP-идентификатора к роли,
  статус провайдеров.
- **Связи:** даёт `lizenz`/`consensus` право быть валидатором; даёт `anteil`
  право быть Поставщиком.

### 1.2 `lizenz` — Лицензии LZN (§4.1, §5.1)
- **Отвечает за:** активацию/деактивацию LZN (лимит ≤ ⌊эталон/3⌋ на адрес),
  срок заморозки, учёт «майнинговой мощности» `L_i`, трекинг наград.
- **Msg:** `ActivateLZN`, `DeactivateLZN`.
- **Хранит:** активированные LZN на адрес, расписание разморозки, reward-tracker.
- **Связи:** `L_i` — верхняя граница участия валидатора в `consensus`
  (`b_i + s_i ≤ L_i`).

### 1.3 `anteil` — Внутренний рынок ANT (§4.1, §5.2, §5.5)
- **Отвечает за:** книгу ордеров ANT↔WRT, детерминированный матчинг,
  эскроу, эпохальную эмиссию и сброс ANT у Поставщиков, запрет прямых
  переводов ANT (`economic_engine.go`).
- **Msg:** `PlaceOrder`, `CancelOrder`, `UpdateOrder`.
- **Хранит:** ордера (bids/asks), позиции пользователей, состояние эпохи
  (объём продаж, коэффициент эмиссии).
- **Связи:** источник ANT для валидаторов; потребитель WRT в пользу Поставщиков.

### 1.4 `consensus` — PoVB: вес и награды (§5.1, §5.4, §6.1)
- **Отвечает за:** объявление per-height сжигания `b_i`/ставки `s_i`,
  расчёт веса `w_i = s_i / L_i`, формирование `ValidatorSet` для N+1 в EndBlocker,
  глобальный лимит λ и потолок K, дележ комиссий `F·(b_i/B)`, базовую награду WRT,
  слешинг/evidence.
- **Msg:** `DeclarePerHeightBurn`, `UpdateConsensusState`, `SetValidatorWeight`,
  `RegisterConsensusMapping`.
- **Хранит:** объявления участия, текущий вес валидаторов, агрегаты сжигания.
- **Связи:** читает `L_i` из `lizenz`, статус из `ident`; отдаёт `ValidatorUpdates`
  в CometBFT (Слой 0). **Это слой, к которому относятся все проблемы из
  `docs/CANON_PROBLEMS.md` §5.4.**

### 1.5 `governance` — Ограниченное DAO (§7)
- **Отвечает за:** предложения и голосование держателей WRT, тайм-лок,
  применение параметров в пределах min/max, заданных кодом.
- **Msg:** `SubmitProposal`, `Vote`, `ExecuteProposal`.
- **Хранит:** проположения, голоса, управляемые параметры (`governable_params.go`),
  применитель параметров (`parameter_applier.go`).
- **Связи:** меняет параметры всех остальных модулей (λ, K, EpochBlocks,
  HalvingInterval, T_g/T_v, лимит Поставщиков и т.д. — §7.2).

### 1.6 `integration` — Кросс-модульная связность
- **Отвечает за:** агрегированный статус валидатора по всем модулям
  (`ValidatorIntegrationStatus`), кросс-модульные события, health-метрики связей.
- **Хранит:** карту интеграций модулей, журнал кросс-модульных событий.
- **Связи:** «склейка» — объединяет `ident`+`lizenz`+`anteil`+`consensus` в единый
  взгляд на участника.

---

## Слой 2 — Активы и роли (доменная модель)

Не модуль, а **сквозная доменная модель**, реализуемая модулями Слоя 1.

### Активы
| актив | роль | эмиссия | обращение |
|---|---|---|---|
| **WRT** | ценность + голос DAO | фикс., халвинг (§6.2) | свободное |
| **LZN** | лицензия/мощность `L_i` | одноразовая | торгуемо, активация замораживает |
| **ANT** | топливо PoVB | эпохальная Поставщикам (§5.5) | **только** внутр. рынок + служебные движения (§4.1) |

### Роли (§4.2)
- **Гражданин** — не-верифицирован; WRT/LZN; ANT недоступен.
- **Поставщик** — верифицирован; сторона предложения ANT; MOA `T_g`.
- **Валидатор** — верифицирован; активирует LZN, покупает+жжёт ANT; MOA `T_v`.
- **Genesis-задел (§6.3):** 2 фиксированных адреса (Поставщик + Валидатор) без ZKP.

### Сквозные механики
- **MOA** (§5.3) — «иммунная система»: статус снимается без подписанной tx в окне.
- **Эпоха ANT** (§5.5) — сброс+эмиссия у Поставщиков по границе `EpochBlocks`.
- **Халвинг WRT** (§6.2) — каждые `HalvingInterval` блоков.

---

## Слой 3 — Доступ и клиенты

**Назначение:** как внешний мир говорит с цепью.

- **gRPC / REST gateway** — `app/grpc_server.go`, `app/server.go`: Query/Msg наружу.
- **CLI** — `x/*/client/cli/`: `tx.go` (отправка Msg), `query.go` (чтение).
- **Кошелёк Volnix** — эталонный открытый клиент: книга ордеров, подпись tx
  (§5). Альтернативные клиенты к тому же on-chain API допускаются.
- **`backend/`** — репозиторная подсистема production-сервисов-обвязки (REST API,
  индексеры, вебхуки) — **отдельно** от ноды и от симуляции.

---

## Слой 4 — Наблюдаемость и эксплуатация

- **monitoring** — `app/monitoring.go`: метрики ноды/модулей.
- **infrastructure/** — Grafana-дашборды, мониторинг-конфиги.
- **CI/CD** — `.github/workflows/`: линт, сборка, тесты, покрытие.
- **Миграции/апгрейды** — `app/upgrade.go`, governance-proposal → upgrade handler.

---

## Off-chain — Симуляция R&D (НЕ часть ноды)

Отдельная песочница, **зеркалящая** экономику §3–§6 упрощённо; не консенсус Cosmos.

- **`simulation/backend/`** — FastAPI: `core/engine.py` (BeginBlock→DeliverTx→
  EndBlock), `core/consensus.py` (PreVote/PreCommit), `core/anteil`-аналоги рынка,
  `core/network.py` (NetworkSim — логическая многоузловость), `core/auto_declare.py`.
- **`simulation/frontend/`** — React/Vite дашборд.
- **Источник истины:** on-chain модули `x/` и `proto/`. Симуляция может упрощать.

---

## Сводная таблица «часть → код → канон»

| Подсистема | Код | Канон | Транзакции |
|---|---|---|---|
| Инфраструктура узла | `app/`, `cmd/volnixd*` | §6.1–6.2 | — |
| Идентичность | `x/ident` | §3.1–3.2, §4.2 | VerifyIdentity, MigrateRole, ChangeRole, RegisterVerificationProvider |
| Лицензии | `x/lizenz` | §4.1, §5.1 | ActivateLZN, DeactivateLZN |
| Внутренний рынок | `x/anteil` | §4.1, §5.2, §5.5 | PlaceOrder, CancelOrder, UpdateOrder |
| Консенсус/PoVB | `x/consensus` | §5.1, §5.4, §6.1 | DeclarePerHeightBurn, UpdateConsensusState, SetValidatorWeight, RegisterConsensusMapping |
| DAO | `x/governance` | §7 | SubmitProposal, Vote, ExecuteProposal |
| Связность | `x/integration` | — | — (агрегатор) |
| Активы/роли | сквозь `x/*` | §4.1–4.2 | — |
| Клиенты | `x/*/client/cli`, кошелёк, `backend/` | §5 | — |
| Наблюдаемость | `app/monitoring.go`, `infrastructure/` | §7.2 (операц.) | — |
| Симуляция (off-chain) | `simulation/` | зеркало §3–§6 | — |
