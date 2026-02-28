# План настройки трёх узлов Volnix Protocol

**Дата:** 2026-02-28

Этот план описывает пошаговую настройку трёх узлов на одном устройстве (для разработки/тестирования) или на трёх разных устройствах.

---

## Предварительные требования

- Собран бинарник: `./build/volnixd`
- Реализована загрузка конфига из `config.toml` при старте
- Команда `volnixd tendermint show-node-id` доступна

---

## Вариант A: Три узла на одном устройстве

### Шаг 1. Сборка и подготовка директорий

```bash
cd /path/to/helvetia_protocol
make build
# или: go build -o build/volnixd ./cmd/volnixd

mkdir -p testnet/node0 testnet/node1 testnet/node2
```

### Шаг 2. Инициализация узлов

```bash
./build/volnixd init node0 -home testnet/node0
./build/volnixd init node1 -home testnet/node1
./build/volnixd init node2 -home testnet/node2
```

### Шаг 3. Получение Node ID

```bash
./build/volnixd tendermint show-node-id -home testnet/node0
./build/volnixd tendermint show-node-id -home testnet/node1
./build/volnixd tendermint show-node-id -home testnet/node2
```

Сохраните вывод — это NODE0_ID, NODE1_ID, NODE2_ID.

### Шаг 4. Genesis с тремя валидаторами

1. Соберите pubkeys всех валидаторов:

```bash
cat testnet/node0/config/priv_validator_key.json | jq '.pub_key'
cat testnet/node1/config/priv_validator_key.json | jq '.pub_key'
cat testnet/node2/config/priv_validator_key.json | jq '.pub_key'
```

2. Откройте `testnet/node0/config/genesis.json` и в массив `validators` добавьте валидаторов из node1 и node2. Формат каждого:

```json
{
  "address": "<address из priv_validator_key>",
  "pub_key": { "type": "tendermint/PubKeyEd25519", "value": "<base64>" },
  "power": "10",
  "name": "node1"
}
```

3. Скопируйте обновлённый genesis на все узлы:

```bash
cp testnet/node0/config/genesis.json testnet/node1/config/genesis.json
cp testnet/node0/config/genesis.json testnet/node2/config/genesis.json
```

### Шаг 5. Редактирование config.toml

Порты для трёх узлов на одной машине:

| Узел  | P2P   | RPC   |
|-------|-------|-------|
| node0 | 26656 | 26657 |
| node1 | 26666 | 26667 |
| node2 | 26676 | 26677 |

**testnet/node0/config/config.toml** — секция `[p2p]` и `[rpc]`:

```toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
external_address = ""
persistent_peers = "NODE1_ID@127.0.0.1:26666,NODE2_ID@127.0.0.1:26676"
addr_book_strict = false
allow_duplicate_ip = true
persistent_peers_max_dial_period = "30s"
unconditional_peer_ids = "NODE1_ID,NODE2_ID"

[rpc]
laddr = "tcp://0.0.0.0:26657"
```

**testnet/node1/config/config.toml**:

```toml
[p2p]
laddr = "tcp://0.0.0.0:26666"
external_address = ""
persistent_peers = "NODE0_ID@127.0.0.1:26656,NODE2_ID@127.0.0.1:26676"
addr_book_strict = false
allow_duplicate_ip = true
persistent_peers_max_dial_period = "30s"
unconditional_peer_ids = "NODE0_ID,NODE2_ID"

[rpc]
laddr = "tcp://0.0.0.0:26667"
```

**testnet/node2/config/config.toml**:

```toml
[p2p]
laddr = "tcp://0.0.0.0:26676"
external_address = ""
persistent_peers = "NODE0_ID@127.0.0.1:26656,NODE1_ID@127.0.0.1:26666"
addr_book_strict = false
allow_duplicate_ip = true
persistent_peers_max_dial_period = "30s"
unconditional_peer_ids = "NODE0_ID,NODE1_ID"

[rpc]
laddr = "tcp://0.0.0.0:26677"
```

Замените NODE0_ID, NODE1_ID, NODE2_ID на реальные значения из шага 3.

**Обязательно для localhost (все узлы на 127.0.0.1):**
```toml
addr_book_strict = false
allow_duplicate_ip = true
persistent_peers_max_dial_period = "30s"
unconditional_peer_ids = "NODE1_ID,NODE2_ID"
```
- `addr_book_strict = false` — CometBFT примет 127.0.0.1
- `allow_duplicate_ip = true` — **критично**: без этого третий узел не подключится (все с одного IP)
- `persistent_peers_max_dial_period = "30s"` — узлы не сдаются при переподключении (для late joiners)
- `unconditional_peer_ids` — ID остальных узлов (для node0: node1, node2; для node1: node0, node2; для node2: node0, node1)

### Шаг 6. Запуск

В трёх терминалах:

```bash
# Терминал 1
./build/volnixd start -home testnet/node0

# Терминал 2
./build/volnixd start -home testnet/node1

# Терминал 3
./build/volnixd start -home testnet/node2
```

### Шаг 7. Проверка

```bash
curl http://localhost:26657/net_info | jq '.result.n_peers'   # node0, ожидание: 2
curl http://localhost:26667/net_info | jq '.result.n_peers'   # node1
curl http://localhost:26677/net_info | jq '.result.n_peers'   # node2

curl http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
```

