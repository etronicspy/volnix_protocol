"""Genesis validator is adopted as enrichment agent #1."""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import AsyncMock

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.agent import FlipRequest
from volnix_traffic.profit import MarketSnapshot
from volnix_traffic.registry import KIND_ENRICHMENT, BotRegistry, BotWallet
from volnix_traffic.settings import TrafficSettings
from volnix_traffic.supervisor import EnrichmentSupervisor
from volnix_traffic import actions


@pytest.mark.asyncio
async def test_adopt_genesis_registers_and_counts_in_50():
    settings = TrafficSettings(autostart=False, target_enrichment_bots=50)
    reg = BotRegistry()
    sup = EnrichmentSupervisor(settings, reg)

    async def fake_create(_client, bot):
        bot.address = "volnix1genesis"
        bot.pub_hex = "aa"
        return bot

    monkey = pytest.MonkeyPatch()
    monkey.setattr(actions, "create_wallet", fake_create)
    try:
        bot = await sup.adopt_genesis(AsyncMock())
    finally:
        monkey.undo()

    assert bot is not None
    assert bot.seed == settings.genesis_seed
    assert bot.genesis is True
    assert bot.kind == KIND_ENRICHMENT
    assert bot.role == "validator"
    assert len(reg.enrichment()) == 1
    assert settings.genesis_seed in sup.agents

    again = await sup.adopt_genesis(AsyncMock())
    assert again is bot
    assert len(reg.enrichment()) == 1


@pytest.mark.asyncio
async def test_genesis_flip_and_retire_forbidden():
    settings = TrafficSettings(autostart=False)
    reg = BotRegistry()
    genesis = BotWallet(
        seed=settings.genesis_seed,
        address="volnix1genesis",
        role="validator",
        kind=KIND_ENRICHMENT,
        genesis=True,
        verified=True,
    )
    reg.add(genesis)
    sup = EnrichmentSupervisor(settings, reg)
    ag = sup._ensure_agent(genesis)
    req = FlipRequest(agent=ag, new_role="supplier")
    snap = MarketSnapshot()
    ok = await sup.execute_flip(AsyncMock(), req, snap, {})
    assert ok is False
    assert genesis.kind == KIND_ENRICHMENT
    assert genesis.seed in sup.agents
