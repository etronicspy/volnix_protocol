"""Role flip via new wallet; floor and genesis guards."""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import AsyncMock, patch

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.agent import FlipRequest
from volnix_traffic.profit import SCALE, MarketSnapshot
from volnix_traffic.registry import (
    KIND_ENRICHMENT,
    KIND_RETIRED,
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    BotRegistry,
    BotWallet,
)
from volnix_traffic.settings import TrafficSettings
from volnix_traffic.supervisor import EnrichmentSupervisor


def _snap() -> MarketSnapshot:
    return MarketSnapshot(ant_price=10, lzn_price=10, block_reward=50 * SCALE)


@pytest.mark.asyncio
async def test_flip_sell_send_verify():
    settings = TrafficSettings(autostart=False, min_suppliers=1, min_validators=1)
    reg = BotRegistry()
    genesis = BotWallet(
        seed=settings.genesis_seed,
        address="volnix1gen",
        role=ROLE_VALIDATOR,
        kind=KIND_ENRICHMENT,
        genesis=True,
        verified=True,
    )
    old = BotWallet(
        seed="old-s",
        address="volnix1old",
        role=ROLE_SUPPLIER,
        kind=KIND_ENRICHMENT,
        verified=True,
    )
    extra_s = BotWallet(
        seed="other-s",
        address="volnix1os",
        role=ROLE_SUPPLIER,
        kind=KIND_ENRICHMENT,
        verified=True,
    )
    reg.add(genesis)
    reg.add(old)
    reg.add(extra_s)
    sup = EnrichmentSupervisor(settings, reg)
    ag = sup._ensure_agent(old)
    req = FlipRequest(agent=ag, new_role=ROLE_VALIDATOR)
    by_addr = {
        old.address: {
            "address": old.address,
            "wrt": 20 * SCALE,
            "ant": SCALE,
            "lzn": 0,
            "open_orders": 0,
        }
    }

    async def fake_create(_c, bot):
        bot.address = "volnix1new"
        return bot

    with (
        patch("volnix_traffic.supervisor.actions.create_wallet", new=fake_create),
        patch("volnix_traffic.supervisor.actions.send_wrt", new_callable=AsyncMock) as send,
        patch("volnix_traffic.supervisor.actions.verify_identity", new_callable=AsyncMock) as verify,
        patch("volnix_traffic.supervisor.actions.place_order", new_callable=AsyncMock) as order,
    ):
        send.return_value = True
        verify.return_value = True
        order.return_value = True
        ok = await sup.execute_flip(AsyncMock(), req, _snap(), by_addr)

    assert ok is True
    assert old.kind == KIND_RETIRED
    assert old.seed not in sup.agents
    assert any(b.address == "volnix1new" and b.kind == KIND_ENRICHMENT for b in reg.all())
    send.assert_awaited()
    verify.assert_awaited()
    assert len(reg.enrichment()) == 3  # genesis + extra supplier + new validator
    assert all(b.kind != KIND_RETIRED or b.seed == old.seed for b in reg.all())


@pytest.mark.asyncio
async def test_flip_refuses_last_supplier():
    settings = TrafficSettings(autostart=False, min_suppliers=1, min_validators=1)
    reg = BotRegistry()
    genesis = BotWallet(
        seed=settings.genesis_seed,
        address="volnix1gen",
        role=ROLE_VALIDATOR,
        kind=KIND_ENRICHMENT,
        genesis=True,
        verified=True,
    )
    only_s = BotWallet(
        seed="only-s",
        address="volnix1s",
        role=ROLE_SUPPLIER,
        kind=KIND_ENRICHMENT,
        verified=True,
    )
    reg.add(genesis)
    reg.add(only_s)
    sup = EnrichmentSupervisor(settings, reg)
    ag = sup._ensure_agent(only_s)
    req = FlipRequest(agent=ag, new_role=ROLE_VALIDATOR)
    ok = await sup.execute_flip(AsyncMock(), req, _snap(), {})
    assert ok is False
    assert only_s.kind == KIND_ENRICHMENT
