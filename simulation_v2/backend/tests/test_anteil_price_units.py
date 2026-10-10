"""Order price is micro-WRT per whole ANT/LZN (fixed-point)."""

from volnix.app.modules.anteil import deliver_place, max_qty_for_quote, quote_cost
from volnix.app.state import SCALE, Account, AppState
from volnix.types.msgs import MsgPlaceOrder
from volnix.types.role import Role


def test_quote_cost_and_affordable():
    assert quote_cost(SCALE, SCALE) == SCALE  # 1 ANT at 1 WRT
    assert quote_cost(SCALE, 250_000) == 250_000  # 0.25 WRT/ANT
    assert quote_cost(SCALE // 2, SCALE) == SCALE // 2
    assert max_qty_for_quote(250_000, 250_000) == SCALE
    assert max_qty_for_quote(SCALE, SCALE) == SCALE


def test_trade_below_one_wrt_per_ant():
    st = AppState()
    st.accounts["sup"] = Account(address="sup", role=Role.SUPPLIER, ant=SCALE)
    st.accounts["val"] = Account(
        address="val", role=Role.VALIDATOR, wrt=SCALE, lzn_activated=SCALE
    )
    px = 250_000  # 0.25 WRT per ANT
    deliver_place(
        st,
        MsgPlaceOrder(
            owner="sup",
            market="ANT/WRT",
            side="SELL",
            order_type="LIMIT",
            amount=SCALE,
            price=px,
        ),
    )
    deliver_place(
        st,
        MsgPlaceOrder(
            owner="val",
            market="ANT/WRT",
            side="BUY",
            order_type="LIMIT",
            amount=SCALE,
            price=px,
        ),
    )
    assert st.last_ant_wrt_price == px
    assert st.accounts["sup"].wrt == 250_000
    assert st.accounts["val"].ant == SCALE
    assert st.accounts["val"].wrt == SCALE - 250_000
