"""WRT P&L: wealth_wrt, reservation pricing, NPV market gates, MTM growth."""

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
    SCALE,
    EnrichmentAgent,
    _ant_reservation,
)
from volnix_traffic.enrichment import forecast_validator, wealth_wrt
from volnix_traffic.profit import (
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    AccountView,
    MarketSnapshot,
)
from volnix_traffic.registry import KIND_ENRICHMENT, BotWallet

ALPHA = Fraction(1, 50)
LAM = Fraction(1, 3)


def _wallet(role: str = ROLE_VALIDATOR, address: str = "volnix1val") -> BotWallet:
    return BotWallet(
        seed="v0",
        address=address,
        role=role,
        kind=KIND_ENRICHMENT,
        verified=True,
    )


def _snap(**kwargs) -> MarketSnapshot:
    base = dict(
        height=100,
        ant_price=50 * SCALE,
        lzn_price=100 * SCALE,
        l_total=SCALE,
        n_validators=1,
        block_reward=50 * SCALE,
        lambda_f=1.0 / 3.0,
        epoch_blocks=10_080,
    )
    base.update(kwargs)
    return MarketSnapshot(**base)


def test_wealth_wrt_marks_inventory():
    acc = AccountView(
        address="a",
        role=ROLE_VALIDATOR,
        wrt=1_000,
        ant=100,
        lzn=2,
        lzn_activated=3,
    )
    snap = _snap(ant_price=10 * SCALE, lzn_price=20 * SCALE)
    # cost = qty * price // SCALE
    assert wealth_wrt(acc, snap) == 1_000 + (100 * 10 * SCALE) // SCALE + ((2 + 3) * 20 * SCALE) // SCALE


def test_forecast_validator_without_lzn_charges_lzn_price():
    rich = AccountView(
        address="volnix1v",
        role=ROLE_VALIDATOR,
        wrt=200 * SCALE,
        lzn=0,
        lzn_activated=0,
        ant=2 * SCALE,
    )
    snap = _snap(ant_price=SCALE, lzn_price=100 * SCALE)
    fc = forecast_validator(
        rich,
        snap,
        rivals=[],
        horizon=12,
        alpha=ALPHA,
        lam=LAM,
        k=150,
        b_frac=0.5,
        s_frac=0.4,
    )
    assert fc.cost_wrt == 100 * SCALE  # 1 LZN at 100 WRT
    assert fc.enter is True

    poor = AccountView(
        address="volnix1poor",
        role=ROLE_VALIDATOR,
        wrt=10,  # cannot afford 1 LZN
        lzn=0,
        lzn_activated=0,
        ant=2 * SCALE,
    )
    miss = forecast_validator(
        poor,
        snap,
        rivals=[],
        horizon=12,
        alpha=ALPHA,
        lam=LAM,
        k=150,
        b_frac=0.5,
        s_frac=0.4,
    )
    assert miss.enter is False
    assert miss.cost_wrt == 0
    assert miss.net == 0


def test_forecast_validator_does_not_clamp_negative_net():
    """Very expensive ANT can make entered declare net negative — must not clamp to 0."""
    acc = AccountView(
        address="volnix1v",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn_activated=SCALE,
        ant=2 * SCALE,
    )
    # ant_price huge vs block reward → burn costs dominate
    snap = _snap(ant_price=10_000 * SCALE, lzn_price=SCALE, block_reward=50 * SCALE)
    fc = forecast_validator(
        acc,
        snap,
        rivals=[],
        horizon=12,
        alpha=ALPHA,
        lam=LAM,
        k=150,
        b_frac=0.5,
        s_frac=0.4,
    )
    if fc.enter:
        assert fc.net < 0


def test_growth_wrt_mtm_rises_with_ant_inventory():
    ag = EnrichmentAgent(_wallet())
    snap = _snap(ant_price=100 * SCALE, lzn_price=SCALE)
    # First mark at cash-only wealth
    view0 = AccountView(address="volnix1val", role=ROLE_VALIDATOR, wrt=SCALE, ant=0)
    ag._update_growth(wealth_wrt(view0, snap), height=1)
    assert ag.growth_wrt == 0
    # Same cash, more ANT → MTM growth positive
    view1 = AccountView(address="volnix1val", role=ROLE_VALIDATOR, wrt=SCALE, ant=50_000)
    ag._update_growth(wealth_wrt(view1, snap), height=2)
    assert ag.growth_wrt == (50_000 * 100 * SCALE) // SCALE


