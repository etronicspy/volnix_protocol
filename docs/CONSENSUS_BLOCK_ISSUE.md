# Проблема: блоки не создаются (высота застревает на 1)

## Симптомы

- `latest_block_height` = 1 (genesis block) при 2+ валидаторах
- Консенсус застревает на height 2, round 5–60+, step 6 (RoundStepCommit)
- `dump_consensus_state` показывает: prevotes/precommits есть, но за **разные block_id**

## Выводы (2026-02-28)

### Работает
- **1 валидатор** — блоки создаются (height 393 за ~25 сек)
- **vote_extensions_enable_height = "0"** — в genesis (строка!) отключает vote extensions
- **unsafe = true** в RPC — для `dump_consensus_state`

### Не работает
- **2+ валидатора** — в каждом раунде голосуют за разные block_id → нет 2/3 precommits

### Внедрённые правки (без эффекта)
1. **NoOpPrepareProposal** — детерминированная передача req.Txs
2. **VOLNIX_SKIP_VALIDATOR_UPDATES=1** — отключение ValidatorUpdates в EndBlocker для тестнета

### Причина (подтверждено 2026-02-28)

**Десинхронизация по раундам.** Логи `[CONSENSUS_DEBUG]` показали:

- **Height 1:** все ноды получили один блок (927A8636...), proposer node1 — OK.
- **Height 2:** каждая нода вызывает PrepareProposal и ProcessProposal только для **своего** блока:
  - node0: block 54D92648 (proposer=B8FD)
  - node1: block 218B246E (proposer=0A53)
  - node2: block 2946E481 (proposer=9D33)
  - node3: block 740422F8 (proposer=AF30)

Каждая нода получает ProcessProposal только когда **она сама** proposer. Блоки от других нод не доходят до ProcessProposal. P2P соединения установлены (n_peers=3), genesis идентичен (md5).

**Вывод:** ноды в разных раундах. Когда node0 proposer в round 0, node1/2/3 уже в round 1/2/3 (таймаут до получения блока). Каждая нода создаёт свой блок в своём раунде и не получает блоки других — они приходят «слишком поздно» (для другого раунда).

## Что уже пробовали

1. **vote_extensions_enable_height = "0"** (строка в genesis)
2. **NoOpPrepareProposal** — без эффекта
3. **VOLNIX_SKIP_VALIDATOR_UPDATES=1** — без эффекта
4. **skip_timeout_commit = true**, **timeout_propose = "10s"** — без эффекта
5. **create_empty_blocks = true**, **allow_duplicate_ip = true**

## Рекомендуемые шаги

1. **Временный workaround:** 1 валидатор для dev/test
2. **Запуск всех нод одновременно** — в течение 1–2 сек (см. `scripts/testnet-start-all.sh`)
3. **Увеличение timeout_propose** — дать время на распространение блока (уже 10s в config)
4. **CometBFT source:** проверить block creation и broadcast в v0.38

## Текущая конфигурация

- `vote_extensions_enable_height`: "0" (строка в genesis)
- `create_empty_blocks`: true
- `unsafe`: true (RPC)
- `VOLNIX_SKIP_VALIDATOR_UPDATES=1` для тестнета (см. scripts/testnet-reset-and-start.sh)

## P2P верификация (2026-02-28)

Скрипт `scripts/testnet-verify-p2p.sh` проверяет:

- **Node IDs** — соответствие `node_key.json` и `persistent_peers`
- **persistent_peers** — формат `node_id@127.0.0.1:port`, порты 26656/26666/26676/26686
- **unconditional_peer_ids** — те же ID, что в persistent_peers
- **allow_duplicate_ip = true** — обязательно для localhost (все ноды на 127.0.0.1)
- **addr_book_strict = false** — для приватной сети

Результат проверки: конфигурация корректна, n_peers=3 при запуске. Проблема не в P2P.

## Sequential start (2026-02-28)

Скрипт `scripts/testnet-sequential-start.sh`: node0 сначала одна (genesis с 1 валидатором), затем node1.

**Исправлено:** AppHash не совпадал из‑за недетерминированности в `x/consensus/keeper`:
- `timestamppb.Now()` → `timestamppb.New(ctx.BlockTime())` (GetConsensusState, CreateBlindAuction, CommitBid, SelectAuctionWinner)
- `rand.Intn`/`rand.Uint64` → `deterministicRand(ctx)` на основе block height + block time

**Результат после фикса:** node1 успешно синхронизируется с node0 (height 280 за 30с при node0 на 827).
