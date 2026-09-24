"""Horizon forecast: validator reward only on enter; supplier wipe risk."""

from __future__ import annotations

import sys
from fractions import Fraction
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.compete import Rival
from volnix_traffic.enrichment import forecast_supplier, forecast_validator
from volnix_traffic.profit import (
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    SCALE,
    AccountView,
    MarketSnapshot,
)

ALPHA = Fraction(1, 50)
LAM = Fraction(1, 3)


def test_validator_reward_only_when_enter():
    acc = AccountView(
        address="volnix1v",
        role=ROLE_VALIDATOR,
        wrt=10 * SCALE,
        lzn_activated=SCALE,
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
        alpha=ALPHA,
        lam=LAM,
        k=150,
        b_frac=0.5,
        s_frac=0.4,
    )
    assert fc.enter is True
    assert fc.net > 0

    # High-weight rival + K=1 can lock us out if we cannot beat w_i
    rival = Rival(address="volnix1boss", l_i=SCALE, b_i=100_000, s_i=900_000)
    locked = AccountView(
        address="volnix1v",
        role=ROLE_VALIDATOR,
        lzn_activated=SCALE,
        ant=entry_tiny(),
    )
    miss = forecast_validator(
        locked,
        snap,
        rivals=[rival],
        horizon=12,
        alpha=ALPHA,
        lam=LAM,
        k=1,
        b_frac=0.5,
        s_frac=0.4,
    )
    if not miss.enter:
        assert miss.income_wrt == 0


def entry_tiny() -> int:
    from volnix_traffic.compete import entry_burn

    return entry_burn(SCALE, ALPHA) + 2


def test_supplier_wipe_risk_near_epoch():
    acc = AccountView(address="volnix1s", role=ROLE_SUPPLIER, ant=100 * SCALE)
    snap = MarketSnapshot(
        height=10_070,
        epoch_blocks=10_080,
        ant_price=10,
        l_total=SCALE,
        n_suppliers=2,
        lambda_f=1 / 3,
        bid_vol=1,
        ask_vol=1,
        ant_sold_hint=50 * SCALE,
    )
    far_snap = MarketSnapshot(
        height=10,
        epoch_blocks=10_080,
        ant_price=10,
        l_total=SCALE,
        n_suppliers=2,
        lambda_f=1 / 3,
        bid_vol=1,
        ask_vol=1,
        ant_sold_hint=50 * SCALE,
    )
    far = forecast_supplier(acc, far_snap, horizon=5)
    near = forecast_supplier(acc, snap, horizon=20)
    assert near.risk_wrt >= far.risk_wrt
