"""KPI и аналитика поверх StateManager (Gini, velocity, burn ratio и др.)."""
from __future__ import annotations

import math
from typing import Dict, List, Tuple

from core.models import Role
from core.state import (
    BLOCKS_PER_EPOCH,
    GENESIS_PROVIDER_ADDR,
    GENESIS_VALIDATOR_ADDR,
    SIM_TREASURY_ADDR,
    StateManager,
)


def gini_coefficient(values: List[float]) -> float:
    """Стандартный Gini ∈ [0, 1]. Возвращает 0 для пустого / нулевого набора."""
    arr = [float(v) for v in values if v > 0]
    if not arr:
        return 0.0
    arr.sort()
    n = len(arr)
    total = sum(arr)
    if total <= 0:
        return 0.0
    cum = 0.0
    for i, x in enumerate(arr, start=1):
        cum += i * x
    return (2.0 * cum) / (n * total) - (n + 1) / n


def balances_by_asset(sm: StateManager, asset: str) -> Dict[str, float]:
    """Балансы по адресам (без казны симуляции — она не «активный» участник)."""
    out: Dict[str, float] = {}
    for addr, acc in sm.accounts.items():
        if addr == SIM_TREASURY_ADDR:
            continue
        a = asset.lower()
        if a == "wrt":
            out[addr] = float(acc.wrt_balance)
        elif a == "lzn":
            out[addr] = float(acc.lzn_balance + acc.lzn_frozen_mining)
        elif a == "ant":
            out[addr] = float(acc.ant_balance)
        elif a == "frozen":
            out[addr] = float(acc.lzn_frozen_mining)
    return out


def total_supply(sm: StateManager, asset: str) -> float:
    return sum(balances_by_asset(sm, asset).values())


def velocity(sm: StateManager, asset: str = "wrt", window: int = 100) -> float:
    """Сумма потоков (transfer + match) за последние `window` блоков / supply.

    Простейшая аппроксимация: считаем суммарный объём tx указанного актива в
    последних `window` блоках в RAM, делим на supply того же актива.
    """
    if window <= 0:
        return 0.0
    blocks = sm.blocks[-window:] if window <= len(sm.blocks) else list(sm.blocks)
    flow = 0.0
    a = asset.lower()
    for blk in blocks:
        for tx in blk.get("transactions") or []:
            asset_t = (tx.get("asset_type") or "wrt").lower()
            if a == "wrt" and tx.get("tx_type") in {"transfer", "match"}:
                if asset_t == "wrt":
                    flow += float(tx.get("amount") or 0)
            elif a == "ant" and tx.get("tx_type") == "match":
                flow += float(tx.get("amount") or 0)
            elif a == "lzn" and tx.get("tx_type") == "transfer" and asset_t == "lzn":
                flow += float(tx.get("amount") or 0)
    supply = total_supply(sm, asset)
    if supply <= 0:
        return 0.0
    return flow / supply


def burn_ratio(sm: StateManager) -> Tuple[float, float, float]:
    """(burned_this_epoch, emission_estimate, ratio).

    Здесь emission_estimate = epoch_ant_sold_volume × epoch_emission_coefficient
    (то, что будет начислено провайдерам на ближайшей границе эпохи).
    """
    burned = float(sm.current_epoch_burn)
    sold = float(sm.epoch_ant_sold_volume)
    coeff = float(sm.epoch_emission_coefficient)
    emission_estimate = sold * coeff
    ratio = burned / emission_estimate if emission_estimate > 1e-12 else 0.0
    return burned, emission_estimate, ratio


def role_counts(sm: StateManager) -> Dict[str, int]:
    out: Dict[str, int] = {"citizen": 0, "provider": 0, "validator": 0, "treasury": 0}
    for addr, acc in sm.accounts.items():
        if addr == SIM_TREASURY_ADDR:
            out["treasury"] += 1
            continue
        if acc.role == Role.PROVIDER:
            out["provider"] += 1
        elif acc.role == Role.VALIDATOR:
            out["validator"] += 1
        else:
            out["citizen"] += 1
    return out


