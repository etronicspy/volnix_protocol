# Simulator v2 API

Base URL (default): `http://127.0.0.1:8001`.

Two surfaces share the same node:

1. **CometBFT-compatible RPC** at the root (`/status`, `/block`, …).
2. **Explorer REST** under `/api/v1/` — primary contract for the frontend.
3. **WebSocket** `/ws` for live `new_block` / `new_tx` / `validator_set_update` / `epoch_boundary`.

Amounts in responses are **micro-units** unless a `*_display` field is present
(divide by `1_000_000`).

---

## Explorer `/api/v1`

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/chain/summary` | Height, supplies, `L_total`, epoch/halving countdown, λ/α/K, wall-clock `produce_interval_sec` + `auto_produce` |
| GET | `/blocks?tail=100` | Block tape (+ PoVB summary: `set_updated`, `n_declares`, `n_passed`, `sum_b`, `fill`, `l_decl`, `b_min`/`b_max`) |
| GET | `/blocks/{height}` | Block + `block_results` + `consensus` (signing/next set, commit for H, PoVB, voted/total power) |
| GET | `/txs` | Recent transactions |
| GET | `/txs/{hash}` | Decoded tx + result events |
| GET | `/accounts` | All wallets: role, set membership, balances (`?role=` optional) |
| GET | `/accounts/{address}` | Role, balances, activated LZN, last activity, open orders |
| GET | `/accounts/{address}/txs` | Account history |
| GET | `/validators` | Current set: `L_i`, `w_i`, power, proposer |
| GET | `/validators/{address}` | One validator + account |
| GET | `/povb/{height}` | Declares, `f_i`, λ corridor, Fill, exclusions, top-K |
| GET | `/epochs` | Epoch records |
| GET | `/epochs/{n}` | Wipe / emit for one epoch |
| GET | `/market/orderbook?market=ANT/WRT` | Bids / asks (`ANT/WRT` or `LZN/WRT`) |
| GET | `/market/trades` | Recent `anteil.trade_executed` |
| GET | `/params` | Live DAO/genesis parameters |
| GET | `/supply` | WRT / LZN / ANT aggregates |
| GET | `/search?q=` | Height, tx hash, block hash, or address |
| POST | `/tx` | Broadcast a signed tx (`{ "tx": {…} }`) |

### Operator helpers (stand bootstrap, not protocol messages)

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/operator/account` | Derive `volnix1…` from a seed |
| POST | `/operator/mint` | `MsgSend` WRT from genesis (must already hold subsidy; no premint) |
| POST | `/operator/verify` | Sign + mempool `MsgVerifyIdentity` |
| POST | `/operator/declare` | Sign + mempool `MsgDeclareParticipation` |
| POST | `/operator/order` | Sign + mempool `MsgPlaceOrder` |
| POST | `/operator/tx?seed=` | Sign + mempool arbitrary message list (body = `[{type,…}]`) |
| POST | `/operator/produce` | Produce `count` blocks now |
| GET/POST | `/operator/produce-interval` | Wall-clock sim speed / block rate (`interval_sec` ∈ `[0.001, 60]`); mirrored in `/chain/summary` as `produce_interval_sec` |
| GET/POST | `/operator/consensus` | Fault model: `absent` / `nil_vote` |

---

## Traffic process (`simulation_v2/traffic/`)

Separate economy / bot process. Default control URL: `http://127.0.0.1:8002`.
Talks to the node operator + explorer APIs; env prefix `VOLNIX_SIM2_TRAFFIC_*`.
See [`../traffic/README.md`](../traffic/README.md).

Pace: traffic **follows** node `produce_interval_sec` from `/chain/summary` — adaptive poll and
per-height catch-up so economy ticks stay synced with Sim speed (frontend slider →
`/operator/produce-interval`). `intensity` remains actions **per height**.

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/status` | running, height, intensity, `produce_interval_sec`, `effective_poll_sec`, wallet roles, last tick |
| POST | `/start` | optional `{ "intensity": 2.0 }` |
| POST | `/stop` | pause height loop |
| POST | `/intensity` | `{ "intensity": 1.5 }` |
| GET | `/wallets` | local bot registry (seeds / roles) |
| POST | `/tick` | force one market→bots→declare tick |

---

## CometBFT RPC

`GET /status`, `/genesis`, `/block`, `/block_by_hash`, `/blockchain`,
`/block_results`, `/commit`, `/validators`, `/tx`, `/tx_search`, `/abci_query`,
`/unconfirmed_txs`, `/net_info`.

`POST /broadcast_tx_sync` with `{ "tx": {…} }`.

Responses wrap `{ "jsonrpc": "2.0", "id": -1, "result": … }`.

---

## WebSocket `/ws`

On connect: `{ "type": "init", "height", "chain_id", "app_hash" }`.

Then: `new_block`, `new_tx`, `validator_set_update`, `epoch_boundary`, `ping`.

---

## Transaction shape

```json
{
  "body": { "messages": [{ "type": "povb/MsgDeclareParticipation", "validator": "volnix1…", "b_i": 400000, "s_i": 200000 }], "memo": "", "timeout_height": 0 },
  "auth_info": { "signer_infos": [{ "address": "volnix1…", "pub_hex": "…", "sequence": 0 }], "fee": { "amount": 0, "gas_limit": 200000, "denom": "uwrt" } },
  "signatures": ["<64 hex>"]
}
```

Message types: `bank/MsgSend` (WRT only), `ident/MsgVerifyIdentity`,
`ident/MsgMigrateRole`, `lizenz/MsgActivateLZN`, `lizenz/MsgDeactivateLZN`,
`anteil/MsgPlaceOrder`, `anteil/MsgCancelOrder`, `povb/MsgDeclareParticipation`,
`gov/MsgSubmitProposal`, `gov/MsgVote`.
