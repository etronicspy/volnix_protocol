"""Citizen pool: no verify, bounded spontaneous MsgSend."""

from __future__ import annotations

import random
import sys
from pathlib import Path
from unittest.mock import AsyncMock, patch

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.citizens import MIN_TRANSFER, CitizenTraffic
from volnix_traffic.registry import KIND_CITIZEN, BotRegistry, BotWallet
from volnix_traffic.settings import TrafficSettings


@pytest.mark.asyncio
async def test_spend_includes_enrichment_wallets():
    settings = TrafficSettings(autostart=False)
    reg = BotRegistry()
    bots = [
        BotWallet(seed="e0", address="volnix1e0", kind="enrichment", role="validator", genesis=True),
        BotWallet(seed="e1", address="volnix1e1", kind="enrichment", role="supplier"),
        BotWallet(seed="c0", address="volnix1c0", kind=KIND_CITIZEN),
        BotWallet(seed="c1", address="volnix1c1", kind=KIND_CITIZEN),
    ]
    for w in bots:
        reg.add(w)
    by_addr = {
        w.address: {"address": w.address, "wrt": MIN_TRANSFER * 20, "role": w.role} for w in bots
    }
    traffic = CitizenTraffic(settings, reg)
    with patch("volnix_traffic.citizens.actions.send_wrt", new_callable=AsyncMock) as send:
        send.return_value = True
        n = await traffic.step(AsyncMock(), by_addr, intensity=10.0, rng=random.Random(0))
    assert n == 4
    srcs = {call.args[1].address for call in send.await_args_list}
    assert srcs == {w.address for w in bots}


@pytest.mark.asyncio
async def test_spend_stops_when_intensity_zero():
    settings = TrafficSettings(autostart=False)
    reg = BotRegistry()
    for i in range(3):
        w = BotWallet(seed=f"w{i}", address=f"volnix1w{i}", kind=KIND_CITIZEN)
        reg.add(w)
    by_addr = {
        f"volnix1w{i}": {"address": f"volnix1w{i}", "wrt": MIN_TRANSFER * 20} for i in range(3)
    }
    traffic = CitizenTraffic(settings, reg)
    with patch("volnix_traffic.citizens.actions.send_wrt", new_callable=AsyncMock) as send:
        n = await traffic.step(AsyncMock(), by_addr, intensity=0.0, rng=random.Random(0))
    assert n == 0
    send.assert_not_awaited()


@pytest.mark.asyncio
async def test_citizen_step_sends_n_transfers():
    settings = TrafficSettings(autostart=False, citizen_transfers_per_tick=3)
    reg = BotRegistry()
    for i in range(4):
        w = BotWallet(seed=f"c{i}", address=f"volnix1c{i}", kind=KIND_CITIZEN)
        reg.add(w)
    by_addr = {
        f"volnix1c{i}": {"address": f"volnix1c{i}", "wrt": MIN_TRANSFER * 20, "role": "citizen"}
        for i in range(4)
    }
    traffic = CitizenTraffic(settings, reg)
    with patch("volnix_traffic.citizens.actions.send_wrt", new_callable=AsyncMock) as send:
        send.return_value = True
        n = await traffic.step(AsyncMock(), by_addr, n_transfers=3, rng=random.Random(0))
    assert n == 3
    assert send.await_count == 3


@pytest.mark.asyncio
async def test_citizen_pool_never_verifies():
    settings = TrafficSettings(autostart=False, target_citizens=2, bootstrap_wrt=1_000_000)
    reg = BotRegistry()
    traffic = CitizenTraffic(settings, reg)
    idx = {"n": 0}

    async def fake_create(_c, bot):
        bot.address = f"volnix1new{idx['n']}"
        idx["n"] += 1
        return bot

    with (
        patch("volnix_traffic.citizens.actions.create_wallet", new=fake_create),
        patch("volnix_traffic.citizens.actions.fund_wrt", new_callable=AsyncMock) as fund,
        patch("volnix_traffic.citizens.actions.verify_identity", new_callable=AsyncMock) as verify,
    ):
        fund.return_value = True
        created = await traffic.ensure_pool(AsyncMock(), max_new=3)
    assert created == 2
    assert all(b.kind == KIND_CITIZEN for b in reg.all())
    assert all(b.role == "citizen" for b in reg.all())
    verify.assert_not_awaited()
