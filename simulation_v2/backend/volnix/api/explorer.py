"""Explorer REST API under /api/v1 — primary contract for the frontend."""

from __future__ import annotations

from typing import Any, Optional

from fastapi import APIRouter, HTTPException, Query, Request

from volnix.app.modules.anteil import ANT_WRT, LZN_WRT
from volnix.app.modules.mint import block_reward, era_at
from volnix.app.state import SCALE
from volnix.crypto.keys import derive_keypair
from volnix.types.msgs import (
    MsgDeclareParticipation,
    MsgPlaceOrder,
    MsgSend,
    MsgVerifyIdentity,
    msg_from_dict,
)
from volnix.types.tx import Tx

from volnix.api.schemas import (
    BroadcastTxRequest,
    ConsensusFaultRequest,
    OperatorAccountRequest,
    OperatorDeclareRequest,
    OperatorMintRequest,
    OperatorOrderRequest,
    OperatorRoleRequest,
    PaceResetRequest,
    ProduceRequest,
    TimeScaleRequest,
)

router = APIRouter(prefix="/api/v1", tags=["explorer"])


def _node(request: Request):
    return request.app.state.node


def _display(micro: int) -> str:
    q, r = divmod(int(micro), SCALE)
    return f"{q}.{r:06d}"


def _povb_summary(povb: dict[str, Any] | None) -> dict[str, Any]:
    """Compact PoVB fields for the block list tape."""
    if not povb:
        return {
            "set_updated": False,
            "n_declares": 0,
            "n_passed": 0,
            "sum_b": 0,
            "fill": 0,
            "l_decl": 0,
            "b_min": 0,
            "b_max": 0,
        }
    declares = povb.get("declares") or []
    passed = povb.get("passed") or []
    return {
        "set_updated": bool(povb.get("set_updated")),
        "n_declares": len(declares),
        "n_passed": len(passed),
        "sum_b": int(povb.get("sum_b") or 0),
        "fill": int(povb.get("fill") or 0),
        "l_decl": int(povb.get("l_decl") or 0),
        "b_min": int(povb.get("b_min") if povb.get("b_min") is not None else povb.get("lower") or 0),
        "b_max": int(povb.get("b_max") if povb.get("b_max") is not None else povb.get("upper") or 0),
    }


def _commit_for_height(node, height: int):
    """Commit that finalized height H (stored on block H+1, or tip `_last_commit`)."""
    nxt = node.blocks.get(height + 1)
    if nxt is not None and nxt.last_commit is not None:
        return nxt.last_commit
    latest = node.blocks.latest_height()
    if height == latest and node._last_commit is not None:
        # After produce(H), `_last_commit` holds the commit for H until H+1 is built.
        lc = node._last_commit
        if getattr(lc, "height", None) == height:
            return lc
    return None


def _consensus_bundle(node, height: int, results) -> dict[str, Any]:
    """Full consensus snapshot for block-detail analysis (§5.4 + CometBFT)."""
    signing_h = max(0, height - 1) if height > 0 else 0
    signing = node.valsets.get(signing_h)
    next_set = node.valsets.get(height)
    # Genesis height 0: signing set is the initial set at 0.
    if height == 0 and signing is None:
        signing = node.valsets.get(0)

    commit = _commit_for_height(node, height)
    power_by_addr: dict[str, int] = {}
    if signing is not None:
        power_by_addr = {v.address: v.power for v in signing.validators}

    voted_power = 0
    total_power = signing.total_power() if signing is not None else 0
    round_ = 0
    commit_dict = None
    if commit is not None:
        commit_dict = commit.to_dict()
        round_ = int(commit.round)
        for sig in commit.signatures:
            if sig.block_id_flag == "commit":
                voted_power += power_by_addr.get(sig.validator_address, 0)

    povb = dict(results.povb) if results and results.povb else {}
    return {
        "signing_set": signing.to_dict() if signing else None,
        "next_set": next_set.to_dict() if next_set else None,
        "commit": commit_dict,
        "povb": povb,
        "voted_power": voted_power,
        "total_power": total_power,
        "round": round_,
    }


