"""DAO change of EpochBlocks / HalvingInterval recomputes LZN horizon, keeps R (§5.5)."""

from __future__ import annotations

from volnix.app.modules.epoch import (
    lzn_epoch_amount,
    lzn_horizon,
    process_boundary,
    remaining_boundaries,
)
from volnix.app.modules.gov import _apply
from volnix.app.state import LZN_HALVING_ERAS, SCALE, Account, AppState, Proposal
from volnix.types.role import Role


def _prop(changes: dict) -> Proposal:
    return Proposal(
        proposal_id=1,
        proposer="p",
        title="period",
        description="",
        parameter_changes=changes,
        deposit=0,
        submit_height=1,
        voting_end=2,
        execute_height=3,
    )


def test_epoch_blocks_change_keeps_pool_recomputes_n_rem():
    st = AppState()
    st.height = 5_000
    st.params.epoch_blocks = 10_080
    st.params.halving_interval = 2_100_000
    pool = st.lzn_pool_remaining
    events = _apply(st, _prop({"epoch_blocks": 5_040}))
    assert st.params.epoch_blocks == 5_040
    assert st.lzn_pool_remaining == pool
    h_end, n_rem = lzn_horizon(st)
    assert h_end == LZN_HALVING_ERAS * 2_100_000
    assert n_rem == remaining_boundaries(5_040, 5_040, h_end)
    assert any(e.type == "anteil.lzn_horizon_recomputed" for e in events)
    rec = next(e for e in events if e.type == "anteil.lzn_horizon_recomputed")
    attrs = {a.key: a.value for a in rec.attributes}
    assert attrs["pool_remaining"] == str(pool)
    assert attrs["n_rem"] == str(n_rem)


def test_next_boundary_uses_new_quota():
    st = AppState()
    st.height = 5_000
    st.params.epoch_blocks = 10_080
    st.lzn_pool_remaining = 10_000
    _apply(st, _prop({"epoch_blocks": 5_040}))
    st.accounts["s"] = Account(address="s", role=Role.SUPPLIER)
    st.height = 5_040
    h_end, _ = lzn_horizon(st)
    n_rem = remaining_boundaries(5_040, 5_040, h_end)
    expect = lzn_epoch_amount(10_000, n_rem)
    events = process_boundary(st)
    assert st.lzn_pool_remaining == 10_000 - expect
    assert st.accounts["s"].lzn == expect * SCALE
    assert any(
        e.type == "anteil.epoch_emission"
        and any(a.key == "denom" and a.value == "ulzn" for a in e.attributes)
        for e in events
    )


def test_halving_interval_change_new_h_end():
    st = AppState()
    st.height = 100
    st.params.halving_interval = 2_100_000
    pool = st.lzn_pool_remaining
    _apply(st, _prop({"halving_interval": 1_000_000}))
    assert st.lzn_pool_remaining == pool
    h_end, n_rem = lzn_horizon(st)
    assert h_end == LZN_HALVING_ERAS * 1_000_000
    assert n_rem > 0
