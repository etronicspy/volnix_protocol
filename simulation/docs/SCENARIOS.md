# YAML-сценарии симуляции

Сценарий — это детерминированный, перезапускаемый прогон шагов
(`mint`, `set_role`, `order`, `wait_blocks`, …) над инициализированной
симуляцией с финальными ассертами на её состояние.

## Где живут

- Каталог по умолчанию: `simulation/scenarios/`
- Каждый файл — YAML с расширением `.yaml`.

## Структура

```yaml
name: epoch_wipe_stress         # имя в отчёте
description: |                  # любой свободный текст
  Краткое описание сценария.
seed: 42                        # необязательно, фиксирует random
steps:                          # список шагов; каждый шаг — {action: {args}}
  - set_block_time: { seconds: 0.5 }
  - create_account: { address: alice, role: citizen, zkp: true }
  - mint: { receiver: alice, amount: 100, asset: wrt }
  - bot_start: { intensity: 5, probe_ratio: 0.1 }
  - wait_blocks: { n: 30 }
  - bot_stop: {}
asserts:                        # проверки после всех шагов
  - balance: { address: alice, asset: wrt, op: ">=", value: 100 }
  - block_height: { op: ">=", value: 30 }
  - canon_log: { status: reject, canon: "§5.4", min_count: 0 }
  - mempool_size: { op: "<=", value: 1000 }
```

## Поддерживаемые шаги (`steps`)

Реализованы в `simulation/backend/core/scenarios.py`
(`ScenarioRunner._step_<name>`):

| step                 | args                                                                                                | что делает |
| -------------------- | --------------------------------------------------------------------------------------------------- | ---------- |
| `create_account`     | `address` (str), `role` (citizen/provider/validator, по умолчанию citizen), `zkp` (bool)            | гарантирует наличие аккаунта (без tx) |
| `mint`               | `receiver` (str), `amount` (float), `asset` (`wrt`/`lzn`/`ant`, по умолч. `wrt`)                    | минт из казны через `validate_treasury_mint` |
| `set_role`           | `address` (str), `role` (str)                                                                       | через `validate_and_build_tx("set_role", …)` |
| `verify_zkp`         | `address` (str)                                                                                     | ставит `zkp_verified = true` (tx) |
| `transfer`           | `sender` (str), `to` (str), `amount` (float), `asset` (по умолч. `wrt`)                             | tx-перевод |
| `order`              | `address`, `side` (`buy`/`sell`), `amount`, `price?`, `market?` (bool), `max_wrt?`                  | через `validate_and_build_tx("create_order", …)` |
| `cancel_order`       | `address`, `order_id`                                                                               | tx отмены |
| `declare`            | `address`, `burn` (float), `stake?` (float)                                                         | §5.4 declare participation |
| `wait_blocks`        | `n` (int, по умолч. 1)                                                                              | вызывает `engine.produce_block()` N раз |
| `set_block_time`     | `seconds` (float, 0.1..300)                                                                         | меняет интервал блока симуляции |
| `bot_start`          | `intensity?` (float), `probe_ratio?` (float), `enable_probes?` (bool)                               | конфигурирует BotEngine (без асинхр. цикла — для прогона) |
| `bot_stop`           | —                                                                                                   | `bot.stop()` |
| `bot_tick`           | `n?` (int)                                                                                          | детерминированно вызывает `bot.generate_traffic()` N раз |
| `consensus`          | `p_absent?`, `p_nil?`, `p_double_sign?`, `seed?`                                                    | конфигурирует FaultModel для §6.1 |
| `auto_declare`       | —                                                                                                   | один тик `AutoDeclareDaemon.step_once` (canon §5.4 declare за каждого валидатора в `consensus_validator_set`) |
| `network_set`        | `num_nodes?` (int, при отсутствии NetworkSim — первый вызов с ≥2 поднимает NetworkSim), `latency_ms?` (int), `loss_pct?` (float 0..100), `quorum_pct?` (0..1) | конфигурирует NetworkSim (§6.3 multi-node sim revive) |

## Ассерты (`asserts`)

| assert key          | поля                                                                              | проверяет |
| ------------------- | --------------------------------------------------------------------------------- | --------- |
| `balance`           | `address`, `asset` (`wrt`/`lzn`/`ant`/`frozen`), `op` (`==`/`!=`/`>=`/`<=`/`>`/`<`), `value` | баланс кошелька |
| `block_height`      | `op`, `value`                                                                     | текущая высота |
| `mempool_size`      | `op`, `value`                                                                     | размер мемпула |
| `canon_log`         | `status?`, `canon?`, `min_count?`                                                 | количество записей в canon-аудите |

Все ассерты выполняются после `steps` (не между ними); результат каждого
включается в `ScenarioReport.asserts` (`passed`, `actual`, `expected`,
`detail`).

## Запуск

### CLI

```bash
cd simulation/backend
python -m core.scenarios list                    # список сценариев
python -m core.scenarios run epoch_wipe_stress.yaml
```

Код возврата `0` — все ассерты прошли, `1` — есть провалы / ошибка.

### REST

```bash
# Список встроенных сценариев
curl http://localhost:8000/api/scenarios

# Запуск по имени файла в каталоге scenarios/
curl -X POST http://localhost:8000/api/scenarios/run \
  -H 'Content-Type: application/json' \
  -d '{"path": "epoch_wipe_stress.yaml", "reset_state": true}'

# Inline YAML (без файла)
curl -X POST http://localhost:8000/api/scenarios/run \
  -H 'Content-Type: application/json' \
  -d '{"yaml": "name: foo\nsteps: []\nasserts: []", "reset_state": false}'
```

### Из UI

Вкладка **R&D / KPI** → блок «YAML scenarios»: выбрать имя сценария,
галочку `reset_state` и нажать «run». Краткий отчёт со списком
ассертов появится прямо в панели.

## Встроенные сценарии

- `epoch_wipe_stress.yaml` — поведение поставщика на границе эпохи §5.5
  (wipe ANT → эмиссия).
- `validator_coalition_burn.yaml` — declare валидатора в пределах
  λ·L_total (§5.4).
- `provider_market_dump.yaml` — sell-сценарий поставщика и встречная
  покупка валидатора (§5.1).
- `slashing_double_sign.yaml` — двойная подпись + slashing (§6.1).
- `network_partition.yaml` — NetworkSim 4 узла + переключение в режим высокой gossip-латентности и потерь; блоки всё ещё рождаются за счёт кворума ≥ 2/3 узлов.
- `validator_set_growth.yaml` — мульти-нодная конфигурация (4 узла) + auto_declare; smoke-тест роста цепочки в условиях NetworkSim.

## Как добавить новый

1. Положите YAML в `simulation/scenarios/`.
2. Опционально — добавьте `pytest`-тест в
   `simulation/backend/tests/test_scenarios.py`, чтобы CI запускал его
   при каждом PR.
3. При необходимости — расширьте набор `_step_*` или `_run_assert` в
   `core/scenarios.py`.