@router.get("/chain/summary")
def chain_summary(request: Request) -> dict[str, Any]:
    node = _node(request)
    st = node.app.state
    p = st.params
    h = st.height
    era = era_at(h, p.halving_interval) if h else 0
    next_epoch = p.epoch_blocks - (h % p.epoch_blocks) if p.epoch_blocks and h else p.epoch_blocks
    if h and h % p.epoch_blocks == 0:
        next_epoch = p.epoch_blocks
    next_halving = p.halving_interval - (h % p.halving_interval) if p.halving_interval and h else p.halving_interval
    ant_supply = sum(a.ant for a in st.accounts.values()) + sum(
        o.escrow_base for o in st.orders.values() if o.status == "open" and o.market == ANT_WRT
    )
    lzn_free = sum(a.lzn for a in st.accounts.values())
    lzn_act = sum(a.lzn_activated for a in st.accounts.values())
    lzn_escrow = sum(o.escrow_base for o in st.orders.values() if o.status == "open" and o.market == LZN_WRT)
    return {
        "chain_id": st.chain_id,
        "height": h,
        "time": node.blocks.get(h).header.time if node.blocks.get(h) else st.genesis_time,
        "app_hash": st.app_hash(),
        "epoch": st.epoch,
        "blocks_to_epoch": next_epoch,
        "era": era,
        "blocks_to_halving": next_halving,
        "l_total": st.l_total(),
        "l_total_display": _display(st.l_total()),
        "supply": {
            "wrt": st.wrt_supply,
            "wrt_display": _display(st.wrt_supply),
            "lzn_minted_tokens": st.lzn_minted_tokens,
            "lzn_pool_remaining": st.lzn_pool_remaining,
            "lzn_free": lzn_free,
            "lzn_activated": lzn_act,
            "lzn_escrow": lzn_escrow,
            "ant": ant_supply,
            "ant_display": _display(ant_supply),
        },
        "params": {
            "lambda": f"{p.lambda_num}/{p.lambda_den}",
            "alpha": f"{p.alpha_num}/{p.alpha_den}",
            "k": p.max_active_validators,
            "epoch_blocks": p.epoch_blocks,
            "halving_interval": p.halving_interval,
            "base_block_reward": p.base_block_reward,
            "current_block_reward": block_reward(h, p.base_block_reward, p.halving_interval) if h else p.base_block_reward,
        },
        "n_accounts": len(st.accounts),
        "n_validators": len(node.app.validator_set.validators),
        "n_suppliers": len(st.active_suppliers()),
        "mempool": node.mempool.size(),
        **node.pace_snapshot(),
    }


@router.get("/blocks")
def list_blocks(request: Request, tail: int = 100, min_height: int = 0, max_height: int = 0) -> dict[str, Any]:
    node = _node(request)
    latest = node.blocks.latest_height()
    if max_height <= 0:
        max_height = latest
    if min_height <= 0 and tail > 0:
        min_height = max(0, max_height - tail + 1)
    items = []
    for b in reversed(node.blocks.range(min_height, max_height)):
        h = b.header.height
        res = node.results.get(h)
        povb = res.povb if res else None
        items.append(
            {
                "height": h,
                "hash": b.hash(),
                "time": b.header.time,
                "proposer": b.header.proposer_address,
                "n_txs": len(b.data.get("txs", [])),
                "app_hash": b.header.app_hash,
                "data_hash": b.header.data_hash,
                **_povb_summary(povb),
            }
        )
    return {"latest": latest, "blocks": items}


@router.get("/blocks/{height}")
def get_block(request: Request, height: int) -> dict[str, Any]:
    node = _node(request)
    b = node.blocks.get(height)
    if b is None:
        raise HTTPException(404, "block not found")
    res = node.results.get(height)
    return {
        "block": b.to_dict(),
        "hash": b.hash(),
        "results": res.to_dict() if res else None,
        "consensus": _consensus_bundle(node, height, res),
    }

@router.get("/txs")
def list_txs(request: Request, limit: int = 30) -> dict[str, Any]:
    node = _node(request)
    hashes = node.tx_index.all_hashes_newest_first()[:limit]
    items = []
    for h in hashes:
        loc = node.tx_index.get(h)
        if loc:
            items.append({"hash": h, "height": loc.height, "index": loc.index, "sender": loc.sender})
    return {"txs": items}


@router.get("/txs/{tx_hash}")
def get_tx(request: Request, tx_hash: str) -> dict[str, Any]:
    node = _node(request)
    loc = node.tx_index.get(tx_hash)
    if loc is None:
        raise HTTPException(404, "tx not found")
    block = node.blocks.get(loc.height)
    raw = block.data["txs"][loc.index] if block else None
    res = node.results.get(loc.height)
    tx_res = res.txs_results[loc.index] if res and loc.index < len(res.txs_results) else None
    return {
        "hash": tx_hash,
        "height": loc.height,
        "index": loc.index,
        "sender": loc.sender,
        "tx": raw,
        "result": tx_res.to_dict() if tx_res else None,
    }


