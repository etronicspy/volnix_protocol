"""Independent EnrichmentAgent instances do not share strategy state."""

from __future__ import annotations

import sys
from fractions import Fraction
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.agent import EnrichmentAgent, balanced_profile
from volnix_traffic.compete import Rival, pick_declare
from volnix_traffic.profit import SCALE
from volnix_traffic.registry import KIND_ENRICHMENT, ROLE_VALIDATOR, BotWallet

ALPHA = Fraction(1, 50)
LAM = Fraction(1, 3)


def _agent(seed: str, address: str) -> EnrichmentAgent:
    w = BotWallet(
        seed=seed,
        address=address,
        role=ROLE_VALIDATOR,
        kind=KIND_ENRICHMENT,
        verified=True,
    )
    return EnrichmentAgent(w, horizon=12)


def test_strategy_mutation_is_private():
    a = _agent("s1", "volnix1aaa")
    b = _agent("s2", "volnix1bbb")
    a.strategy.b_frac = 0.9
    assert b.strategy.b_frac == balanced_profile().b_frac
    assert a.strategy is not b.strategy


def test_different_balances_different_picks():
    rich = pick_declare(
        address="volnix1rich",
        l_i=SCALE,
        ant=2 * SCALE,
        rivals=[],
        alpha=ALPHA,
        lam=LAM,
        k=150,
        subsidy=50 * SCALE,
        fees=0,
        ant_price=1,
    )
    poor = pick_declare(
        address="volnix1poor",
        l_i=SCALE,
        ant=entry_need(),
        rivals=[],
        alpha=ALPHA,
        lam=LAM,
        k=150,
        subsidy=50 * SCALE,
        fees=0,
        ant_price=1,
    )
    assert rich is not None
    assert poor is not None
    assert (rich.b_i, rich.s_i) != (poor.b_i, poor.s_i)


def entry_need() -> int:
    from volnix_traffic.compete import entry_burn

    return entry_burn(SCALE, ALPHA) + 10


def test_no_shared_roi_cache_on_agents():
    a = _agent("s1", "volnix1aaa")
    b = _agent("s2", "volnix1bbb")
    a.expected_wrt = 99
    assert b.expected_wrt == 0
    assert not hasattr(a, "_roi_cache")
