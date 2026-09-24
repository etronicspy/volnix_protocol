"""Spontaneous 5–10 peer sample; adopt only if strictly better."""

from __future__ import annotations

import random
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.agent import EnrichmentAgent, StrategyCard
from volnix_traffic.enrichment import sample_peer_indices
from volnix_traffic.registry import KIND_ENRICHMENT, ROLE_VALIDATOR, BotWallet


def _card(addr: str, growth: int, expected: int = 0, b_frac: float = 0.5) -> StrategyCard:
    return StrategyCard(
        address=addr,
        role=ROLE_VALIDATOR,
        b_frac=b_frac,
        s_frac=0.4,
        activate_ratio=1.0,
        ask_shade=1.0,
        bid_shade=0.95,
        desired_role=ROLE_VALIDATOR,
        growth_wrt=growth,
        expected_wrt=expected,
    )


def test_sample_size_and_excludes_self():
    rng = random.Random(1)
    idxs = sample_peer_indices(20, 5, 10, rng)
    assert 5 <= len(idxs) <= 10
    assert len(set(idxs)) == len(idxs)
    assert all(0 <= i < 20 for i in idxs)


def test_adopt_only_if_peer_strictly_better():
    wallet = BotWallet(
        seed="me",
        address="volnix1me",
        role=ROLE_VALIDATOR,
        kind=KIND_ENRICHMENT,
    )
    ag = EnrichmentAgent(wallet, rng=random.Random(0))
    ag.growth_wrt = 10
    ag.expected_wrt = 1
    board = [
        _card("volnix1me", 10_000),  # self, must be ignored
        _card("volnix1a", 5, b_frac=0.2),
        _card("volnix1b", 50, b_frac=0.8),
        _card("volnix1c", 8, b_frac=0.3),
        _card("volnix1d", 7),
        _card("volnix1e", 6),
        _card("volnix1f", 4),
    ]
    best = ag.scout(board)
    assert best is not None
    assert best.address != "volnix1me"
    assert best.growth_wrt > ag.growth_wrt
    before = ag.strategy.b_frac
    ag.adopt(best)
    assert ag.strategy.b_frac == best.b_frac
    assert ag.adopted_from == best.address
    assert before != ag.strategy.b_frac or best.b_frac == before


def test_keep_own_profile_when_sample_not_better():
    wallet = BotWallet(seed="me", address="volnix1me", role=ROLE_VALIDATOR, kind=KIND_ENRICHMENT)
    ag = EnrichmentAgent(wallet, rng=random.Random(2))
    ag.growth_wrt = 1_000
    ag.strategy.b_frac = 0.33
    board = [_card(f"volnix1p{i}", 10) for i in range(8)]
    assert ag.scout(board) is None
    assert ag.strategy.b_frac == 0.33


def test_board_card_is_snapshot():
    wallet = BotWallet(seed="me", address="volnix1me", role=ROLE_VALIDATOR, kind=KIND_ENRICHMENT)
    ag = EnrichmentAgent(wallet)
    card = ag.card()
    ag.strategy.b_frac = 0.11
    assert card.b_frac == 0.50
