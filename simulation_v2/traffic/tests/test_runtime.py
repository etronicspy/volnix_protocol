"""Runtime pace sync and height catch-up."""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import AsyncMock

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.runtime import MAX_CATCHUP, TrafficRuntime, effective_poll_sec
from volnix_traffic.settings import TrafficSettings


def test_effective_poll_follows_produce_interval():
    assert effective_poll_sec(1.0, 0.25) == pytest.approx(0.25)
    assert effective_poll_sec(0.1, 0.25) == pytest.approx(0.025)
    assert effective_poll_sec(0.001, 0.25) == pytest.approx(0.001)
    assert effective_poll_sec(60.0, 0.25) == pytest.approx(0.25)
    assert effective_poll_sec(0.4, 0.25) == pytest.approx(0.1)


@pytest.mark.asyncio
async def test_catch_up_ticks_each_missed_height(tmp_path: Path):
    settings = TrafficSettings(
        autostart=False,
        node_url="http://127.0.0.1:9",
        poll_interval_sec=0.25,
        enable_market=False,
        enable_bots=False,
        enable_declare=False,
    )
    runtime = TrafficRuntime(settings, state_path=tmp_path / "wallets.json")
    runtime.last_height = 1
    runtime.client.chain_summary = AsyncMock(
        return_value={"height": 5, "produce_interval_sec": 0.05, "l_total": 0, "params": {}}
    )
    runtime.client.params = AsyncMock(return_value={})
    runtime.client.accounts = AsyncMock(return_value=[])
    runtime.client.orderbook = AsyncMock(return_value={})

    n = await runtime.catch_up_to(5)
    assert n == 4
    assert runtime.last_height == 5
    assert runtime.client.chain_summary.await_count == 4
    assert runtime.produce_interval_sec == 0.05
    assert runtime.effective_poll_sec == pytest.approx(0.0125)


@pytest.mark.asyncio
async def test_catch_up_respects_max_cap(tmp_path: Path):
    settings = TrafficSettings(
        autostart=False,
        node_url="http://127.0.0.1:9",
        enable_market=False,
        enable_bots=False,
        enable_declare=False,
    )
    runtime = TrafficRuntime(settings, state_path=tmp_path / "wallets.json")
    runtime.last_height = 1
    runtime.client.chain_summary = AsyncMock(
        return_value={"height": 100, "produce_interval_sec": 1.0, "l_total": 0, "params": {}}
    )
    runtime.client.params = AsyncMock(return_value={})
    runtime.client.accounts = AsyncMock(return_value=[])
    runtime.client.orderbook = AsyncMock(return_value={})

    n = await runtime.catch_up_to(100)
    assert n == MAX_CATCHUP
    assert runtime.last_height == 1 + MAX_CATCHUP


def test_status_includes_pace_fields(tmp_path: Path):
    settings = TrafficSettings(autostart=False, node_url="http://127.0.0.1:9")
    runtime = TrafficRuntime(settings, state_path=tmp_path / "wallets.json")
    runtime.produce_interval_sec = 0.5
    runtime.effective_poll_sec = 0.125
    st = runtime.status()
    assert st["produce_interval_sec"] == 0.5
    assert st["effective_poll_sec"] == 0.125
