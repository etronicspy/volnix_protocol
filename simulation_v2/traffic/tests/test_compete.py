"""Competitive declare picker (no live node)."""

from __future__ import annotations

import sys
from fractions import Fraction
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.compete import Rival, entry_burn, pick_declare, simulate_enter
from volnix_traffic.profit import SCALE

ALPHA = Fraction(1, 50)
LAM = Fraction(1, 3)
ADDR = "volnix1self"


def test_raise_s_to_enter_against_high_weight_rival():
    # top-K=2: two heavier rivals occupy the set unless we raise s_i (w_i).
    r1 = Rival(address="volnix1a", l_i=SCALE, b_i=400_000, s_i=400_000)
    r2 = Rival(address="volnix1b", l_i=SCALE, b_i=400_000, s_i=350_000)
    pick = pick_declare(
        address=ADDR,
        l_i=SCALE,
        ant=2 * SCALE,
        rivals=[r1, r2],
        alpha=ALPHA,
        lam=LAM,
        k=2,
        subsidy=50 * SCALE,
        fees=0,
        ant_price=1,
        b_frac=0.70,
        s_frac=0.20,
    )
    assert pick is not None
    assert pick.enter is True
    assert pick.s_i / SCALE > 0.35


def test_raise_b_for_reward_without_self_exclusion():
    rival = Rival(address="volnix1rival", l_i=SCALE, b_i=400_000, s_i=400_000)
    pick = pick_declare(
        address=ADDR,
        l_i=SCALE,
        ant=2 * SCALE,
        rivals=[rival],
        alpha=ALPHA,
        lam=LAM,
        k=150,
        subsidy=50 * SCALE,
        fees=0,
        ant_price=1,
        b_frac=0.50,
        s_frac=0.40,
    )
    assert pick is not None
    assert pick.enter is True
    # Prefer a meaningful burn share, not s-only.
    assert pick.b_i > 0
    enter, _ = simulate_enter(ADDR, pick.b_i, pick.s_i, SCALE, [rival], ALPHA, LAM, 150)
    assert enter is True


def test_lift_b_to_meet_b_min_solo():
    pick = pick_declare(
        address=ADDR,
        l_i=SCALE,
        ant=2 * SCALE,
        rivals=[],
        alpha=ALPHA,
        lam=LAM,
        k=150,
        subsidy=50 * SCALE,
        fees=0,
        ant_price=1,
        b_frac=0.10,
        s_frac=0.80,
    )
    assert pick is not None
    assert pick.enter is True
    # B_min at L_decl=1e6, λ=1/3 is 333334
    assert pick.b_i >= 333_334


def test_insufficient_ant_returns_valid_or_none():
    f_i = entry_burn(SCALE, ALPHA)
    pick = pick_declare(
        address=ADDR,
        l_i=SCALE,
        ant=f_i + 5,
        rivals=[],
        alpha=ALPHA,
        lam=LAM,
        k=150,
        subsidy=50 * SCALE,
        fees=0,
        ant_price=1,
    )
    if pick is None:
        return
    assert pick.s_i > 0
    assert pick.f_i + pick.b_i + pick.s_i <= SCALE
    assert pick.f_i + pick.b_i + pick.s_i <= f_i + 5
