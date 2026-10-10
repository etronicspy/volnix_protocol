"""WRT ops matrix: supplier LZN sell, joint LZN buy, activate gate, ANT hold."""

from __future__ import annotations

import sys
from fractions import Fraction
from pathlib import Path
from unittest.mock import AsyncMock, patch

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.agent import (
    MIN_ORDER,
    REVIEW_INTERVAL_BLOCKS,
    SCALE,
    EnrichmentAgent,
)
from volnix_traffic.profit import (
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    AccountView,
    MarketSnapshot,
    ProfitStrategy,
)
from volnix_traffic.registry import KIND_ENRICHMENT, BotWallet

ALPHA = Fraction(1, 50)
LAM = Fraction(1, 3)


def _snap(**kwargs) -> MarketSnapshot:
    base = dict(
        height=100,
        ant_price=50 * SCALE,
        lzn_price=100 * SCALE,
        l_total=SCALE,
        n_validators=1,
        n_suppliers=2,
        block_reward=50 * SCALE,
        lambda_f=1.0 / 3.0,
        epoch_blocks=10_080,
        ant_sold_hint=50 * SCALE,
    )
    base.update(kwargs)
    return MarketSnapshot(**base)


def _supplier() -> EnrichmentAgent:
    return EnrichmentAgent(
        BotWallet(
            seed="s0",
            address="volnix1sup",
            role=ROLE_SUPPLIER,
            kind=KIND_ENRICHMENT,
            verified=True,
        )
    )


def _validator() -> EnrichmentAgent:
    return EnrichmentAgent(
        BotWallet(
            seed="v0",
            address="volnix1val",
            role=ROLE_VALIDATOR,
            kind=KIND_ENRICHMENT,
            verified=True,
        )
    )


@pytest.mark.asyncio
async def test_supplier_sells_free_lzn():
    ag = _supplier()
    view = AccountView(
        address="volnix1sup",
        role=ROLE_SUPPLIER,
        wrt=SCALE,
        ant=0,
        lzn=10 * SCALE,
    )
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await ag._maybe_market(client, view, _snap(lzn_price=150 * SCALE), {})
    assert ok is True
    place.assert_awaited()
    kwargs = place.await_args.kwargs
    assert kwargs["market"] == "LZN/WRT"
    assert kwargs["side"] == "SELL"
    assert kwargs["amount"] >= MIN_ORDER
    assert kwargs["price"] >= 1


@pytest.mark.asyncio
async def test_supplier_markdowns_unsold_ask():
    """After REVIEW_INTERVAL without a fill, ask steps down (inventory pressure)."""
    from volnix_traffic.profit import private_ant_value

    ag = _supplier()
    ag.strategy.ask_shade = 1.0
    view = AccountView(address="volnix1sup", role=ROLE_SUPPLIER, wrt=SCALE, ant=100_000)
    snap0 = _snap(ant_price=0, lzn_price=0, last_ant_price=0, height=10)
    seed = private_ant_value(snap0)
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        await ag._maybe_market(client, view, snap0, {})
    assert place.await_args.kwargs["price"] == seed

    # Own open ask still sitting; advance past review window with no fill.
    book = {
        "asks": [
            {
                "order_id": "o1",
                "owner": "volnix1sup",
                "price": seed,
                "remaining": 10_000,
            }
        ],
        "bids": [],
    }
    snap1 = _snap(
        ant_price=0,
        lzn_price=0,
        last_ant_price=0,
        height=10 + REVIEW_INTERVAL_BLOCKS,
    )
    with (
        patch("volnix_traffic.agent.actions.cancel_order", new_callable=AsyncMock) as cancel,
        patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place2,
    ):
        cancel.return_value = True
        place2.return_value = True
        await ag._maybe_market(client, view, snap1, {}, ant_orderbook=book)
    assert place2.await_args.kwargs["price"] == max(1, int(seed * 0.96))
    assert place2.await_args.kwargs["price"] < seed


@pytest.mark.asyncio
async def test_supplier_keeps_ask_inside_review_window():
    """Within REVIEW_INTERVAL, unsold ask is not requoted."""
    from volnix_traffic.profit import private_ant_value

    ag = _supplier()
    ag.strategy.ask_shade = 1.0
    view = AccountView(address="volnix1sup", role=ROLE_SUPPLIER, wrt=SCALE, ant=100_000)
    snap0 = _snap(ant_price=0, lzn_price=0, last_ant_price=0, height=10)
    seed = private_ant_value(snap0)
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        await ag._maybe_market(client, view, snap0, {})
    book = {
        "asks": [
            {"order_id": "o1", "owner": "volnix1sup", "price": seed, "remaining": 10_000}
        ],
        "bids": [],
    }
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place2:
        place2.return_value = True
        await ag._maybe_market(
            client,
            view,
            _snap(ant_price=0, lzn_price=0, height=10 + REVIEW_INTERVAL_BLOCKS - 1),
            {},
            ant_orderbook=book,
        )
    place2.assert_not_awaited()