def accepted_block_ratio(sm: StateManager) -> float:
    """Доля принятых блоков: блоков в ленте / попыток (попыток ≈ blocks + reject-записи в canon_log)."""
    in_chain = max(0, sm.current_height)
    rejects = 0
    for entry in sm.canon_log.to_list_newest_first():
        if entry.get("status") == "reject" and entry.get("canon") in {"§5.4", "§6.1"}:
            rejects += 1
    attempts = in_chain + rejects
    if attempts <= 0:
        return 1.0
    return in_chain / attempts


def fee_distribution_per_validator(sm: StateManager, last_n: int = 100) -> Dict[str, float]:
    """Сумма block_reward по proposer-у в последних N блоках."""
    n = min(last_n, len(sm.blocks))
    out: Dict[str, float] = {}
    for blk in sm.blocks[-n:]:
        for tx in blk.get("transactions") or []:
            if tx.get("tx_type") == "block_reward":
                addr = tx.get("receiver") or ""
                if addr:
                    out[addr] = out.get(addr, 0.0) + float(tx.get("amount") or 0)
    return out


def avg_match_spread(sm: StateManager, last_n: int = 100) -> float:
    """Средний |price - mid_price| по match-tx последних N блоков. 0 при отсутствии данных."""
    n = min(last_n, len(sm.blocks))
    spreads: List[float] = []
    for blk in sm.blocks[-n:]:
        market = sm.get_orderbook()
        bids = market.get("bids") or []
        asks = market.get("asks") or []
        if not bids or not asks:
            continue
        mid = (float(bids[0]["price"]) + float(asks[0]["price"])) / 2.0
        for tx in blk.get("transactions") or []:
            if tx.get("tx_type") == "match":
                p = float(tx.get("price") or 0)
                spreads.append(abs(p - mid))
    if not spreads:
        return 0.0
    return sum(spreads) / len(spreads)


def kpi_snapshot(sm: StateManager) -> dict:
    """Скоринг для UI/Prometheus — компактный набор показателей."""
    burned, emission_est, ratio = burn_ratio(sm)
    bal_wrt = balances_by_asset(sm, "wrt")
    bal_ant = balances_by_asset(sm, "ant")
    bal_lzn = balances_by_asset(sm, "lzn")
    return {
        "height": sm.current_height,
        "blocks_per_epoch": BLOCKS_PER_EPOCH,
        "epoch_remaining_blocks": max(0, BLOCKS_PER_EPOCH - (sm.current_height % BLOCKS_PER_EPOCH)),
        "supply": {
            "wrt": sum(bal_wrt.values()),
            "lzn": sum(bal_lzn.values()),
            "ant": sum(bal_ant.values()),
        },
        "gini": {
            "wrt": round(gini_coefficient(list(bal_wrt.values())), 6),
            "ant": round(gini_coefficient(list(bal_ant.values())), 6),
            "lzn": round(gini_coefficient(list(bal_lzn.values())), 6),
        },
        "velocity": {
            "wrt": round(velocity(sm, "wrt"), 6),
            "ant": round(velocity(sm, "ant"), 6),
            "lzn": round(velocity(sm, "lzn"), 6),
        },
        "burn": {
            "current_epoch": round(burned, 6),
            "emission_estimate": round(emission_est, 6),
            "ratio": round(ratio, 6),
        },
        "accepted_block_ratio": round(accepted_block_ratio(sm), 6),
        "avg_match_spread": round(avg_match_spread(sm), 6),
        "fee_distribution_top": _top_n(fee_distribution_per_validator(sm), 5),
        "role_counts": role_counts(sm),
        "consensus_validator_count": len(sm.consensus_validator_set or []),
        "mempool_size": len(sm.mempool),
        "last_price": float(sm.last_price),
    }


def _top_n(d: Dict[str, float], n: int) -> List[dict]:
    items = sorted(d.items(), key=lambda x: -x[1])[:n]
    return [{"address": addr, "value": round(val, 6)} for addr, val in items]
