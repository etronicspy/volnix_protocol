from __future__ import annotations

from volnix.app.state import AppState, Order
from volnix.types.events import Event, ev
from volnix.types.msgs import MsgCancelOrder, MsgPlaceOrder
from volnix.types.role import Role

ANT_WRT = "ANT/WRT"
LZN_WRT = "LZN/WRT"


class AnteilError(ValueError):
    pass


def _check_side(role: Role, market: str, side: str) -> None:
    if market not in (ANT_WRT, LZN_WRT):
        raise AnteilError("unknown market")
    if side not in ("BUY", "SELL"):
        raise AnteilError("side must be BUY or SELL")
    if market == ANT_WRT:
        if side == "SELL" and role != Role.SUPPLIER:
            raise AnteilError("only suppliers may sell ANT")
        if side == "BUY" and role not in (Role.SUPPLIER, Role.VALIDATOR):
            raise AnteilError("only supplier or validator may buy ANT")
    else:
        # SELL LZN: any holder (incl. citizen after MOA). BUY: supplier|validator only.
        if side == "BUY" and role not in (Role.SUPPLIER, Role.VALIDATOR):
            raise AnteilError("only supplier or validator may buy LZN")


def deliver_place(state: AppState, msg: MsgPlaceOrder) -> list[Event]:
    acc = state.accounts.get(msg.owner)
    if acc is None:
        raise AnteilError("owner not found")
    _check_side(acc.role, msg.market, msg.side)
    if msg.amount <= 0:
        raise AnteilError("amount must be positive")
    if msg.order_type == "LIMIT" and msg.price <= 0:
        raise AnteilError("limit price must be positive")
    if msg.order_type not in ("LIMIT", "MARKET"):
        raise AnteilError("order_type must be LIMIT or MARKET")

    order_id = f"ord-{state.next_order_id}"
    state.next_order_id += 1
    order = Order(
        order_id=order_id,
        owner=msg.owner,
        market=msg.market,
        side=msg.side,
        order_type=msg.order_type,
        amount=msg.amount,
        price=msg.price,
        created_height=state.height,
        created_index=state.next_order_id,
    )
    _lock_escrow(state, acc, order)
    state.orders[order_id] = order
    events = [
        ev(
            "anteil.order_placed",
            order_id=order_id,
            owner=msg.owner,
            market=msg.market,
            side=msg.side,
            amount=msg.amount,
            price=msg.price,
        )
    ]
    events.extend(_match(state, order))
    return events


def deliver_cancel(state: AppState, msg: MsgCancelOrder) -> list[Event]:
    order = state.orders.get(msg.order_id)
    if order is None or order.status != "open":
        raise AnteilError("order not found or not open")
    if order.owner != msg.owner:
        raise AnteilError("not order owner")
    return cancel_order(state, msg.order_id)


def cancel_order(state: AppState, order_id: str, protocol: bool = False) -> list[Event]:
    order = state.orders.get(order_id)
    if order is None or order.status != "open":
        return []
    _unlock_escrow(state, order)
    order.status = "cancelled"
    return [ev("anteil.order_cancelled", order_id=order_id, protocol=int(protocol))]


def cancel_ant_orders(state: AppState) -> list[Event]:
    events: list[Event] = []
    for oid, order in list(state.orders.items()):
        if order.status == "open" and order.market == ANT_WRT:
            events.extend(cancel_order(state, oid, protocol=True))
    return events


def _lock_escrow(state: AppState, acc, order: Order) -> None:
    if order.side == "SELL":
        if order.market == ANT_WRT:
            if acc.ant < order.amount:
                raise AnteilError("insufficient ANT")
            acc.ant -= order.amount
            order.escrow_base = order.amount
        else:
            if acc.lzn < order.amount:
                raise AnteilError("insufficient free LZN")
            acc.lzn -= order.amount
            order.escrow_base = order.amount
    else:
        quote = order.amount * order.price if order.order_type == "LIMIT" else acc.wrt
        if order.order_type == "MARKET":
            quote = acc.wrt
        if acc.wrt < quote or quote <= 0:
            raise AnteilError("insufficient WRT for buy")
        # For market buy lock all WRT intended; unused returned after match.
        lock = order.amount * order.price if order.order_type == "LIMIT" else quote
        acc.wrt -= lock
        order.escrow_quote = lock


