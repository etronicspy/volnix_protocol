"""Тесты KPI-аналитики `core.analytics`."""
from __future__ import annotations

from core import analytics
from core.models import Role
from core.state import SIM_TREASURY_ADDR


def test_gini_zero_when_empty():
    assert analytics.gini_coefficient([]) == 0.0
    assert analytics.gini_coefficient([0, 0, 0]) == 0.0


def test_gini_zero_when_perfectly_equal():
    assert analytics.gini_coefficient([10, 10, 10, 10]) < 1e-9


def test_gini_high_when_concentrated():
    # Gini для n=100 значений с одним богатым ≈ (n-1)/n = 0.99
    arr = [0.0001] * 99 + [1_000_000.0]
    val = analytics.gini_coefficient(arr)
    assert 0.9 < val < 1.0


def test_gini_monotone_with_concentration():
    """Чем сильнее концентрация — тем выше Gini (при одинаковом N)."""
    equal = analytics.gini_coefficient([100, 100, 100, 100])
    skewed = analytics.gini_coefficient([1, 1, 1, 1000])
    assert skewed > equal


def test_balances_by_asset_skips_treasury(state_manager, mk_account):
    a = mk_account("alice", role=Role.CITIZEN)
    a.wrt_balance = 100
    state_manager.accounts[SIM_TREASURY_ADDR].wrt_balance = 999999
    balances = analytics.balances_by_asset(state_manager, "wrt")
    assert SIM_TREASURY_ADDR not in balances
    assert balances.get("alice") == 100


def test_total_supply_excludes_treasury(state_manager, mk_account):
    state_manager.accounts[SIM_TREASURY_ADDR].ant_balance = 0
    before = analytics.total_supply(state_manager, "ant")
    mk_account("p1", role=Role.PROVIDER).ant_balance = 500
    mk_account("p2", role=Role.PROVIDER).ant_balance = 250
    after = analytics.total_supply(state_manager, "ant")
    assert after - before == 750


def test_role_counts_basic(state_manager, mk_account):
    mk_account("c1")
    mk_account("p1", role=Role.PROVIDER)
    mk_account("v1", role=Role.VALIDATOR)
    counts = analytics.role_counts(state_manager)
    # Genesis ProviderAddr + VAlidator уже считаются. Просто проверим что наши добавились
    assert counts["citizen"] >= 1
    assert counts["provider"] >= 2  # genesis_provider + p1
    assert counts["validator"] >= 2  # genesis_validator + v1
    assert counts["treasury"] == 1


def test_accepted_block_ratio_no_rejects_is_one(state_manager):
    # Свежий genesis — попыток 1 (genesis-блок), отказов 0
    assert analytics.accepted_block_ratio(state_manager) == 1.0


def test_burn_ratio_zero_when_no_emission(state_manager):
    burned, emission, ratio = analytics.burn_ratio(state_manager)
    assert burned == 0.0
    assert emission == 0.0
    assert ratio == 0.0


def test_burn_ratio_nonzero_when_sales(state_manager):
    state_manager.current_epoch_burn = 50.0
    state_manager.epoch_ant_sold_volume = 100.0
    state_manager.epoch_emission_coefficient = 1.0
    burned, emission, ratio = analytics.burn_ratio(state_manager)
    assert burned == 50.0
    assert emission == 100.0
    assert ratio == 0.5


def test_kpi_snapshot_has_expected_fields(state_manager):
    snap = analytics.kpi_snapshot(state_manager)
    for key in (
        "height",
        "blocks_per_epoch",
        "supply",
        "gini",
        "velocity",
        "burn",
        "accepted_block_ratio",
        "avg_match_spread",
        "role_counts",
        "consensus_validator_count",
        "mempool_size",
        "last_price",
    ):
        assert key in snap
    for asset in ("wrt", "lzn", "ant"):
        assert asset in snap["supply"]
        assert asset in snap["gini"]
