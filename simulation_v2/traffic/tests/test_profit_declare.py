"""Unit tests for traffic profit / declare helpers (no live node)."""

from __future__ import annotations

import sys
from fractions import Fraction
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.declare import entry_burn, suggest_declare_bs, suggest_lambda_declare
from volnix_traffic.market import floor_price, miner_reservation_price
from volnix_traffic.profit import (
    ROLE_CITIZEN,
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    SCALE,
    AccountView,
    MarketSnapshot,
    ProfitStrategy,
    snapshot_from_chain,
)


def test_floor_price_and_reservation():
    assert floor_price(1.9) == 1
    assert floor_price(0.1) == 1
    p = miner_reservation_price(
        l_i=SCALE, l_active=SCALE, block_reward=50 * SCALE, lambda_f=1 / 3
    )
    assert p > 1


def test_suggest_declare_matches_node_shape():
    alpha = Fraction(1, 50)
    pair = suggest_declare_bs(SCALE, 2 * SCALE, alpha)
    assert pair is not None
    b_i, s_i = pair
    f_i = entry_burn(SCALE, alpha)
    assert f_i == SCALE // 50
    assert b_i > 0 and s_i > 0
    assert f_i + b_i + s_i <= SCALE
    assert f_i + b_i + s_i <= 2 * SCALE

    lam = Fraction(1, 3)
    pair2 = suggest_lambda_declare(SCALE, 2 * SCALE, lam, alpha)
    assert pair2 is not None
    assert pair2[1] > 0


def test_profit_prefers_verified_when_capital():
    strat = ProfitStrategy()
    snap = MarketSnapshot(
        block_reward=50 * SCALE,
        ant_price=150,
        l_total=SCALE,
        n_validators=1,
        n_suppliers=2,
        epoch_blocks=10080,
        lambda_f=1 / 3,
        ant_sold_hint=100 * SCALE,
    )
    rich = AccountView(address="a", role=ROLE_CITIZEN, wrt=100 * SCALE, lzn=SCALE)
    role = strat.best_first_role(rich, snap)
    assert role in (ROLE_SUPPLIER, ROLE_VALIDATOR)

    poor = AccountView(address="b", role=ROLE_CITIZEN, wrt=0, lzn=0, ant=0)
    # Still may pick supplier/validator aspirationally via desired_role
    desired = strat.desired_role(poor, snap)
    assert desired in (ROLE_CITIZEN, ROLE_SUPPLIER, ROLE_VALIDATOR)


def test_validator_roi_equal_share_not_l_mass():
    """§5.4: reward share is among validators (b_i spirit), not L_i/L_total."""
    strat = ProfitStrategy()
    snap = MarketSnapshot(
        block_reward=50 * SCALE,
        ant_price=1,
        l_total=100 * SCALE,  # huge network mass — must not crush income to ~1%
        n_validators=2,
        n_suppliers=1,
        epoch_blocks=10080,
        lambda_f=1 / 3,
    )
    # Small L_i relative to l_total
    acc = AccountView(
        address="v",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn=0,
        lzn_activated=SCALE,
        ant=10 * SCALE,
    )
    roi = strat._estimate_validator(acc, snap)
    # Equal share among 2 validators over half-epoch would be large vs L-share of 1%.
    assert roi.income_wrt > float(snap.block_reward) * 0.05 * 100


def test_snapshot_from_chain_mid_price():
    summary = {
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
    book = {
        "bids": [{"price": 100, "remaining": 1000}],
        "asks": [{"price": 200, "remaining": 2000}],
    }
    snap = snapshot_from_chain(summary, book)
    assert snap.ant_price == 150
    assert snap.bid_vol == 1000
    assert snap.ask_vol == 2000


@pytest.mark.asyncio
async def test_node_client_paths(httpx_mock=None):
    """Smoke: NodeClient builds expected paths (respx-free stub via httpx ASGI not needed)."""
    from unittest.mock import AsyncMock, MagicMock

    from volnix_traffic.client import NodeClient

    client = NodeClient("http://example.test")
    mock_resp = MagicMock()
    mock_resp.raise_for_status = MagicMock()
    mock_resp.json = MagicMock(return_value={"ok": True, "code": 0, "address": "volnix1abc"})
    client._client.get = AsyncMock(return_value=mock_resp)
    client._client.post = AsyncMock(return_value=mock_resp)

    await client.chain_summary()
    client._client.get.assert_awaited()
    call_path = client._client.get.await_args.args[0]
    assert call_path == "/api/v1/chain/summary"

    await client.create_account("bot-0001-dead")
    post_path = client._client.post.await_args.args[0]
    assert post_path == "/api/v1/operator/account"

    await client.aclose()
