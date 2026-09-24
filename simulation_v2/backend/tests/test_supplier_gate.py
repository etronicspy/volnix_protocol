"""Supplier income gate — canon §5.6."""

from volnix.app.modules.anteil import deliver_place
from volnix.app.modules.epoch import process_boundary
from volnix.app.modules.ident import IdentError, deliver_verify
from volnix.app.modules.supplier_gate import new_supplier_allowed, supplier_income
from volnix.app.state import SUPPLIER_MIN_EPOCH_INCOME, SCALE, Account, AppState, EpochRecord
from volnix.types.msgs import MsgPlaceOrder, MsgVerifyIdentity
from volnix.types.role import Role


def test_supplier_income_formula():
    assert supplier_income(2, 1_000_000, 2) == 2 * (1_000_000 // 2)
    assert supplier_income(1, 100, 0) == 0


def test_gate_not_applicable_without_epoch():
    ok, reason = new_supplier_allowed(last_epoch=None, last_ant_wrt_price=5)
    assert ok and reason == ""


def test_gate_not_applicable_without_price():
    rec = EpochRecord(
        epoch=1, height=3, l_total=SCALE, ant_wiped=0, ant_emit=SCALE * 3, lzn_emit=0, suppliers=["s1"]
    )
    ok, reason = new_supplier_allowed(last_epoch=rec, last_ant_wrt_price=None)
    assert ok and reason == ""


def test_gate_blocks_when_income_at_or_below_floor():
    # share = SCALE; price=1008 → income = 1008 WRT == X → block
    rec = EpochRecord(
        epoch=1,
        height=3,
        l_total=SCALE,
        ant_wiped=0,
        ant_emit=SCALE,
        lzn_emit=0,
        suppliers=["s1"],
    )
    ok, reason = new_supplier_allowed(last_epoch=rec, last_ant_wrt_price=1008)
    assert not ok
    assert reason == "supplier income below floor"
    assert supplier_income(1008, SCALE, 1) == SUPPLIER_MIN_EPOCH_INCOME


def test_gate_allows_when_income_above_floor():
    rec = EpochRecord(
        epoch=1,
        height=3,
        l_total=SCALE,
        ant_wiped=0,
        ant_emit=SCALE,
        lzn_emit=0,
        suppliers=["s1"],
    )
    ok, reason = new_supplier_allowed(last_epoch=rec, last_ant_wrt_price=1009)
    assert ok and reason == ""


def test_verify_before_epoch_allows_supplier():
    st = AppState()
    deliver_verify(
        st,
        MsgVerifyIdentity(address="new", zkp_proof="zkp-1", desired_role="supplier"),
    )
    assert st.accounts["new"].role == Role.SUPPLIER


def test_verify_after_epoch_without_trade_allows():
    st = AppState()
    st.height = 3
    st.params.epoch_blocks = 3
    st.accounts["val"] = Account(
        address="val", role=Role.VALIDATOR, lzn_activated=SCALE, genesis_no_zkp=True
    )
    st.accounts["sup"] = Account(address="sup", role=Role.SUPPLIER)
    process_boundary(st)
    assert st.epochs
    assert st.last_ant_wrt_price is None
    deliver_verify(
        st,
        MsgVerifyIdentity(address="new", zkp_proof="zkp-2", desired_role="supplier"),
    )
    assert st.accounts["new"].role == Role.SUPPLIER


def test_verify_blocked_by_low_income_after_trade():
    st = AppState()
    st.height = 3
    st.params.epoch_blocks = 3
    st.accounts["val"] = Account(
        address="val",
        role=Role.VALIDATOR,
        lzn_activated=SCALE,
        wrt=10 * SCALE,
        genesis_no_zkp=True,
    )
    st.accounts["sup"] = Account(address="sup", role=Role.SUPPLIER, ant=SCALE)
    deliver_place(
        st,
        MsgPlaceOrder(
            owner="sup",
            market="ANT/WRT",
            side="SELL",
            order_type="LIMIT",
            amount=1000,
            price=1,
        ),
    )
    deliver_place(
        st,
        MsgPlaceOrder(
            owner="val",
            market="ANT/WRT",
            side="BUY",
            order_type="LIMIT",
            amount=1000,
            price=1,
        ),
    )
    assert st.last_ant_wrt_price == 1
    process_boundary(st)
    # Control epoch so income = 1 * SCALE == X
    st.epochs[-1] = EpochRecord(
        epoch=1,
        height=3,
        l_total=SCALE,
        ant_wiped=0,
        ant_emit=SCALE,
        lzn_emit=0,
        suppliers=["sup"],
    )
    try:
        deliver_verify(
            st,
            MsgVerifyIdentity(address="new", zkp_proof="zkp-low", desired_role="supplier"),
        )
        raise AssertionError("should reject low income")
    except IdentError as exc:
        assert "supplier income below floor" in str(exc)


def test_verify_allowed_when_income_high():
    st = AppState()
    st.epochs.append(
        EpochRecord(
            epoch=1,
            height=3,
            l_total=SCALE,
            ant_wiped=0,
            ant_emit=SCALE,
            lzn_emit=0,
            suppliers=["sup"],
        )
    )
    st.last_ant_wrt_price = 1009  # income = 1009 * SCALE > X (1008 WRT)
    deliver_verify(
        st,
        MsgVerifyIdentity(address="new", zkp_proof="zkp-hi", desired_role="supplier"),
    )
    assert st.accounts["new"].role == Role.SUPPLIER


def test_slots_full_blocks_regardless_of_income():
    st = AppState()
    st.params.max_active_suppliers = 1
    st.accounts["sup"] = Account(address="sup", role=Role.SUPPLIER)
    st.epochs.append(
        EpochRecord(
            epoch=1,
            height=3,
            l_total=SCALE,
            ant_wiped=0,
            ant_emit=SCALE,
            lzn_emit=0,
            suppliers=["sup"],
        )
    )
    st.last_ant_wrt_price = 100  # would pass income gate
    try:
        deliver_verify(
            st,
            MsgVerifyIdentity(address="new", zkp_proof="zkp-slot", desired_role="supplier"),
        )
        raise AssertionError("should reject full slots")
    except IdentError as exc:
        assert "supplier slots full" in str(exc)


def test_snapshot_preserves_last_ant_wrt_price():
    st = AppState()
    st.last_ant_wrt_price = 42
    snap = st.to_snapshot()
    st2 = AppState.from_snapshot(snap)
    assert st2.last_ant_wrt_price == 42
