# Проверка трёхузловой сети Volnix

**Дата проверки:** 2026-02-28

## Принцип работы (по стандарту Cosmos/CometBFT)

1. **persistent_peers** — узлы подключаются к указанным пирам
2. **seeds** — опционально, для первоначального обнаружения
3. **PEX (Peer Exchange)** — обмен адресами между подключёнными узлами
4. **addrbook** — кэш обнаруженных адресов

---

## Результаты проверки

### Текущее состояние узлов

| Узел  | RPC    | Пиры | Подключён к | Block height |
|-------|--------|------|-------------|--------------|
| node0 | 26657  | **0** | —           | 0            |
| node1 | 26667  | 1    | node2       | 0            |
| node2 | 26677  | 1    | node1       | 0            |

### Что работает ✅

| Компонент | Статус | Детали |
|-----------|--------|--------|
| **persistent_peers** | ✅ | node1 ↔ node2 успешно соединяются через persistent_peers |
| **Загрузка config.toml** | ✅ | Порты и persistent_peers читаются из файла |
| **PEX** | ✅ | Включён (`pex = true`) в config |
| **addrbook** | ✅ | Адреса сохраняются в `config/addrbook.json` |
| **addr_book_strict** | ✅ | `false` — 127.0.0.1 принимается |

### Что не работает ❌

| Проблема | Детали |
|----------|--------|
| **node0 изолирован** | 0 пиров. Ошибка при подключении: `auth failure: secret conn failed: EOF` |
| **Блоки не создаются** | Все узлы на height 0. Для консенсуса нужны 2/3 валидаторов (2 из 3). node1 и node2 соединены, но блоки не производятся |

### addrbook

- **node0**: знает node1 и node2 (из persistent_peers), но `last_success` никогда не наступал
- **node1**: знает node0, node2 — адреса получены (в т.ч. через PEX от node2)

---

## Вывод

**Принцип работает частично:**
- persistent_peers, PEX и addrbook работают как задумано
- node1 и node2 образуют связную пару
- node0 не может установить соединение с node1/node2 (auth failure)

**Причина (исправлено):** `allow_duplicate_ip = false` — при localhost все узлы имеют IP 127.0.0.1. CometBFT отклонял второе подключение с того же IP. Третий узел (node0) не мог подключиться к node1/node2, т.к. они уже были соединены друг с другом (оба с 127.0.0.1).

**Исправление:** в config.toml для localhost:
- `allow_duplicate_ip = true`
- `persistent_peers_max_dial_period = "30s"` — узлы не сдаются при переподключении
- `unconditional_peer_ids` — ID всех persistent peers

После исправления **новые узлы могут подключаться в любой момент** (late joiner).

**Сброс для проверки:** `./scripts/testnet-reset-and-start.sh`

---

## Конфигурация (соответствует принципу)

```
node0: persistent_peers = "node1_id@127.0.0.1:26666,node2_id@127.0.0.1:26676"
node1: persistent_peers = "node0_id@127.0.0.1:26656,node2_id@127.0.0.1:26676"
node2: persistent_peers = "node0_id@127.0.0.1:26656,node1_id@127.0.0.1:26666"
seeds = "" (не используется)
pex = true
```
