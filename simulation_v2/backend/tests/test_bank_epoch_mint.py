from volnix.app.modules.anteil import deliver_place
from volnix.app.modules.bank import BankError, deliver_send
from volnix.app.modules.epoch import (
    is_boundary,
    lzn_epoch_amount,
    process_boundary,
    remaining_boundaries,
    split_even,
)
from volnix.app.modules.ident import check_moa
from volnix.app.modules.mint import block_reward, era_at
from volnix.app.state import LZN_HALVING_ERAS, SCALE, Account, AppState
from volnix.types.msgs import MsgPlaceOrder, MsgSend
from volnix.types.role import Role


def test_msgsend_rejects_ant_and_lzn():
    st = AppState()
    st.accounts["a"] = Account(address="a", wrt=10, ant=10, lzn=10, role=Role.SUPPLIER)
    st.accounts["b"] = Account(address="b", role=Role.CITIZEN)
    try:
        deliver_send(st, MsgSend(from_address="a", to_address="b", denom="uant", amount=1))
        raise AssertionError("should reject ANT")
    except BankError:
        pass
    try:
        deliver_send(st, MsgSend(from_address="a", to_address="b", denom="ulzn", amount=1))
        raise AssertionError("should reject LZN")
    except BankError:
        pass
    deliver_send(st, MsgSend(from_address="a", to_address="b", denom="uwrt", amount=3))
    assert st.accounts["a"].wrt == 7
    assert st.accounts["b"].wrt == 3


def test_citizen_may_sell_lzn():
    st = AppState()
    st.height = 1
    st.accounts["c"] = Account(address="c", role=Role.CITIZEN, lzn=SCALE, wrt=0)
    events = deliver_place(
        st,
        MsgPlaceOrder(
            owner="c",
            market="LZN/WRT",
            side="SELL",
            order_type="LIMIT",
            amount=SCALE // 2,
            price=10,
        ),
    )
    assert any(e.type == "anteil.order_placed" for e in events)
    assert st.accounts["c"].lzn == SCALE // 2


def test_citizen_cannot_buy_lzn():
    st = AppState()
    st.accounts["c"] = Account(address="c", role=Role.CITIZEN, wrt=SCALE)
    try:
        deliver_place(
            st,
            MsgPlaceOrder(
                owner="c",
                market="LZN/WRT",
                side="BUY",
                order_type="LIMIT",
                amount=1,
                price=1,
            ),
        )
        raise AssertionError("should reject")
    except Exception as exc:
        assert "buy LZN" in str(exc)


def test_moa_unfreezes_lzn_for_sell():
    st = AppState()
    st.height = 100
    acc = Account(
        address="v",
        role=Role.VALIDATOR,
        lzn_activated=SCALE,
        ant=50,
        last_tx_height=0,
        created_height=0,
    )
    st.accounts["v"] = acc
    st.params.moa_validator_window = 10
    st.height = 20
    check_moa(st)
    assert st.accounts["v"].role == Role.CITIZEN
    assert st.accounts["v"].lzn_activated == 0
    assert st.accounts["v"].lzn == SCALE
    assert st.accounts["v"].ant == 0


def test_epoch_wipe_and_ant_emit():
    st = AppState()
    st.height = 3
    st.params.epoch_blocks = 3
    st.accounts["val"] = Account(
        address="val", role=Role.VALIDATOR, lzn_activated=SCALE, ant=100, genesis_no_zkp=True
    )
    st.accounts["sup"] = Account(address="sup", role=Role.SUPPLIER, ant=50)
    events = process_boundary(st)
    assert st.accounts["val"].ant == 100
    assert st.accounts["sup"].ant == SCALE * 3
    types = [e.type for e in events]
    assert "anteil.epoch_reset" in types
    assert st.epochs[-1].ant_wiped == 50
    assert st.epochs[-1].ant_emit == SCALE * 3


def test_lzn_emit_schedule_genesis_numbers():
    h_end = 33 * 2_100_000
    epoch_blocks = 10_080
    assert h_end == 69_300_000
    n = h_end // epoch_blocks
    assert n == 6875
    r0 = 999_999_999
    q, r = divmod(r0, n)
    assert q == 145_454
    assert r == 3_749
    assert q * n + r == r0
    n_rem = remaining_boundaries(10_080, epoch_blocks, h_end)
    assert n_rem == 6875
    assert lzn_epoch_amount(r0, n_rem) == 145_455
    assert is_boundary(10_080, 10_080)
    assert not is_boundary(0, 10_080)
    assert not is_boundary(1, 10_080)


def test_split_even_remainder_by_address():
    shares = split_even(5, ["z", "a", "m"])
    assert shares["a"] == 2
    assert shares["m"] == 2
    assert shares["z"] == 1


def test_halving_and_zero_after_33():
    base = 50_000_000
    interval = 2_100_000
    assert era_at(1, interval) == 0
    assert era_at(interval, interval) == 0
    assert era_at(interval + 1, interval) == 1
    assert block_reward(1, base, interval) == base
    assert block_reward(interval + 1, base, interval) == base // 2
    assert block_reward(33 * interval + 1, base, interval) == 0
    assert LZN_HALVING_ERAS == 33
