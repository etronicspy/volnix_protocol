"""Unit tests for LZN demand paths in traffic (no live node)."""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import AsyncMock, patch

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.engine import BotEngine, MIN_ORDER, SCALE
from volnix_traffic.profit import (
    ROLE_VALIDATOR,
    MarketSnapshot,
    reservation_bootstrap,
    snapshot_from_chain,
)
from volnix_traffic.registry import BotRegistry, BotWallet
from volnix_traffic.settings import TrafficSettings


def _summary(**kwargs):
    base = {
        "height": 10,
        "l_total": SCALE,
        "n_validators": 1,
        "n_suppliers": 0,
        "params": {
            "epoch_blocks": 10080,
            "current_block_reward": 50 * SCALE,
            "base_block_reward": 50 * SCALE,
        },
        "supply": {"ant": 100 * SCALE},
    }
    base.update(kwargs)
    return base


def test_lzn_price_bootstrap_when_book_empty():
    summary = _summary()
    ant_book = {
        "bids": [{"price": 100, "remaining": 1000}],
        "asks": [{"price": 200, "remaining": 2000}],
    }
    snap = snapshot_from_chain(summary, ant_book, lzn_orderbook={"bids": [], "asks": []})
    expected = reservation_bootstrap(50 * SCALE, 1 / 3, SCALE)
    assert snap.ant_price == 150
    assert snap.lzn_price == expected


def test_lzn_price_from_lzn_book_mid():
    summary = _summary()
    ant_book = {"bids": [], "asks": []}
    lzn_book = {
        "bids": [{"price": 400, "remaining": 500}],
        "asks": [{"price": 600, "remaining": 500}],
    }
    snap = snapshot_from_chain(summary, ant_book, lzn_orderbook=lzn_book)
    assert snap.lzn_price == 500


@pytest.mark.asyncio
async def test_trade_lzn_buys_without_sellers():
    settings = TrafficSettings(autostart=False, target_wallets=1)
    engine = BotEngine(settings, BotRegistry())
    bot = BotWallet(seed="bot-test", address="volnix1buyer", role=ROLE_VALIDATOR, verified=True)
    engine.registry.add(bot)
    by_addr = {
        bot.address: {
            "address": bot.address,
            "role": ROLE_VALIDATOR,
            "wrt": 10 * SCALE,
            "lzn": 0,
            "lzn_activated": 0,
            "ant": 0,
        }
    }
    snap = MarketSnapshot(lzn_price=100, ant_price=50, l_total=SCALE, block_reward=50 * SCALE)
    client = AsyncMock()

    with patch("volnix_traffic.engine.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await engine._trade_lzn(client, [bot], by_addr, snap, {"bids": [], "asks": []})
    assert ok is True
    place.assert_awaited_once()
    kwargs = place.await_args.kwargs
    assert kwargs["market"] == "LZN/WRT"
    assert kwargs["side"] == "BUY"
    assert kwargs["price"] == max(1, int(100 * 0.95))
    assert kwargs["amount"] >= MIN_ORDER


@pytest.mark.asyncio
async def test_prep_bids_when_validator_has_zero_lzn():
    settings = TrafficSettings(autostart=False, target_wallets=1, min_bot_wrt=1)
    engine = BotEngine(settings, BotRegistry())
    bot = BotWallet(seed="bot-prep", address="volnix1val", role=ROLE_VALIDATOR, verified=True)
    engine.registry.add(bot)
    by_addr = {
        bot.address: {
            "address": bot.address,
            "role": ROLE_VALIDATOR,
            "wrt": 10 * SCALE,
            "lzn": 0,
            "lzn_activated": 0,
            "ant": 0,
        }
    }
    snap = MarketSnapshot(lzn_price=200, ant_price=50, l_total=SCALE)
    client = AsyncMock()

    with patch("volnix_traffic.engine.actions.place_order", new_callable=AsyncMock) as place:
        place.return_value = True
        ok = await engine._prep(client, [bot], by_addr, snap, {"bids": [], "asks": []})
    assert ok is True
    place.assert_awaited_once()
    kwargs = place.await_args.kwargs
    assert kwargs["market"] == "LZN/WRT"
    assert kwargs["side"] == "BUY"
    assert kwargs["price"] == max(1, int(200 * 0.95))
