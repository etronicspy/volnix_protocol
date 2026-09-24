# Volnix Simulator v2 — Traffic / Economy Engine

Separate process that drives **network traffic and economic behaviour** against
the simulation_v2 node. It does **not** live inside `backend/volnix/node/`.

Canon: working stand **5.2-sim** ([`../docs/volnix_protocol.md`](../docs/volnix_protocol.md)).

## What it does

On each new block height (poll `/api/v1/chain/summary`):

1. **AutoMarket** — suppliers post ANT asks; validators refill ANT under a miner reservation price.
2. **BotEngine** — grows a pool of `bot-*` wallets, stub ZKP (`stub-zkp-{uuid}`), WRT transfers, market orders, LZN activate, profit-driven first role choice.
3. **AutoDeclare** — bot validators submit `MsgDeclareParticipation` (`b_i ≈ λ·L_i`).

**Sim speed sync:** the loop reads `produce_interval_sec` from the node summary and sets
`effective_poll_sec = clamp(produce_interval × 0.25, 1 ms, configured poll)`. If several
heights arrive between polls, it **catch-up ticks** each missed height (cap 20 per loop
iteration) so `intensity` (actions per height) is not silently dropped. Frontend **Sim speed**
slider only changes the node produce interval; traffic follows automatically.

Signing is done by the node via `/api/v1/operator/*` (seed → keypair → mempool).

## Run

Terminal 1 — node:

```bash
cd simulation_v2/backend
pip install -r requirements-dev.txt
python3 main.py
```

Terminal 2 — traffic:

```bash
cd simulation_v2/traffic
pip install -r requirements.txt
python3 main.py
```

Control API (default): `http://127.0.0.1:8002`

Explorer **Operator** page (`/operator`) includes a Traffic panel that talks to this
API (`VITE_TRAFFIC_URL`, default `:8002`). CORS is open for local stand use.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/status` | running, height, intensity, `produce_interval_sec`, `effective_poll_sec`, roles |
| POST | `/start` | `{ "intensity": 2.0 }` optional |
| POST | `/stop` | |
| POST | `/intensity` | `{ "intensity": 1.5 }` |
| GET | `/wallets` | local bot registry |
| POST | `/tick` | force one tick |

## Config

[`config/default.yaml`](config/default.yaml) plus env prefix **`VOLNIX_SIM2_TRAFFIC_`**:

| Variable | Default | Meaning |
|----------|---------|---------|
| `NODE_URL` | `http://127.0.0.1:8001` | Node base URL |
| `CONTROL_PORT` | `8002` | Control HTTP port |
| `AUTOSTART` | `true` | Start height loop on boot |
| `INTENSITY` | `2.0` | Bot actions per height tick |
| `POLL_INTERVAL_SEC` | `0.25` | Max poll when paused / chain is slow (fast chains use a shorter effective poll) |
| `TARGET_WALLETS` | `30` | Bot pool size |
| `ENABLE_MARKET` / `ENABLE_BOTS` / `ENABLE_DECLARE` | `true` | Daemon toggles |

## Funding note

Genesis has **no WRT premint** (§6.3). Bootstrap mint uses `/operator/mint` from the genesis validator after it earns block subsidy. Keep in-node `VOLNIX_SIM2_AUTO_DECLARE=true` (or let traffic declare bots) so PoVB burns and rewards continue.

## Tests

```bash
cd simulation_v2/traffic
python3 -m pytest -q
```

## Phase 2 (not in MVP)

- Role migration (`MsgMigrateRole`) as primary role flip
- Negative canon probes
- Frontend panel wiring to `:8002`