def _unlock_escrow(state: AppState, order: Order) -> None:
    acc = state.ensure_account(order.owner)
    if order.escrow_base > 0:
        if order.market == ANT_WRT:
            acc.ant += order.escrow_base
        else:
            acc.lzn += order.escrow_base
        order.escrow_base = 0
    if order.escrow_quote > 0:
        acc.wrt += order.escrow_quote
        order.escrow_quote = 0


def _book(state: AppState, market: str, side: str) -> list[Order]:
    orders = [
        o
        for o in state.orders.values()
        if o.market == market and o.side == side and o.status == "open" and o.remaining > 0
    ]
    if side == "SELL":
        orders.sort(key=lambda o: (o.price, o.created_height, o.created_index, o.order_id))
    else:
        orders.sort(key=lambda o: (-o.price, o.created_height, o.created_index, o.order_id))
    return orders


def _match(state: AppState, incoming: Order) -> list[Event]:
    events: list[Event] = []
    opposite_side = "BUY" if incoming.side == "SELL" else "SELL"
    book = _book(state, incoming.market, opposite_side)
    for other in book:
        if incoming.remaining <= 0:
            break
        if incoming.order_type == "LIMIT" and other.order_type == "LIMIT":
            if incoming.side == "BUY" and incoming.price < other.price:
                break
            if incoming.side == "SELL" and incoming.price > other.price:
                break
        price = other.price if other.order_type == "LIMIT" else incoming.price
        if price <= 0:
            continue
        qty = min(incoming.remaining, other.remaining)
        if incoming.side == "BUY":
            max_by_quote = incoming.escrow_quote // price if price else 0
            qty = min(qty, max_by_quote)
        if other.side == "BUY":
            max_by_quote = other.escrow_quote // price if price else 0
            qty = min(qty, max_by_quote)
        if qty <= 0:
            continue
        cost = qty * price
        _settle(state, incoming, other, qty, cost, price)
        if incoming.market == ANT_WRT:
            state.last_ant_wrt_price = price
        events.append(
            ev(
                "anteil.trade_executed",
                market=incoming.market,
                price=price,
                amount=qty,
                maker=other.order_id,
                taker=incoming.order_id,
            )
        )
        events.append(ev("anteil.order_matched", order_id=incoming.order_id, amount=qty, price=price))
        events.append(ev("anteil.order_matched", order_id=other.order_id, amount=qty, price=price))
    if incoming.order_type == "MARKET":
        # return unused quote / leftover base and close
        if incoming.remaining > 0:
            _unlock_escrow(state, incoming)
            incoming.status = "cancelled" if incoming.filled == 0 else "filled"
            incoming.filled = incoming.amount  # mark done
        else:
            incoming.status = "filled"
    elif incoming.remaining == 0:
        incoming.status = "filled"
    return events


def _settle(state: AppState, a: Order, b: Order, qty: int, cost: int, price: int) -> None:
    buy, sell = (a, b) if a.side == "BUY" else (b, a)
    buyer = state.ensure_account(buy.owner)
    seller = state.ensure_account(sell.owner)
    if sell.market == ANT_WRT:
        buyer.ant += qty
        sell.escrow_base -= qty
    else:
        buyer.lzn += qty
        sell.escrow_base -= qty
    seller.wrt += cost
    buy.escrow_quote -= cost
    buy.filled += qty
    sell.filled += qty
    if sell.remaining == 0:
        sell.status = "filled"
        if sell.escrow_base > 0:
            _unlock_escrow(state, sell)
    if buy.remaining == 0:
        buy.status = "filled"
        if buy.escrow_quote > 0:
            buyer.wrt += buy.escrow_quote
            buy.escrow_quote = 0