### Позднее подключение (late joiner)

С `allow_duplicate_ip = true` и `persistent_peers_max_dial_period = "30s"` **новый узел может подключиться в любой момент**. Запустите node0 через 5 минут после node1/node2 — он подключится и синхронизирует блокчейн. Высота блоков не влияет на подключение.

### Сброс и перезапуск

```bash
./scripts/testnet-reset-and-start.sh
# Затем запустите узлы в трёх терминалах
```

---

## Добавление четвёртого узла (node3)

Если сеть уже настроена с node0, node1, node2:

```bash
# 1. Инициализация
./build/volnixd init node3 --home testnet/node3

# 2. Node ID
./build/volnixd tendermint show-node-id --home testnet/node3  # NODE3_ID

# 3. Добавить валидатора в genesis
cat testnet/node3/config/priv_validator_key.json | jq '.pub_key, .address'
# Добавить в validators genesis.json на всех узлах, скопировать genesis на node3

# 4. config.toml для node3
# [p2p] laddr = "tcp://0.0.0.0:26686"
# [rpc] laddr = "tcp://0.0.0.0:26687"
# persistent_peers = "NODE0_ID@127.0.0.1:26656,NODE1_ID@127.0.0.1:26666,NODE2_ID@127.0.0.1:26676"
# addr_book_strict = false, allow_duplicate_ip = true, persistent_peers_max_dial_period = "30s"
# unconditional_peer_ids = "NODE0_ID,NODE1_ID,NODE2_ID"

# 5. Обновить node0, node1, node2: добавить NODE3_ID@127.0.0.1:26686 в persistent_peers и unconditional_peer_ids

# 6. Запуск
VOLNIX_GRPC_PORT=9093 ./build/volnixd start --home testnet/node3
```

| Узел  | P2P   | RPC   | gRPC |
|-------|-------|-------|------|
| node0 | 26656 | 26657 | 9090 |
| node1 | 26666 | 26667 | 9091 |
| node2 | 26676 | 26677 | 9092 |
| node3 | 26686 | 26687 | 9093 |
| node4 (full) | 26696 | 26697 | 9094 |
| node5 (full) | 26706 | 26707 | 9095 |
| node6 (full) | 26716 | 26717 | 9096 |

### Добавление full node без перезапуска

Пока валидаторы (node0–node3) работают:

```bash
./build/volnixd init nodeN --home testnet/nodeN
cp testnet/node0/config/genesis.json testnet/nodeN/config/genesis.json
# config.toml: laddr P2P/RPC (уникальные порты), persistent_peers = node0,1,2,3
# addr_book_strict=false, allow_duplicate_ip=true, persistent_peers_max_dial_period="30s"
VOLNIX_GRPC_PORT=909N ./build/volnixd start --home testnet/nodeN
```

Пример портов: node4 (26696/26697), node5 (26706/26707), node6 (26716/26717).

---

## Вариант B: Три узла на разных устройствах

### Шаг 1–3. Аналогично варианту A

На каждом устройстве: сборка, init, получение Node ID.

### Шаг 4. Genesis

Соберите genesis на первом узле (node0), добавьте всех валидаторов, скопируйте `genesis.json` на устройства node1 и node2.

### Шаг 5. config.toml на каждом устройстве

**Node 0** (IP_NODE0 — публичный IP или hostname первого устройства):

```toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
external_address = "IP_NODE0:26656"
persistent_peers = "NODE1_ID@IP_NODE1:26656,NODE2_ID@IP_NODE2:26656"

[rpc]
laddr = "tcp://0.0.0.0:26657"
```

**Node 1** (IP_NODE1):

```toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
external_address = "IP_NODE1:26656"
persistent_peers = "NODE0_ID@IP_NODE0:26656,NODE2_ID@IP_NODE2:26656"

[rpc]
laddr = "tcp://0.0.0.0:26657"
```

**Node 2** (IP_NODE2):

```toml
[p2p]
laddr = "tcp://0.0.0.0:26656"
external_address = "IP_NODE2:26656"
persistent_peers = "NODE0_ID@IP_NODE0:26656,NODE1_ID@IP_NODE1:26656"

[rpc]
laddr = "tcp://0.0.0.0:26657"
```

На каждом устройстве порт P2P — 26656, RPC — 26657. Порт 26656 должен быть открыт в фаерволе.

### Шаг 6–7. Запуск и проверка

Запустите узлы на каждом устройстве. Проверьте подключение:

```bash
curl http://IP_NODE0:26657/net_info | jq '.result.n_peers'
```

---

## Чеклист

- [ ] Сборка `volnixd`
- [ ] Инициализация трёх узлов
- [ ] Получение Node ID для каждого
- [ ] Genesis с тремя валидаторами, скопирован на все узлы
- [ ] config.toml: порты, persistent_peers, external_address (для варианта B)
- [ ] Запуск узлов
- [ ] Проверка: n_peers > 0, блоки создаются

---

## Примечание по gRPC

Текущая реализация слушает gRPC на порту 9090. При трёх узлах на одной машине будет конфликт портов. Для локальной разработки можно использовать только node0 для gRPC (backend API подключается к localhost:9090). Для полной поддержки потребуется доработка — передача порта gRPC через config.
