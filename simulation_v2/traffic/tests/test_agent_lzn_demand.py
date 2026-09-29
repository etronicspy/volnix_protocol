"""EnrichmentAgent validator market: max activate → ANT buffer → BUY LZN."""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import AsyncMock, patch

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.agent import (
    MIN_ORDER,
    SCALE,
    EnrichmentAgent,
    StrategyCard,
)
from volnix_traffic.profit import ROLE_VALIDATOR, AccountView, MarketSnapshot
from volnix_traffic.registry import KIND_ENRICHMENT, BotWallet


def _wallet() -> BotWallet:
    return BotWallet(
        seed="v0",
        address="volnix1val",
        role=ROLE_VALIDATOR,
        kind=KIND_ENRICHMENT,
        verified=True,
    )


def _snap(**kwargs) -> MarketSnapshot:
    base = dict(
        height=100,
        ant_price=50,
        lzn_price=100,
        l_total=SCALE,
        n_validators=1,
        block_reward=50 * SCALE,
        lambda_f=1.0 / 3.0,
    )
    base.update(kwargs)
    return MarketSnapshot(**base)


@pytest.mark.asyncio
async def test_activate_all_free_lzn():
    ag = EnrichmentAgent(_wallet())
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn=2 * SCALE,
        lzn_activated=SCALE,
        ant=SCALE,
    )
    client = AsyncMock()
    with (
        patch("volnix_traffic.agent.actions.activate_lzn", new_callable=AsyncMock) as act,
        patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place,
    ):
        act.return_value = True
        place.return_value = True
        ok = await ag._maybe_market(client, view, _snap(), {})
    assert ok is True
    act.assert_awaited_once()
    assert act.await_args.args[2] == 2 * SCALE


@pytest.mark.asyncio
async def test_buy_lzn_when_zero_inventory():
    ag = EnrichmentAgent(_wallet())
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn=0,
        lzn_activated=0,
        ant=SCALE,  # already buffered relative to tiny L_i
    )
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await ag._maybe_market(client, view, _snap(lzn_price=100, ant_price=50), {})
    assert ok is True
    place.assert_awaited_once()
    kwargs = place.await_args.kwargs
    assert kwargs["market"] == "LZN/WRT"
    assert kwargs["side"] == "BUY"
    assert kwargs["price"] == max(1, int(100 * ag.strategy.bid_shade))
    assert kwargs["amount"] >= MIN_ORDER


@pytest.mark.asyncio
async def test_ant_buffer_before_lzn_buy_when_ant_low():
    ag = EnrichmentAgent(_wallet())
    # Large activated mass → ant_target high; ant empty → ANT buy first.
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn=0,
        lzn_activated=SCALE,
        ant=0,
    )
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await ag._maybe_market(client, view, _snap(), {})
    assert ok is True
    first = place.await_args_list[0].kwargs
    assert first["market"] == "ANT/WRT"
    assert first["side"] == "BUY"


@pytest.mark.asyncio
async def test_multi_step_activate_then_ant_then_lzn():
    """One tick can run activate → ANT → LZN when each step is needed."""
    ag = EnrichmentAgent(_wallet())
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=50 * SCALE,
        lzn=SCALE,
        lzn_activated=0,
        ant=0,
    )
    client = AsyncMock()
    with (
        patch("volnix_traffic.agent.actions.activate_lzn", new_callable=AsyncMock) as act,
        patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place,
    ):
        act.return_value = True
        place.return_value = True
        ok = await ag._maybe_market(
            client, view, _snap(l_total=2 * SCALE, n_validators=1, ant_price=1, lzn_price=1), {}
        )
    assert ok is True
    act.assert_awaited_once()
    markets = [c.kwargs["market"] for c in place.await_args_list]
    assert "ANT/WRT" in markets
    assert "LZN/WRT" in markets


@pytest.mark.asyncio
async def test_lzn_lot_sized_by_l_total_fair_share():
    ag = EnrichmentAgent(_wallet())
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=100 * SCALE,
        lzn=0,
        lzn_activated=0,
        ant=SCALE,
    )
    client = AsyncMock()
    l_total = 10 * SCALE
    n_validators = 2
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        await ag._maybe_market(
            client,
            view,
            _snap(l_total=l_total, n_validators=n_validators, lzn_price=1, ant_price=1),
            {},
        )
    kwargs = place.await_args.kwargs
    assert kwargs["market"] == "LZN/WRT"
    assert kwargs["amount"] == l_total // n_validators


