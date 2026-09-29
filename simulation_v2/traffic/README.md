# Volnix Simulator v2 — Traffic / Economy Engine

Separate process that drives **network traffic and economic behaviour** against
the simulation_v2 node. It does **not** live inside `backend/volnix/node/`.

Canon: working stand **5.5-sim** ([`../docs/volnix_protocol.md`](../docs/volnix_protocol.md)).

## What it does

On each new block height (poll `/api/v1/chain/summary`):

1. **Adopt genesis** — seed `volnix-genesis-validator-v2` is enrichment agent #1 from tick 0.
2. **Citizen pool** — grow to 109 unverified wallets (no ZKP).
3. **Everyday spend** — *all* traffic wallets (citizens, enrichment, genesis) may send a small `bank/MsgSend` WRT each height (mempool → consensus). Imitation of spending money; Intensity is the share that spends (0 = none, 10 = everyone with spare WRT). When intensity > 0, at least `CITIZEN_TRANSFERS_PER_TICK` (clamped 2–4) wallets spend.
4. **Enrichment pool** — up to 50 independent agents (including genesis). Each has its own P&L forecast, competitive `(b_i, s_i)` declare, and may sample 5–10 peer strategy cards to copy the best WRT growth. First new ZKP is a supplier (floor 1+1).
   - **Validators (capacity race):** on-chain reward is still by **`b_i`** (§5.4); larger activated `L_i` only raises the declare ceiling. Agents treat public **`l_total`** (chain summary → market snapshot) as the network mass signal: each tick they **activate all free LZN**, refill an ANT buffer for declare, then **BUY LZN/WRT** (lot sized toward `l_total / n_validators`) while under the per-address freeze cap. Peer adopt never lowers `activate_ratio` (no deactivate).
5. **Role flip** — sell inventory, `MsgSend` WRT to a new wallet, verify the new role. Genesis cannot flip. Last supplier/validator cannot flip.

**Live path:** enrichment supervisor (`EnrichmentSupervisor` + `EnrichmentAgent`). Modules `BotEngine`, `AutoMarket`, and `AutoDeclare` are **legacy** — kept for unit tests only; they are not called from the default height loop.

**Pace sync:** the loop reads `produce_interval_sec` (= wall sleep after stand `time_scale`)
from the node summary (fallback `attempt_window_sec`) and sets
`effective_poll_sec = clamp(window × 0.25, 1 ms, configured poll)`. If several
heights arrive between polls, it **catch-up ticks** each missed height (cap 20 per loop
iteration). Frontend **Time scale** slider → node `/operator/time-scale`.

Signing is done by the node via `/api/v1/operator/*` (seed → keypair → mempool).

When traffic is running, set **`VOLNIX_SIM2_AUTO_DECLARE=false`** on the node so in-node
genesis auto-declare does not overwrite the agent's competitive `(b_i, s_i)`. Leave
`AUTO_DECLARE=true` only as a fallback when traffic is stopped.

## Run

Terminal 1 — node:

```bash
cd simulation_v2/backend
pip install -r requirements-dev.txt
# recommended with traffic:
# export VOLNIX_SIM2_AUTO_DECLARE=false
python3 main.py
```

Terminal 2 — traffic:

```bash
cd simulation_v2/traffic
pip install -r requirements.txt
python3 main.py
```

Control API (default): `http://127.0.0.1:8002`

Explorer **Traffic** page (`/operator`) is a read-only dashboard plus Intensity (`VITE_TRAFFIC_URL`, default `:8002`).

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/status` | running, height, pools, floor, genesis, `params`, agents[], last tick |
| POST | `/start` | `{ "intensity": 2.0 }` optional |
| POST | `/stop` | |
| POST | `/intensity` | `{ "intensity": 1.5 }` |
| GET | `/wallets` | local bot registry |
| POST | `/tick` | force one tick |
| POST | `/reset` | drop the local bot registry; chain state is unchanged |
| POST | `/release` | pause every bot except genesis; genesis only declares so blocks keep finalizing |

## Config

[`config/default.yaml`](config/default.yaml) plus env prefix **`VOLNIX_SIM2_TRAFFIC_`**:

| Variable | Default | Meaning |
|----------|---------|---------|
| `NODE_URL` | `http://127.0.0.1:8001` | Node base URL |
| `CONTROL_PORT` | `8002` | Control HTTP port |
| `AUTOSTART` | `true` | Start height loop on boot |
| `TARGET_ENRICHMENT_BOTS` | `50` | Active S/V agents including genesis |
| `TARGET_CITIZENS` | `109` | Spontaneous WRT pool |
| `CITIZEN_TRANSFERS_PER_TICK` | `3` | Min MsgSend count per height when intensity > 0 (clamped 2–4) |
| `HORIZON_BLOCKS` | `12` | Per-agent P&L horizon |
| `PEER_SAMPLE_MIN` / `MAX` | `5` / `10` | Peer strategy cards sampled each tick |
| `MIN_SUPPLIERS` / `MIN_VALIDATORS` | `1` / `1` | Flip floor |
| `ROLE_FLIP_MARGIN_WRT` | `5000000` | Net-WRT gap before considering flip |
| `FLIP_CONFIRM_TICKS` | `3` | Consecutive ticks of flip signal before acting |
| `MAX_NEW_PER_TICK` | `3` | Max new citizens + enrichment wallets per height |
| `BOOTSTRAP_WRT` | `5000000` | Mint to new wallets |
| `MAX_ACTIONS_PER_TICK` | `40` | Cap on everyday spenders per height |
| `ENABLE_ROLE_FLIP` | `true` | Allow enrichment role flips |
| `GENESIS_SEED` | `volnix-genesis-validator-v2` | Adopted on start |
| `INTENSITY` | `2.0` | Spend share 0–10 (status / start / `/intensity`) |
| `POLL_INTERVAL_SEC` | `0.25` | Max poll when chain is slow |
| `ENABLE_MARKET` / `ENABLE_BOTS` / `ENABLE_DECLARE` | `true` | Daemon toggles |

## Funding note

Genesis has **no WRT premint** (§6.3). Bootstrap mint uses `/operator/mint` from the genesis validator after it earns block subsidy. Traffic declares genesis itself — keep node `VOLNIX_SIM2_AUTO_DECLARE=false` while this process is up.

## Tests

```bash
cd simulation_v2/traffic
python3 -m pytest -q
```
