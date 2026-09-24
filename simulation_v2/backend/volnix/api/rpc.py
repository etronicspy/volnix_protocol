"""CometBFT-compatible RPC surface."""

from __future__ import annotations

from typing import Any, Optional

from fastapi import APIRouter, HTTPException, Query, Request

from volnix.types.tx import Tx

router = APIRouter(tags=["rpc"])


def _node(request: Request):
    return request.app.state.node


@router.get("/status")
def status(request: Request) -> dict[str, Any]:
    return {"jsonrpc": "2.0", "id": -1, "result": _node(request).status()}


@router.get("/genesis")
def genesis(request: Request) -> dict[str, Any]:
    node = _node(request)
    block = node.blocks.get(0)
    return {
        "jsonrpc": "2.0",
        "id": -1,
        "result": {
            "genesis": {
                "genesis_time": node.app.state.genesis_time,
                "chain_id": node.app.state.chain_id,
                "initial_height": "1",
                "consensus_params": node.app.consensus_params,
                "app_hash": block.header.app_hash if block else "",
                "app_state": node.app.state.to_snapshot() if node.blocks.latest_height() == 0 else {},
            }
        },
    }


@router.get("/block")
def block(request: Request, height: Optional[int] = None) -> dict[str, Any]:
    node = _node(request)
    h = height if height is not None else node.blocks.latest_height()
    b = node.blocks.get(h)
    if b is None:
        raise HTTPException(404, f"block {h} not found")
    return {"jsonrpc": "2.0", "id": -1, "result": {"block_id": {"hash": b.hash()}, "block": b.to_dict()}}


@router.get("/block_by_hash")
def block_by_hash(request: Request, hash: str = Query(...)) -> dict[str, Any]:
    b = _node(request).blocks.get_by_hash(hash)
    if b is None:
        raise HTTPException(404, "block not found")
    return {"jsonrpc": "2.0", "id": -1, "result": {"block_id": {"hash": b.hash()}, "block": b.to_dict()}}


@router.get("/blockchain")
def blockchain(request: Request, minHeight: int = 0, maxHeight: int = 0) -> dict[str, Any]:
    node = _node(request)
    latest = node.blocks.latest_height()
    lo = minHeight
    hi = maxHeight if maxHeight > 0 else latest
    metas = []
    for b in node.blocks.range(lo, hi):
        metas.append({"block_id": {"hash": b.hash()}, "header": b.header.to_dict()})
    return {"jsonrpc": "2.0", "id": -1, "result": {"last_height": str(latest), "block_metas": metas}}


@router.get("/block_results")
def block_results(request: Request, height: Optional[int] = None) -> dict[str, Any]:
    node = _node(request)
    h = height if height is not None else node.blocks.latest_height()
    res = node.results.get(h)
    if res is None:
        raise HTTPException(404, f"results {h} not found")
    return {"jsonrpc": "2.0", "id": -1, "result": res.to_dict()}


@router.get("/commit")
def commit(request: Request, height: Optional[int] = None) -> dict[str, Any]:
    node = _node(request)
    h = height if height is not None else node.blocks.latest_height()
    b = node.blocks.get(h)
    if b is None:
        raise HTTPException(404, "block not found")
    nxt = node.blocks.get(h + 1)
    commit_d = nxt.last_commit.to_dict() if nxt and nxt.last_commit else None
    return {
        "jsonrpc": "2.0",
        "id": -1,
        "result": {
            "signed_header": {"header": b.header.to_dict(), "commit": commit_d},
            "canonical": True,
        },
    }


@router.get("/validators")
def validators(request: Request, height: Optional[int] = None) -> dict[str, Any]:
    node = _node(request)
    h = height if height is not None else node.blocks.latest_height()
    vs = node.valsets.get(h) or node.app.validator_set
    vals = [v.to_dict() for v in vs.validators]
    return {
        "jsonrpc": "2.0",
        "id": -1,
        "result": {"block_height": str(h), "validators": vals, "count": str(len(vals)), "total": str(len(vals))},
    }


@router.get("/tx")
def tx(request: Request, hash: str = Query(...)) -> dict[str, Any]:
    node = _node(request)
    loc = node.tx_index.get(hash)
    if loc is None:
        raise HTTPException(404, "tx not found")
    block = node.blocks.get(loc.height)
    if block is None:
        raise HTTPException(404, "block missing")
    raw = block.data["txs"][loc.index]
    res = node.results.get(loc.height)
    tx_res = res.txs_results[loc.index] if res and loc.index < len(res.txs_results) else None
    return {
        "jsonrpc": "2.0",
        "id": -1,
        "result": {
            "hash": hash,
            "height": str(loc.height),
            "index": loc.index,
            "tx": raw,
            "tx_result": tx_res.to_dict() if tx_res else None,
        },
    }


@router.get("/tx_search")
def tx_search(request: Request, query: str = "", page: int = 1, per_page: int = 30) -> dict[str, Any]:
    node = _node(request)
    hashes = node.tx_index.all_hashes_newest_first()
    if query:
        hashes = [h for h in hashes if query in h]
    start = (page - 1) * per_page
    chunk = hashes[start : start + per_page]
    txs = []
    for h in chunk:
        loc = node.tx_index.get(h)
        if loc:
            txs.append({"hash": h, "height": str(loc.height), "index": loc.index})
    return {"jsonrpc": "2.0", "id": -1, "result": {"txs": txs, "total_count": str(len(hashes))}}


@router.get("/abci_query")
def abci_query(request: Request, path: str = "", data: str = "") -> dict[str, Any]:
    node = _node(request)
    value: Any = None
    if path == "app_hash":
        value = node.app.state.app_hash()
    elif path.startswith("account/"):
        addr = path.split("/", 1)[1]
        acc = node.app.state.accounts.get(addr)
        value = acc.to_dict() if acc else None
    elif path == "params":
        value = node.app.state.params.to_dict()
    else:
        value = {"path": path, "data": data}
    return {
        "jsonrpc": "2.0",
        "id": -1,
        "result": {
            "response": {
                "code": 0,
                "log": "ok",
                "info": path,
                "value": value,
                "height": str(node.app.state.height),
            }
        },
    }


@router.post("/broadcast_tx_sync")
async def broadcast_tx_sync(request: Request) -> dict[str, Any]:
    node = _node(request)
    body = await request.json()
    raw = body.get("tx") if isinstance(body, dict) else None
    if raw is None:
        raise HTTPException(400, "tx required")
    tx_obj = Tx.from_dict(raw)
    out = node.broadcast_tx(tx_obj)
    return {"jsonrpc": "2.0", "id": -1, "result": out}


@router.get("/unconfirmed_txs")
def unconfirmed(request: Request) -> dict[str, Any]:
    pending = _node(request).mempool.pending()
    return {
        "jsonrpc": "2.0",
        "id": -1,
        "result": {"n_txs": str(len(pending)), "txs": [t.to_dict() for t in pending]},
    }


@router.get("/net_info")
def net_info() -> dict[str, Any]:
    return {
        "jsonrpc": "2.0",
        "id": -1,
        "result": {"listening": True, "n_peers": "0", "peers": []},
    }
