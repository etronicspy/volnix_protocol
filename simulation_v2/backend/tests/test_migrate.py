"""MsgMigrateRole — canon §3.2 (headroom + new ZKP nullifier)."""

from __future__ import annotations

import pytest

from volnix.app.modules.ident import IdentError, deliver_migrate
from volnix.app.state import SCALE, Account, AppState
from volnix.types.msgs import MsgMigrateRole
from volnix.types.role import Role


def test_migrate_binds_new_zkp_not_src():
    st = AppState()
    st.height = 5
    st.accounts["src"] = Account(
        address="src",
        role=Role.SUPPLIER,
        zkp_id="old-zkp",
        ant=100,
        lzn=0,
        lzn_activated=SCALE,
    )
    st.nullifiers.add("old-zkp")
    st.ensure_account("dst")

    with pytest.raises(IdentError, match="new zkp"):
        deliver_migrate(
            st,
            MsgMigrateRole(from_address="src", to_address="dst", zkp_proof="old-zkp"),
        )

    deliver_migrate(
        st,
        MsgMigrateRole(from_address="src", to_address="dst", zkp_proof="new-zkp"),
    )
    assert st.accounts["src"].role == Role.CITIZEN
    assert st.accounts["src"].zkp_id == ""
    assert st.accounts["dst"].role == Role.SUPPLIER
    assert st.accounts["dst"].zkp_id == "new-zkp"
    assert st.accounts["dst"].lzn_activated == SCALE
    assert "old-zkp" not in st.nullifiers
    assert "new-zkp" in st.nullifiers


def test_migrate_headroom_caps_activated():
    st = AppState()
    st.height = 1
    # Cap = floor(supply/3)*SCALE; set tiny supply so room is small
    st.params.lzn_total_supply = 3  # max frozen per addr = 1 * SCALE
    max_frozen = st.params.lzn_max_frozen_per_address
    assert max_frozen == SCALE

    st.accounts["src"] = Account(
        address="src",
        role=Role.VALIDATOR,
        zkp_id="zkp-src",
        lzn=100,
        lzn_activated=2 * SCALE,  # over headroom
        ant=10,
    )
    st.nullifiers.add("zkp-src")
    dst = st.ensure_account("dst")
    dst.lzn_activated = 0

    deliver_migrate(
        st,
        MsgMigrateRole(from_address="src", to_address="dst", zkp_proof="zkp-dst"),
    )
    assert st.accounts["dst"].lzn_activated == max_frozen
    # excess activated became free LZN (+ original free)
    assert st.accounts["dst"].lzn == 100 + SCALE


def test_migrate_rejects_used_nullifier():
    st = AppState()
    st.accounts["src"] = Account(
        address="src", role=Role.SUPPLIER, zkp_id="a", lzn_activated=SCALE
    )
    st.nullifiers.add("a")
    st.nullifiers.add("taken")
    st.ensure_account("dst")
    with pytest.raises(IdentError, match="already used"):
        deliver_migrate(
            st,
            MsgMigrateRole(from_address="src", to_address="dst", zkp_proof="taken"),
        )