@pytest.mark.asyncio
async def test_supplier_ask_uses_private_value_not_reservation_cap():
    """Free market: ask = private/last/mid × shade; no hard reservation ceiling."""
    wallet = BotWallet(
        seed="s0",
        address="volnix1sup",
        role=ROLE_SUPPLIER,
        kind=KIND_ENRICHMENT,
        verified=True,
    )
    ag = EnrichmentAgent(wallet)
    ag.strategy.ask_shade = 2.0
    view = AccountView(address="volnix1sup", role=ROLE_SUPPLIER, wrt=SCALE, ant=100_000)
    # Empty book mid; last trade anchors the quote.
    snap = _snap(
        ant_price=0,
        last_ant_price=100 * SCALE,
        block_reward=50 * SCALE,
        lambda_f=1 / 3,
        l_total=SCALE,
    )
    reservation = _ant_reservation(snap, SCALE)
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await ag._maybe_market(client, view, snap, {})
    assert ok is True
    px = place.await_args.kwargs["price"]
    assert px == max(1, int(100 * SCALE * 2.0))
    assert px > reservation  # no hard cap at reservation


@pytest.mark.asyncio
async def test_validator_crosses_ask_within_reservation_when_npv_positive():
    ag = EnrichmentAgent(_wallet())
    # Enough WRT to buy corridor fuel (~λL) at the ask.
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=100 * SCALE,
        lzn=0,
        lzn_activated=SCALE,
        ant=0,
    )
    snap = _snap(ant_price=100 * SCALE, block_reward=50 * SCALE)
    reservation = _ant_reservation(snap, SCALE)
    ask_px = max(1, reservation - 10 * SCALE)
    book = {
        "asks": [{"order_id": "a1", "owner": "volnix1sup", "price": ask_px, "remaining": SCALE}],
        "bids": [],
    }
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await ag._maybe_market(
            client,
            view,
            snap,
            {},
            ant_orderbook=book,
        )
    assert ok is True
    first = place.await_args_list[0].kwargs
    assert first["market"] == "ANT/WRT"
    assert first["side"] == "BUY"
    assert first["price"] == ask_px


@pytest.mark.asyncio
async def test_validator_fuel_crosses_ask_above_reservation():
    """Fuel-for-mine pays best ask even when above private reservation."""
    ag = EnrichmentAgent(_wallet())
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=200 * SCALE,
        lzn=0,
        lzn_activated=SCALE,
        ant=0,
    )
    snap = _snap(ant_price=100 * SCALE, block_reward=50 * SCALE)
    reservation = _ant_reservation(snap, SCALE)
    ask_px = reservation + 50 * SCALE
    book = {
        "asks": [{"order_id": "a1", "owner": "volnix1sup", "price": ask_px, "remaining": SCALE}],
        "bids": [],
    }
    client = AsyncMock()
    with patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await ag._maybe_market(client, view, snap, {}, ant_orderbook=book)
    assert ok is True
    ant_buys = [
        c
        for c in place.await_args_list
        if c.kwargs.get("market") == "ANT/WRT" and c.kwargs.get("amount", 0) > 0
    ]
    assert ant_buys
    assert ant_buys[0].kwargs["price"] == ask_px
    assert ask_px > reservation


@pytest.mark.asyncio
async def test_validator_buys_mine_fuel_even_when_npv_rejects():
    """Fuel-for-mine bypasses NPV so validators can keep declaring."""
    ag = EnrichmentAgent(_wallet())
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=100 * SCALE,
        lzn=0,
        lzn_activated=SCALE,
        ant=0,
    )
    snap = _snap(ant_price=50 * SCALE, lzn_price=1 * SCALE, block_reward=50 * SCALE)
    client = AsyncMock()
    with (
        patch.object(ag, "_declare_npv", return_value=False),
        patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place,
    ):
        place.return_value = True
        await ag._maybe_market(client, view, snap, {})
    ant_buys = [
        c
        for c in place.await_args_list
        if c.kwargs.get("market") == "ANT/WRT" and c.kwargs.get("amount", 0) > 0
    ]
    assert ant_buys
    assert ant_buys[0].kwargs["side"] == "BUY"


@pytest.mark.asyncio
async def test_validator_skips_buffer_ant_buy_when_not_profitable():
    """Already have mine fuel: speculative buffer needs expected_wrt >= 0."""
    from volnix_traffic.compete import entry_burn

    ag = EnrichmentAgent(_wallet())
    activated = SCALE
    f_i = entry_burn(activated, ALPHA)
    mine_fuel = f_i + max(1, int(float(LAM) * activated)) + 1
    view = AccountView(
        address="volnix1val",
        role=ROLE_VALIDATOR,
        wrt=100 * SCALE,
        lzn=0,
        lzn_activated=activated,
        ant=mine_fuel,  # corridor covered — only buffer remains
    )
    snap = _snap(ant_price=50 * SCALE, lzn_price=1 * SCALE, block_reward=50 * SCALE)
    client = AsyncMock()
    with (
        patch.object(ag, "_declare_npv", return_value=False),
        patch("volnix_traffic.agent.actions.place_order", new_callable=AsyncMock) as place,
    ):
        place.return_value = True
        await ag._maybe_market(client, view, snap, {})
    ant_buys = [
        c
        for c in place.await_args_list
        if c.kwargs.get("market") == "ANT/WRT" and c.kwargs.get("amount", 0) > 0
    ]
    assert ant_buys == []


