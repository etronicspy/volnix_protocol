# Volnix Simulator v2 — blockchain node (Python)

Off-chain node that **simulates the real chain** under working canon **5.5-sim**
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
- **Adaptive attempt window (§6.2):** declare acceptance lasts current `T` (60→…→1 s).
  Empty attempts accumulate `missed_budget` and halve `T`; catch-up debt
  `⌊missed/60⌋` blocks at `T=1`, then reset to 60 s.

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
volnix/node/                  # mempool + produce loop + AttemptPace
volnix/api/                   # CometBFT RPC + /api/v1 + /ws
data/                         # runtime chain (gitignored)
```

## Environment (`VOLNIX_SIM2_`)

| Variable | Default | Meaning |
|----------|---------|---------|
| `DATA_DIR` | `backend/data` | JSONL stores |
| `GENESIS_PATH` | `config/genesis.default.json` | genesis template |
| `PORT` | `8001` | HTTP port |
| `AUTO_PRODUCE` | `true` | start the produce loop on boot (pace from §6.2, not a fixed interval) |
| `AUTO_DECLARE` | `true` | stand helper: enqueue genesis `MsgDeclareParticipation` each height. **Required** for emission without traffic: canon 5.5-sim has **no** last_applied replay — without a fresh declare tx the height is **not finalized** (empty attempt shrinks `T`). Set **`false`** when `simulation_v2/traffic` owns `(b_i, s_i)` |
| `TIME_SCALE` | `60` | stand-only: `wall_sleep = attempt_window / time_scale` (`1`…`3600`; live via `/api/v1/operator/time-scale`). Default 60 → canon minute ≈ 1 s wall-clock |
| `CORS_ORIGINS` | `*` | CORS |

**PoVB burns (5.5-sim):** a height commits only with **new** `MsgDeclareParticipation` in the block (`f_i + b_i + s_i` paid from ANT). Without declares the
node does **not** finalize the height (empty attempt). Entry fee `f_i = ⌊α · L_i⌋` with genesis
`α = 1/50` → **20 000** micro-ANT (at `L_i = 1 LZN`). Full §6.3(5) package burns
**920 000** micro (`f+b+s`) when the set updates.

Canonical time in the header advances `BaseBlockTime` (60s) per finalized height.
Canonical attempt window `T` is adaptive (§6.2). Stand `time_scale` only compresses wall-clock
(`wall_sleep_sec` / compat `produce_interval_sec` on `/api/v1/chain/summary`). Traffic polls that field.

## Genesis (canon §6.3)

One validator, no ZKP, **1 LZN** activated, **10 080 ANT**, **0 WRT** premint (§6.3),
`chain_id = volnix-sim-2`. WRT appears only via block subsidy.
Parameters: `EpochBlocks = 10080`, `HalvingInterval = 2100000`, `λ = 1/3`,
`α = 1/50`, `K = 150`, `max_active_suppliers = 108`,
`SUPPLIER_MIN_EPOCH_INCOME (X) = 1008 WRT` (constitutional, §5.6).
