"""CSV/JSONL экспортёры для блоков, ордеров, балансов и тиков."""
from __future__ import annotations

import csv
import io
import json
from typing import Iterable, List, Optional

from core.state import SIM_TREASURY_ADDR, StateManager


def blocks_jsonl(sm: StateManager, from_height: Optional[int] = None, to_height: Optional[int] = None) -> str:
    blocks: Iterable[dict]
    if from_height is None and to_height is None:
        blocks = sm.blocks
    else:
        fh = 0 if from_height is None else int(from_height)
        th = sm.current_height if to_height is None else int(to_height)
        # Сначала пытаемся персистентный ledger, иначе фильтруем in-memory blocks.
        from_ledger = sm.get_blocks_range(fh, th)
        if from_ledger:
            blocks = from_ledger
        else:
            blocks = [b for b in sm.blocks if fh <= int(b.get("height", -1)) <= th]
    buf = io.StringIO()
    for blk in blocks:
        buf.write(json.dumps(blk, ensure_ascii=False))
        buf.write("\n")
    return buf.getvalue()


def ticks_csv(sm: StateManager) -> str:
    """OHLC-тики price_history → CSV (time,price,ts)."""
    buf = io.StringIO()
    w = csv.writer(buf, lineterminator="\n")
    w.writerow(["time", "price", "ts"])
    for row in sm.price_history:
        w.writerow([row.get("time", ""), row.get("price", ""), row.get("ts", "")])
    return buf.getvalue()


def balances_csv(sm: StateManager) -> str:
    """Срез балансов: address,role,zkp,wrt,lzn,lzn_frozen_mining,ant."""
    buf = io.StringIO()
    w = csv.writer(buf, lineterminator="\n")
    w.writerow(["address", "role", "zkp_verified", "wrt", "lzn", "lzn_frozen_mining", "ant"])
    for addr in sorted(sm.accounts.keys()):
        acc = sm.accounts[addr]
        w.writerow(
            [
                addr,
                acc.role.value,
                int(acc.zkp_verified),
                acc.wrt_balance,
                acc.lzn_balance,
                acc.lzn_frozen_mining,
                acc.ant_balance,
            ]
        )
    return buf.getvalue()


def canon_log_jsonl(sm: StateManager, limit: int = 1000) -> str:
    buf = io.StringIO()
    for entry in sm.canon_log.to_list_newest_first()[:limit]:
        buf.write(json.dumps(entry, ensure_ascii=False))
        buf.write("\n")
    return buf.getvalue()