@pytest.mark.asyncio
async def test_tick_declares_even_when_declare_npv_would_reject():
    """Chain liveness: declare whenever pick_declare succeeds — NPV not consulted."""
    from volnix_traffic.compete import DeclarePick

    ag = EnrichmentAgent(_wallet())
    ag.flip_confirm_ticks = 99
    snap = _snap()
    row = {
        "address": "volnix1val",
        "role": ROLE_VALIDATOR,
        "wrt": 10 * SCALE,
        "ant": SCALE,
        "lzn": 0,
        "lzn_activated": SCALE,
    }
    pick = DeclarePick(b_i=1, s_i=1, enter=True, expected_wrt=-10_000, f_i=20_000)
    client = AsyncMock()
    with (
        patch("volnix_traffic.agent.pick_declare", return_value=pick),
        patch.object(ag, "_declare_npv", return_value=False),
        patch("volnix_traffic.agent.actions.declare", new_callable=AsyncMock) as decl,
        patch.object(ag, "_maybe_market", new_callable=AsyncMock, return_value=False),
    ):
        decl.return_value = True
        out = await ag.tick(
            client,
            height=1,
            snap=snap,
            row=row,
            rivals=[],
            board=[],
            alpha=ALPHA,
            lam=LAM,
            k=150,
            n_suppliers=2,
            n_validators=2,
            min_suppliers=1,
            min_validators=1,
            enable_market=False,
            enable_declare=True,
            enable_role_flip=False,
        )
    assert out.declared is True
    decl.assert_awaited_once()


@pytest.mark.asyncio
async def test_tick_declares_before_flip_request():
    """Flip must not skip declare: both can appear on the same tick result."""
    from volnix_traffic.compete import DeclarePick

    wallet = BotWallet(
        seed="v0",
        address="volnix1val",
        role=ROLE_VALIDATOR,
        kind=KIND_ENRICHMENT,
        verified=True,
        genesis=False,
    )
    ag = EnrichmentAgent(wallet)
    ag.flip_confirm_ticks = 1
    ag.flip_streak = 1
    ag.flip_target = ROLE_SUPPLIER
    ag.role_flip_margin_wrt = -10**18  # any alt forecast wins
    snap = _snap()
    row = {
        "address": "volnix1val",
        "role": ROLE_VALIDATOR,
        "wrt": 10 * SCALE,
        "ant": SCALE,
        "lzn": 0,
        "lzn_activated": SCALE,
    }
    pick = DeclarePick(b_i=1, s_i=1, enter=True, expected_wrt=1, f_i=20_000)
    client = AsyncMock()
    with (
        patch("volnix_traffic.agent.pick_declare", return_value=pick),
        patch("volnix_traffic.agent.actions.declare", new_callable=AsyncMock) as decl,
        patch.object(ag, "_maybe_market", new_callable=AsyncMock, return_value=False),
        patch(
            "volnix_traffic.agent.forecast_roles",
            return_value={
                ROLE_VALIDATOR: type("F", (), {"net": 0})(),
                ROLE_SUPPLIER: type("F", (), {"net": 1_000_000})(),
            },
        ),
    ):
        decl.return_value = True
        out = await ag.tick(
            client,
            height=1,
            snap=snap,
            row=row,
            rivals=[],
            board=[],
            alpha=ALPHA,
            lam=LAM,
            k=150,
            n_suppliers=2,
            n_validators=2,
            min_suppliers=1,
            min_validators=1,
            enable_market=False,
            enable_declare=True,
            enable_role_flip=True,
        )
    assert out.declared is True
    assert out.flip is not None
    assert out.flip.new_role == ROLE_SUPPLIER
    decl.assert_awaited_once()


def test_declare_npv_allows_any_non_negative_expected_wrt():
    from volnix_traffic.compete import DeclarePick

    ag = EnrichmentAgent(_wallet())
    snap = _snap()
    with patch(
        "volnix_traffic.agent.pick_declare",
        return_value=DeclarePick(b_i=1, s_i=1, enter=False, expected_wrt=0, f_i=20_000),
    ):
        assert ag._declare_npv(
            address="volnix1val",
            l_i=SCALE,
            ant=SCALE,
            rivals=[],
            alpha=ALPHA,
            lam=LAM,
            k=150,
            snap=snap,
            ant_price=50 * SCALE,
            wealth=100 * SCALE,
        )


def test_declare_npv_rejects_negative_expected_wrt():
    from volnix_traffic.compete import DeclarePick

    ag = EnrichmentAgent(_wallet())
    snap = _snap()
    with patch(
        "volnix_traffic.agent.pick_declare",
        return_value=DeclarePick(b_i=1, s_i=1, enter=True, expected_wrt=-1, f_i=20_000),
    ):
        assert not ag._declare_npv(
            address="volnix1val",
            l_i=SCALE,
            ant=SCALE,
            rivals=[],
            alpha=ALPHA,
            lam=LAM,
            k=150,
            snap=snap,
            ant_price=50 * SCALE,
            wealth=100 * SCALE,
        )