@pytest.mark.asyncio
async def test_no_deactivate_after_adopting_low_activate_ratio():
    ag = EnrichmentAgent(_wallet())
    peer = StrategyCard(
        address="volnix1peer",
        role=ROLE_VALIDATOR,
        b_frac=0.6,
        s_frac=0.3,
        activate_ratio=0.1,
        ask_shade=1.0,
        bid_shade=0.9,
        desired_role=ROLE_VALIDATOR,
        growth_wrt=100,
        expected_wrt=50,
    )
    ag.adopt(peer)
    assert ag.strategy.activate_ratio == 1.0
    assert ag.strategy.b_frac == 0.6

    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=SCALE,
        lzn=0,
        lzn_activated=2 * SCALE,
        ant=SCALE,
    )
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.deactivate_lzn", new_callable=AsyncMock) as deact:
        with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
            place.return_value = True
            await ag._maybe_market(client, view, _snap(), {"lzn_freeze_until": 1})
    deact.assert_not_awaited()


@pytest.mark.asyncio
async def test_validator_merges_open_bids_into_one():
    ag = EnrichmentAgent(_wallet())
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn=0,
        lzn_activated=0,
        ant=SCALE,
    )
    book = {
        "bids": [
            {"order_id": "ord-1", "owner": "volnix1val", "price": 90, "remaining": 20_000},
            {"order_id": "ord-2", "owner": "volnix1val", "price": 90, "remaining": 30_000},
        ],
        "asks": [],
    }
    client = AsyncMock()
    with (
        patch("volnix_traffic.agent.actions.cancel_order", new_callable=AsyncMock) as cancel,
        patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place,
    ):
        cancel.return_value = True
        place.return_value = True
        ok = await ag._maybe_market(
            client,
            view,
            _snap(lzn_price=100, ant_price=50, l_total=50_000, n_validators=1),
            {},
            lzn_orderbook=book,
        )
    assert ok is True
    assert cancel.await_count == 2
    place.assert_awaited_once()
    kwargs = place.await_args.kwargs
    assert kwargs["market"] == "LZN/WRT"
    assert kwargs["side"] == "BUY"
    assert kwargs["amount"] == 20_000 + 30_000


@pytest.mark.asyncio
async def test_supplier_replaces_asks_with_one():
    from volnix_traffic.profit import ROLE_SUPPLIER

    wallet = BotWallet(
        seed="s0",
        address="volnix1sup",
        role=ROLE_SUPPLIER,
        kind=KIND_ENRICHMENT,
        verified=True,
    )
    ag = EnrichmentAgent(wallet)
    view = AccountView(
        address="volnix1sup",
        role=ROLE_SUPPLIER,
        wrt=SCALE,
        ant=100_000,
    )
    book = {
        "asks": [
            {"order_id": "ord-a", "owner": "volnix1sup", "price": 50, "remaining": 40_000},
            {"order_id": "ord-b", "owner": "volnix1sup", "price": 55, "remaining": 15_000},
        ],
        "bids": [],
    }
    client = AsyncMock()
    with (
        patch("volnix_traffic.agent.actions.cancel_order", new_callable=AsyncMock) as cancel,
        patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place,
    ):
        cancel.return_value = True
        place.return_value = True
        ok = await ag._maybe_market(client, view, _snap(ant_price=50), {}, ant_orderbook=book)
    assert ok is True
    assert cancel.await_count == 2
    kwargs = place.await_args.kwargs
    assert kwargs["side"] == "SELL"
    assert kwargs["amount"] == 40_000 + 15_000 + min(100_000, max(MIN_ORDER, int(100_000 * 0.10)))


def test_forecast_uses_free_plus_activated_capacity():
    from fractions import Fraction

    from volnix_traffic.enrichment import forecast_validator

    # Only free LZN — should still get a declare capacity (activated+free).
    acc = AccountView(
        address="volnix1v",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn=SCALE,
        lzn_activated=0,
        ant=2 * SCALE,
    )
    snap = MarketSnapshot(
        height=10,
        block_reward=50 * SCALE,
        ant_price=1,
        l_total=SCALE,
        n_validators=1,
        epoch_blocks=10080,
        lambda_f=1 / 3,
    )
    fc = forecast_validator(
        acc,
        snap,
        rivals=[],
        horizon=12,
        alpha=Fraction(1, 50),
        lam=Fraction(1, 3),
        k=150,
        b_frac=0.5,
        s_frac=0.4,
    )
    assert fc.enter is True
    assert fc.net > 0