def _account_row(node, acc) -> dict[str, Any]:
    """Compact wallet row for the accounts list."""
    in_set = any(v.address == acc.address for v in node.app.validator_set.validators)
    open_orders = sum(
        1 for o in node.app.state.orders.values() if o.owner == acc.address and o.status == "open"
    )
    return {
        "address": acc.address,
        "role": acc.role.value,
        "genesis_no_zkp": acc.genesis_no_zkp,
        "in_validator_set": in_set,
        "is_proposer": node.app.validator_set.proposer == acc.address,
        "wrt": acc.wrt,
        "lzn": acc.lzn,
        "lzn_activated": acc.lzn_activated,
        "ant": acc.ant,
        "wrt_display": _display(acc.wrt),
        "lzn_display": _display(acc.lzn),
        "lzn_activated_display": _display(acc.lzn_activated),
        "ant_display": _display(acc.ant),
        "sequence": acc.sequence,
        "last_tx_height": acc.last_tx_height,
        "created_height": acc.created_height,
        "lzn_freeze_until": acc.lzn_freeze_until,
        "open_orders": open_orders,
    }


@router.get("/accounts")
def list_accounts(
    request: Request,
    role: Optional[str] = Query(default=None, description="Filter: citizen|supplier|validator"),
) -> dict[str, Any]:
    """All simulation wallets: role/status and balances."""
    node = _node(request)
    st = node.app.state
    items = []
    for addr in sorted(st.accounts):
        acc = st.accounts[addr]
        if role and acc.role.value != role:
            continue
        items.append(_account_row(node, acc))
    return {
        "height": st.height,
        "count": len(items),
        "accounts": items,
    }


@router.get("/accounts/{address}")
def get_account(request: Request, address: str) -> dict[str, Any]:
    st = _node(request).app.state
    acc = st.accounts.get(address)
    if acc is None:
        raise HTTPException(404, "account not found")
    orders = [o.to_dict() for o in st.orders.values() if o.owner == address and o.status == "open"]
    rec = st.declares.get(address)
    return {
        "account": acc.to_dict(),
        "balances_display": {
            "wrt": _display(acc.wrt),
            "lzn": _display(acc.lzn),
            "lzn_activated": _display(acc.lzn_activated),
            "ant": _display(acc.ant),
        },
        "open_orders": orders,
        "last_declare": {
            "b_i": rec.b_i,
            "s_i": rec.s_i,
            "f_i": rec.f_i,
            "w_i": rec.w_i,
        }
        if rec
        else None,
    }


@router.get("/accounts/{address}/txs")
def account_txs(request: Request, address: str) -> dict[str, Any]:
    locs = _node(request).tx_index.by_account(address)
    return {
        "address": address,
        "txs": [{"hash": l.tx_hash, "height": l.height, "index": l.index} for l in reversed(locs)],
    }


@router.get("/validators")
def list_validators(request: Request) -> dict[str, Any]:
    node = _node(request)
    vs = node.app.validator_set
    items = []
    for v in vs.validators:
        acc = node.app.state.accounts.get(v.address)
        items.append(
            {
                **v.to_dict(),
                "role": acc.role if acc else "",
                "ant": acc.ant if acc else 0,
                "is_proposer": v.address == vs.proposer,
            }
        )
    return {"proposer": vs.proposer, "total_power": vs.total_power(), "validators": items}


@router.get("/validators/{address}")
def get_validator(request: Request, address: str) -> dict[str, Any]:
    node = _node(request)
    vs = node.app.validator_set.by_address()
    v = vs.get(address)
    acc = node.app.state.accounts.get(address)
    if acc is None:
        raise HTTPException(404, "not found")
    return {"validator": v.to_dict() if v else None, "account": acc.to_dict()}


@router.get("/povb/{height}")
def povb_at(request: Request, height: int) -> dict[str, Any]:
    node = _node(request)
    res = node.results.get(height)
    if res is None:
        raise HTTPException(404, "no results at height")
    return res.povb or {"height": height, "declares": []}


@router.get("/epochs")
def list_epochs(request: Request) -> dict[str, Any]:
    return {"epochs": [e.to_dict() for e in _node(request).app.state.epochs]}


@router.get("/epochs/{n}")
def get_epoch(request: Request, n: int) -> dict[str, Any]:
    for e in _node(request).app.state.epochs:
        if e.epoch == n:
            return e.to_dict()
    raise HTTPException(404, "epoch not found")


