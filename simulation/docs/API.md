# Volnix Simulation API (v0.1.0)

> **Автогенерация.** Не редактируйте файл руками — обновляйте через
> `python -m scripts.gen_api_md simulation/docs/API.md` (см. CI).

## `misc`

### `GET /` — Read Root

### `GET /api/account/{address}/history` — Api Account History

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `address` | path | string | ✓ |  |  |
| `limit` | query | integer |  | 100 |  |

### `GET /api/analytics/gini` — Api Gini

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `asset` | query | string |  | wrt |  |

### `GET /api/analytics/kpi` — Api Analytics Kpi

### `GET /api/blocks` — Api Blocks

Если from/to не заданы — отдаём `tail` последних блоков из RAM.

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `from_height` | query | integer | null |  |  |  |
| `to_height` | query | integer | null |  |  |  |
| `tail` | query | integer |  | 50 |  |

### `GET /api/blocks/{height}` — Api Block By Height

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `height` | path | integer | ✓ |  |  |

### `POST /api/bot/control` — Control Bot

**Тело запроса:**
- `application/json` — #/components/schemas/BotControlRequest

### `GET /api/bot/status` — Get Bot Status

### `GET /api/canon-log` — Api Canon Log

Поток canon-аудита.

Если `since_id` > 0 и persistent ledger включён — отдаём из JSONL;
иначе — из in-memory ringbuffer (хвост).

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `since_id` | query | integer |  | 0 |  |
| `limit` | query | integer |  | 200 |  |

### `GET /api/export/balances.csv` — Api Export Balances

### `GET /api/export/blocks.jsonl` — Api Export Blocks

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `from_height` | query | integer | null |  |  |  |
| `to_height` | query | integer | null |  |  |  |

### `GET /api/export/canon_log.jsonl` — Api Export Canon Log

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `limit` | query | integer |  | 1000 |  |

### `GET /api/export/ticks.csv` — Api Export Ticks

### `GET /api/market/bars` — Api Market Bars

OHLC в формате Apache ECharts candlestick: category[], values[][open,close,low,high], times[].

interval_sec: 0 — одна свеча на сделку; 1, 60, 300, … — корзина в секундах.

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `interval_sec` | query | integer |  | 0 |  |
| `limit_ticks` | query | integer |  | 50000 |  |

### `GET /api/market/history` — Api Market History

История тиков цены для графика/виджета (хвост до limit, max 50_000).

Формат тика: time (строка), price (float), ts (unix_seconds, желательно).
Поле ts обязательно для корректной агрегации; для Apache ECharts см. GET /api/market/bars.

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `limit` | query | integer |  | 10000 |  |

### `POST /api/network/config` — Api Network Config

**Тело запроса:**
- `application/json` — #/components/schemas/NetworkConfigBody

### `GET /api/network/nodes` — Api Network Nodes

### `GET /api/network/topology` — Api Network Topology

### `GET /api/scenarios` — Api Scenarios List

### `POST /api/scenarios/run` — Api Scenarios Run

**Тело запроса:**
- `application/json` — #/components/schemas/ScenarioRunBody

### `POST /api/sim-operator/accounts` — Create Accounts

**Тело запроса:**
- `application/json` — #/components/schemas/CreateAccountsRequest

### `POST /api/sim-operator/block-time` — Set Block Time

**Тело запроса:**
- `application/json` — #/components/schemas/BlockTimeRequest

### `POST /api/sim-operator/mint` — Mint Tokens

Как в узле: только tx mint из казначейства → мемпул → исполнение в блоке.

**Тело запроса:**
- `application/json` — #/components/schemas/MintRequest

### `POST /api/sim-operator/order` — Create Order

Те же проверки, что у кошелька (баланс, роль SELL).

**Тело запроса:**
- `application/json` — #/components/schemas/OrderRequest

### `POST /api/sim-operator/reset` — Reset State

### `POST /api/sim-operator/role` — Set Role

Те же правила, что у кошелька: set_role только через мемпул.

**Тело запроса:**
- `application/json` — #/components/schemas/RoleRequest

### `POST /api/sim-operator/speed` — Set Sim Speed

**Тело запроса:**
- `application/json` — #/components/schemas/SimSpeedRequest

### `GET /api/sim/consensus` — Api Get Consensus

### `POST /api/sim/consensus` — Api Set Consensus

Этап 3: модель сбоев консенсуса (PreVote/PreCommit/double_sign).

Все нули → однопропозерный fallback (поведение Этапа 2 и раньше).

**Тело запроса:**
- `application/json` — #/components/schemas/ConsensusFaultBody

### `GET /api/state` — Get State

### `GET /api/tx/{tx_hash}` — Api Get Tx

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `tx_hash` | path | string | ✓ |  |  |

### `GET /api/wallet/open-orders` — Wallet Open Orders

**Параметры:**

| name | in | type | required | default | description |
| ---- | -- | ---- | -------- | ------- | ----------- |
| `address` | query | string | ✓ |  |  |

### `POST /api/wallet/submit` — Wallet Submit Tx

**Тело запроса:**
- `application/json` — #/components/schemas/WalletSubmitBody

### `GET /metrics` — Api Metrics

