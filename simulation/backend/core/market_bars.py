"""Агрегация price_history → OHLC для графиков (формат Apache ECharts candlestick)."""

from __future__ import annotations

import math
import time
from typing import Any, Dict, List, Tuple

MAX_TRADE_BARS = 8000
MAX_BUCKET_BARS = 4000


def _normalize_ticks(rows: List[dict]) -> List[Tuple[float, float]]:
    now = time.time()
    out: List[Tuple[float, float]] = []
    n = len(rows)
    for i, t in enumerate(rows):
        raw_ts = t.get("ts")
        try:
            price = float(t.get("price", 0))
        except (TypeError, ValueError):
            price = 0.0
        if raw_ts is None or not isinstance(raw_ts, (int, float)):
            ts = now - (n - 1 - i)
        else:
            ts = float(raw_ts)
        out.append((ts, price))
    out.sort(key=lambda x: x[0])
    return out


def ticks_to_ohlc_bars(rows: List[dict], interval_sec: int) -> List[Dict[str, float]]:
    """interval_sec <= 0: одна «свеча» на сделку; иначе корзина по секундам."""
    if not rows:
        return []
    norm = _normalize_ticks(rows)

    if interval_sec <= 0:
        if len(norm) > MAX_TRADE_BARS:
            norm = norm[-MAX_TRADE_BARS:]
        last_t = float("-inf")
        bars: List[Dict[str, float]] = []
        for ts, price in norm:
            t = ts
            if t <= last_t:
                t = last_t + 1e-6
            last_t = t
            bars.append(
                {"t": t, "open": price, "high": price, "low": price, "close": price}
            )
        return bars

    interval = float(interval_sec)
    buckets: Dict[float, Dict[str, float]] = {}
    for ts, price in norm:
        b = math.floor(ts / interval) * interval
        ex = buckets.get(b)
        if ex is None:
            buckets[b] = {
                "t": b,
                "open": price,
                "high": price,
                "low": price,
                "close": price,
            }
        else:
            ex["high"] = max(ex["high"], price)
            ex["low"] = min(ex["low"], price)
            ex["close"] = price

    ordered = sorted(buckets.keys())
    if len(ordered) > MAX_BUCKET_BARS:
        ordered = ordered[-MAX_BUCKET_BARS:]
    return [buckets[k] for k in ordered]


def bars_to_echarts_payload(bars: List[Dict[str, float]], trade_mode: bool) -> Dict[str, Any]:
    """
    ECharts candlestick: каждая точка [open, close, lowest, highest].
    Ось X — уникальные подписи (индекс + время), чтобы не схлопывать бары.
    """
    category: List[str] = []
    times: List[float] = []
    values: List[List[float]] = []
    for i, bar in enumerate(bars):
        t = float(bar["t"])
        times.append(t)
        if trade_mode:
            category.append(
                f"{i + 1} · {time.strftime('%H:%M:%S', time.localtime(t))}"
            )
        else:
            category.append(time.strftime("%m-%d %H:%M", time.localtime(t)))
        o, cl, lo, hi = bar["open"], bar["close"], bar["low"], bar["high"]
        values.append([o, cl, lo, hi])
    return {"category": category, "times": times, "values": values}
