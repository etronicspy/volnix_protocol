"""LZN activation share cap vs L_total — mirrors Cosmos x/lizenz."""

from __future__ import annotations

import pytest

from volnix.app.modules.lizenz import (
    LizenzError,
    activation_headroom,
    deliver_activate,
    max_activated_per_validator,
)
from volnix.app.state import SCALE, Account, AppState
from volnix.types.msgs import MsgActivateLZN
from volnix.types.role import Role


def test_max_activated_floor_div():
    assert max_activated_per_validator(0) == 0
    assert max_activated_per_validator(10) == 3
    assert max_activated_per_validator(10_000_000) == 3_333_333


def test_sole_validator_may_grow_without_share_cap():
    st = AppState()
    st.accounts["v1"] = Account(
        address="v1",
        role=Role.VALIDATOR,
        lzn=10 * SCALE,
        lzn_activated=SCALE,
    )
    # Sole holder: activating more must succeed (new would be > ⌊newTotal/3⌋ without exemption).
    deliver_activate(st, MsgActivateLZN(validator="v1", amount=2 * SCALE))
    assert st.accounts["v1"].lzn_activated == 3 * SCALE


def test_second_validator_capped_at_floor_new_l_total_over_3():
    st = AppState()
    st.accounts["v1"] = Account(
        address="v1", role=Role.VALIDATOR, lzn_activated=10 * SCALE
    )
    st.accounts["v2"] = Account(
        address="v2", role=Role.VALIDATOR, lzn=10 * SCALE, lzn_activated=0
    )
    # newTotal = 10M + X; X ≤ ⌊(10M+X)/3⌋ → X ≤ 5M
    deliver_activate(st, MsgActivateLZN(validator="v2", amount=5 * SCALE))
    assert st.accounts["v2"].lzn_activated == 5 * SCALE

    st.accounts["v2"].lzn = 10 * SCALE
    with pytest.raises(LizenzError, match="one-third of L_total"):
        deliver_activate(st, MsgActivateLZN(validator="v2", amount=SCALE))


def test_exceeds_when_second_takes_too_much():
    st = AppState()
    st.accounts["v1"] = Account(
        address="v1", role=Role.VALIDATOR, lzn_activated=6 * SCALE
    )
    st.accounts["v2"] = Account(
        address="v2", role=Role.VALIDATOR, lzn=10 * SCALE, lzn_activated=0
    )
    # newTotal=6M+5M=11M → max ⌊11M/3⌋=3M; 5M exceeds
    with pytest.raises(LizenzError, match="one-third of L_total"):
        deliver_activate(st, MsgActivateLZN(validator="v2", amount=5 * SCALE))


def test_headroom_matches_cap():
    st = AppState()
    st.accounts["v1"] = Account(
        address="v1", role=Role.VALIDATOR, lzn_activated=10 * SCALE
    )
    st.accounts["v2"] = Account(
        address="v2", role=Role.VALIDATOR, lzn=100 * SCALE, lzn_activated=0
    )
    assert activation_headroom(st, "v2") == 5 * SCALE