@pytest.mark.asyncio
async def test_validator_joint_lzn_buy_when_ant_zero():
    ag = _validator()
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=200 * SCALE,
        lzn=0,
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
            client,
            view,
            _snap(lzn_price=1 * SCALE, ant_price=1 * SCALE, l_total=SCALE, n_validators=1),
            {},
        )
    assert ok is True
    lzn_buys = [
        c
        for c in place.await_args_list
        if c.kwargs.get("market") == "LZN/WRT" and c.kwargs.get("side") == "BUY" and c.kwargs.get("amount", 0) > 0
    ]
    assert lzn_buys
    assert lzn_buys[0].kwargs["amount"] >= MIN_ORDER


@pytest.mark.asyncio
async def test_activate_free_lzn_always():
    """Free LZN activates without NPV — mining capacity is not gated on profit."""
    ag = _validator()
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=100,  # cannot buy corridor fuel
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
        await ag._maybe_market(
            client,
            view,
            _snap(ant_price=50 * SCALE, lzn_price=100 * SCALE, l_total=0),
            {},
        )
    act.assert_awaited_once()
    assert act.await_args.args[2] == SCALE


def test_best_first_role_uses_horizon_forecast():
    strat = ProfitStrategy()
    snap = _snap(ant_price=1 * SCALE, lzn_price=1 * SCALE)
    rich = AccountView(address="a", role="citizen", wrt=100 * SCALE, lzn=SCALE, ant=2 * SCALE)
    role = strat.best_first_role(rich, snap, alpha=ALPHA, lam=LAM, k=150)
    assert role in (ROLE_SUPPLIER, ROLE_VALIDATOR)
    cached = strat.get_cached_roi("a")
    assert cached is not None
    assert ROLE_VALIDATOR in cached and ROLE_SUPPLIER in cached


@pytest.mark.asyncio
async def test_supplier_empty_book_asks_private_opponent_value():
    """No book mid / last trade → ANT ask from private reservation × ask_shade."""
    from volnix_traffic.profit import private_ant_value

    ag = _supplier()
    ag.strategy.ask_shade = 1.0
    view = AccountView(address="volnix1sup", role=ROLE_SUPPLIER, wrt=SCALE, ant=100_000)
    snap = _snap(ant_price=0, lzn_price=0, last_ant_price=0)
    expected = private_ant_value(snap)
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await ag._maybe_market(client, view, snap, {})
    assert ok is True
    ant_sells = [
        c
        for c in place.await_args_list
        if c.kwargs.get("market") == "ANT/WRT" and c.kwargs.get("side") == "SELL"
    ]
    assert ant_sells
    assert ant_sells[0].kwargs["price"] == expected


@pytest.mark.asyncio
async def test_supplier_joins_best_bid():
    ag = _supplier()
    view = AccountView(address="volnix1sup", role=ROLE_SUPPLIER, wrt=SCALE, ant=100_000)
    book = {"bids": [{"price": 77, "remaining": 50_000, "owner": "other"}], "asks": []}
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        await ag._maybe_market(
            client, view, _snap(ant_price=0, lzn_price=0), {}, ant_orderbook=book
        )
    ant_sells = [
        c
        for c in place.await_args_list
        if c.kwargs.get("market") == "ANT/WRT" and c.kwargs.get("side") == "SELL"
    ]
    assert ant_sells
    assert ant_sells[0].kwargs["price"] == 77


@pytest.mark.asyncio
async def test_supplier_ask_uses_last_trade_when_no_bid():
    ag = _supplier()
    ag.strategy.ask_shade = 1.0
    view = AccountView(address="volnix1sup", role=ROLE_SUPPLIER, wrt=SCALE, ant=100_000)
    snap = _snap(ant_price=0, lzn_price=0, last_ant_price=33 * SCALE)
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        await ag._maybe_market(client, view, snap, {})
    ant_sells = [
        c
        for c in place.await_args_list
        if c.kwargs.get("market") == "ANT/WRT" and c.kwargs.get("side") == "SELL"
    ]
    assert ant_sells
    assert ant_sells[0].kwargs["price"] == 33 * SCALE
