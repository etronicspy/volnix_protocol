# Volnix Simulator v2 — blockchain node (Python)

Off-chain node that **simulates the real chain** under working canon **5.2-sim**
([`../docs/volnix_protocol.md`](../docs/volnix_protocol.md)). Greenfield: it does
not wrap `simulation/` (v1). Go modules in `x/` are **not** the spec.

## What it is

- A single-process chain: mempool → BeginBlock → DeliverTx → EndBlock (PoVB §5.4,
  rewards, epoch §5.5) → CometBFT-style rounds (+2/3) → persist.
- **Source of truth:** append-only JSONL (`blocks.jsonl`). State is rebuilt by
  replay; `app_snapshot.json` is only a cache.
- Blocks carry a full CometBFT-like header (`data_hash`, `app_hash`,
  `validators_hash`, `last_results_hash`, `last_commit`, …). Tx hash is
  SHA-256 of canonical bytes. Merkle roots follow RFC 6962. Signatures are
  deterministic stubs (light fidelity).
- Amounts are **micro-units** (`SCALE = 1_000_000` = `UNIT`): 1 displayed token = 1e6.
- PoVB uses **`L_decl`** (sum of `L_i` over valid declares) for λ corridor and Fill;
  epoch emission uses network **`L_total`**. Subsidy and fees both split by **`b_i`**.

## Run

```bash
cd simulation_v2/backend
python3 -m pip install -r requirements-dev.txt
python3 main.py
```

Default listen: `http://127.0.0.1:8001`. Explorer contract: [`../docs/API.md`](../docs/API.md).

```bash
python3 -m pytest -q
```

## Layout

```
config/genesis.default.json   # do not rename to genesis.json (gitignored)
volnix/crypto/                # hash, Merkle, addresses, stub keys
volnix/types/                 # Header, Block, Tx, Msg*, ValidatorSet
volnix/store/                 # JSONL blockstore / results / tx index
volnix/app/modules/           # bank, ident, lizenz, anteil, povb, mint, epoch, gov
volnix/consensus/             # proposer-priority, rounds, evidence
volnix/node/                  # mempool + produce loop
volnix/api/                   # CometBFT RPC + /api/v1 + /ws
data/                         # runtime chain (gitignored)
```

## Environment (`VOLNIX_SIM2_`)

| Variable | Default | Meaning |
|----------|---------|---------|
| `DATA_DIR` | `backend/data` | JSONL stores |
| `GENESIS_PATH` | `config/genesis.default.json` | genesis template |
| `PORT` | `8001` | HTTP port |
| `PRODUCE_INTERVAL` | `1.0` | wall-clock seconds between blocks / Sim speed (`0.001`…`60`; live via `/api/v1/operator/produce-interval`; exposed on `/api/v1/chain/summary` as `produce_interval_sec` for traffic sync) |
| `AUTO_PRODUCE` | `true` | start the produce loop on boot |
| `AUTO_DECLARE` | `true` | enqueue genesis `MsgDeclareParticipation` each height (§6.3(5) `b/s`) so PoVB burns `f_i` + `b_i+s_i`; empty blocks burn nothing |
| `CORS_ORIGINS` | `*` | CORS |

**PoVB burns:** without a declare, EndBlocker does not burn ANT (canon §5.4). Entry fee
`f_i = ⌊α · L_i⌋` with genesis `α = 1/50` → **20 000** micro-ANT per valid declare
(at `L_i = 1 LZN`). Full §6.3(5) package burns **920 000** micro (`f+b+s`) when the set updates.

Canonical block time in the header is still `BaseBlockTime` (60s). Wall clock is independent.
The traffic process reads `produce_interval_sec` and adapts its poll / height catch-up so
economy ticks stay aligned with Sim speed.

## Genesis (canon §6.3)

One validator, no ZKP, **1 LZN** activated, **10 080 ANT**, **0 WRT** premint (§6.3),
`chain_id = volnix-sim-2`. WRT appears only via block subsidy.
Parameters: `EpochBlocks = 10080`, `HalvingInterval = 2100000`, `λ = 1/3`,
`α = 1/50`, `K = 150`, `max_active_suppliers = 108`,
`SUPPLIER_MIN_EPOCH_INCOME (X) = 1008 WRT` (constitutional, §5.6).
