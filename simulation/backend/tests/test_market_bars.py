"""OHLC агрегация для ECharts (core/market_bars.py)."""
from __future__ import annotations

from core.market_bars import bars_to_echarts_payload, ticks_to_ohlc_bars


def test_empty_ticks_return_empty():
    assert ticks_to_ohlc_bars([], 0) == []
    assert ticks_to_ohlc_bars([], 60) == []


def test_trade_mode_one_bar_per_tick():
    """interval_sec ≤ 0 → одна свеча на сделку."""
    ticks = [
        {"ts": 1.0, "price": 10.0, "time": "00:00:01"},
        {"ts": 2.0, "price": 11.0, "time": "00:00:02"},
        {"ts": 3.0, "price": 9.5, "time": "00:00:03"},
    ]
    bars = ticks_to_ohlc_bars(ticks, 0)
    assert len(bars) == 3
    # каждая свеча open == close == high == low (одна точка)
    for b, t in zip(bars, ticks):
        assert b["open"] == b["close"] == b["high"] == b["low"] == t["price"]


def test_bucket_mode_aggregates_to_ohlc():
    """interval_sec > 0 → группируем в корзины."""
    ticks = [
        {"ts": 100.0, "price": 10.0},
        {"ts": 130.0, "price": 12.0},
        {"ts": 159.0, "price": 8.0},  # все в корзине [60, 120) или (60, 120]
        {"ts": 200.0, "price": 15.0},
    ]
    bars = ticks_to_ohlc_bars(ticks, 60)
    # 60-секундные корзины: floor(100/60)*60=60, 120, 180
    assert len(bars) == 3
    # Первая корзина (60..120): только tick at 100 → OHLC=(10,10,10,10)
    assert bars[0]["t"] == 60.0
    assert bars[0]["open"] == 10.0
    # Вторая корзина (120..180): 130 и 159 → open=12, close=8, high=12, low=8
    assert bars[1]["t"] == 120.0
    assert bars[1]["open"] == 12.0
    assert bars[1]["close"] == 8.0
    assert bars[1]["high"] == 12.0
    assert bars[1]["low"] == 8.0


def test_payload_format():
    bars = [
        {"t": 60.0, "open": 1.0, "close": 2.0, "low": 0.5, "high": 2.5},
        {"t": 120.0, "open": 2.0, "close": 1.5, "low": 1.5, "high": 2.0},
    ]
    payload = bars_to_echarts_payload(bars, trade_mode=False)
    assert "category" in payload and len(payload["category"]) == 2
    assert "times" in payload and payload["times"] == [60.0, 120.0]
    assert payload["values"] == [
        [1.0, 2.0, 0.5, 2.5],
        [2.0, 1.5, 1.5, 2.0],
    ]