@router.get("/market/orderbook")
def orderbook(request: Request, market: str = ANT_WRT) -> dict[str, Any]:
    st = _node(request).app.state
    bids, asks = [], []
    for o in st.orders.values():
        if o.status != "open" or o.market != market or o.remaining <= 0:
            continue
        row = {"order_id": o.order_id, "owner": o.owner, "price": o.price, "remaining": o.remaining}
        if o.side == "BUY":
            bids.append(row)
        else:
            asks.append(row)
    bids.sort(key=lambda x: -x["price"])
    asks.sort(key=lambda x: x["price"])
    return {"market": market, "bids": bids, "asks": asks}


@router.get("/market/trades")
def trades(request: Request, tail: int = 50) -> dict[str, Any]:
    node = _node(request)
    found: list[dict[str, Any]] = []
    for h in reversed(node.blocks.heights()):
        res = node.results.get(h)
        if not res:
            continue
        for r in res.txs_results:
            for e in r.events:
                if e.type == "anteil.trade_executed":
                    attrs = {a.key: a.value for a in e.attributes}
                    found.append({"height": h, **attrs})
                    if len(found) >= tail:
                        return {"trades": found}
    return {"trades": found}


@router.get("/params")
def params(request: Request) -> dict[str, Any]:
    return _node(request).app.state.params.to_dict()


@router.get("/supply")
def supply(request: Request) -> dict[str, Any]:
    return chain_summary(request)["supply"]


@router.get("/search")
def search(request: Request, q: str = Query(..., min_length=1)) -> dict[str, Any]:
    node = _node(request)
    q = q.strip()
    if q.isdigit():
        b = node.blocks.get(int(q))
        if b:
            return {"kind": "block", "height": int(q), "hash": b.hash()}
    if node.tx_index.get(q):
        return {"kind": "tx", "hash": q}
    if node.blocks.get_by_hash(q):
        b = node.blocks.get_by_hash(q)
        return {"kind": "block", "height": b.header.height, "hash": q}
    if q in node.app.state.accounts:
        return {"kind": "account", "address": q}
    return {"kind": "unknown", "q": q}


@router.post("/tx")
def submit_tx(request: Request, body: BroadcastTxRequest) -> dict[str, Any]:
    tx = Tx.from_dict(body.tx)
    return _node(request).broadcast_tx(tx)


# --- STAND-ONLY operator helpers (not on-chain messages / not production RPC) ---


@router.post("/operator/account")
def op_account(request: Request, body: OperatorAccountRequest) -> dict[str, Any]:
    """Derive address from a seed. Does not write chain state."""
    kp = derive_keypair(body.seed)
    acc = _node(request).app.state.accounts.get(kp.address)
    return {
        "address": kp.address,
        "pub_hex": kp.pub_hex,
        "account": acc.to_dict() if acc is not None else None,
    }


@router.post("/operator/mint")
def op_mint(request: Request, body: OperatorMintRequest) -> dict[str, Any]:
    """Transfer WRT from genesis validator (earned via block subsidy §5.1; no premint)."""
    node = _node(request)
    if body.denom != "uwrt":
        raise HTTPException(400, "operator mint only supports uwrt")
    if body.amount <= 0:
        raise HTTPException(400, "amount must be positive")
    genesis_seed = "volnix-genesis-validator-v2"
    kp = derive_keypair(genesis_seed)
    genesis_acc = node.app.state.accounts.get(kp.address)
    if genesis_acc is None or genesis_acc.wrt < body.amount:
        raise HTTPException(
            400,
            "genesis WRT insufficient (canon §6.3: no premint; earn via block subsidy, then send)",
        )
    tx = node.compose_tx(
        kp,
        [MsgSend(from_address=kp.address, to_address=body.address, denom="uwrt", amount=body.amount)],
    )
    out = node.broadcast_tx(tx)
    return {"broadcast": out, "from": kp.address, "to": body.address, "amount": body.amount}


@router.post("/operator/tx")
def op_signed_tx(request: Request, seed: str, messages: list[dict[str, Any]]) -> dict[str, Any]:
    node = _node(request)
    kp = derive_keypair(seed)
    msgs = [msg_from_dict(m) for m in messages]
    tx = node.compose_tx(kp, msgs)
    return node.broadcast_tx(tx)


@router.post("/operator/declare")
def op_declare(request: Request, body: OperatorDeclareRequest) -> dict[str, Any]:
    node = _node(request)
    kp = derive_keypair(body.seed)
    tx = node.compose_tx(
        kp,
        [MsgDeclareParticipation(validator=kp.address, b_i=body.b_i, s_i=body.s_i)],
    )
    return node.broadcast_tx(tx)


@router.post("/operator/verify")
def op_verify(request: Request, body: OperatorRoleRequest) -> dict[str, Any]:
    node = _node(request)
    kp = derive_keypair(body.seed)
    tx = node.compose_tx(
        kp,
        [
            MsgVerifyIdentity(
                address=kp.address,
                zkp_proof=body.zkp_proof,
                verification_provider="sim",
                desired_role=body.desired_role,
            )
        ],
    )
    return node.broadcast_tx(tx)


@router.post("/operator/order")
def op_order(request: Request, body: OperatorOrderRequest) -> dict[str, Any]:
    node = _node(request)
    kp = derive_keypair(body.seed)
    tx = node.compose_tx(
        kp,
        [
            MsgPlaceOrder(
                owner=kp.address,
                market=body.market,
                side=body.side,
                order_type=body.order_type,
                amount=body.amount,
                price=body.price,
            )
        ],
    )
    return node.broadcast_tx(tx)


@router.post("/operator/produce")
async def op_produce(request: Request, body: ProduceRequest) -> dict[str, Any]:
    node = _node(request)
    heights = []
    for _ in range(max(1, body.count)):
        block = await node.produce_and_notify()
        if block:
            heights.append(block.header.height)
    return {"produced": heights}


@router.get("/operator/pace")
def op_pace_get(request: Request) -> dict[str, Any]:
    """Adaptive attempt window + stand time_scale (canon §6.2 / stand)."""
    return _node(request).pace_snapshot()


@router.post("/operator/reset-chain")
async def op_reset_chain(request: Request) -> dict[str, Any]:
    """STAND-ONLY: delete chain data and reload genesis. time_scale is kept."""
    node = _node(request)
    snap = node.reset_chain()
    await node._emit(
        {
            "type": "chain_reset",
            "height": snap["height"],
            "chain_id": snap["chain_id"],
            "app_hash": snap["app_hash"],
        }
    )
    return snap


@router.post("/operator/pace")
def op_pace_reset(request: Request, body: Optional[PaceResetRequest] = None) -> dict[str, Any]:
    """Force-reset attempt window to BaseBlockTime (tests / operator)."""
    if body is not None and not body.reset:
        return _node(request).pace_snapshot()
    return _node(request).reset_pace()


@router.get("/operator/time-scale")
def op_time_scale_get(request: Request) -> dict[str, Any]:
    """Stand-only wall-clock acceleration."""
    return _node(request).pace_snapshot()


@router.post("/operator/time-scale")
def op_time_scale_set(request: Request, body: TimeScaleRequest) -> dict[str, Any]:
    return _node(request).set_time_scale(body.time_scale)


@router.get("/operator/produce-interval")
def op_produce_interval_get(request: Request) -> dict[str, Any]:
    """Compat: interval_sec = wall_sleep (prefer GET /operator/pace or /time-scale)."""
    snap = _node(request).pace_snapshot()
    sec = float(snap["produce_interval_sec"])
    return {
        "interval_sec": sec,
        "interval_ms": round(sec * 1000),
        "auto_produce": snap["auto_produce"],
        **{
            k: snap[k]
            for k in (
                "attempt_window_sec",
                "pace_debt_blocks",
                "missed_budget_sec",
                "base_block_time",
                "time_scale",
                "wall_sleep_sec",
            )
        },
    }


@router.post("/operator/produce-interval")
def op_produce_interval_set(request: Request, body: Optional[PaceResetRequest] = None) -> dict[str, Any]:
    """Compat: POST resets canonical attempt window (use POST /time-scale to change speed)."""
    snap = op_pace_reset(request, body if body is not None else PaceResetRequest(reset=True))
    sec = float(snap["produce_interval_sec"])
    return {
        "interval_sec": sec,
        "interval_ms": round(sec * 1000),
        "auto_produce": snap["auto_produce"],
        **{
            k: snap[k]
            for k in (
                "attempt_window_sec",
                "pace_debt_blocks",
                "missed_budget_sec",
                "base_block_time",
                "time_scale",
                "wall_sleep_sec",
            )
        },
    }


@router.post("/operator/consensus")
def op_consensus(request: Request, body: ConsensusFaultRequest) -> dict[str, Any]:
    node = _node(request)
    node.faults.set_absent(body.absent)
    node.faults.set_nil(body.nil_vote)
    return node.faults.snapshot()


@router.get("/operator/consensus")
def op_consensus_get(request: Request) -> dict[str, Any]:
    return _node(request).faults.snapshot()
