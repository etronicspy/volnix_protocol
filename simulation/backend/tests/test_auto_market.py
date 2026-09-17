"""§5.2: эталонный рынок ANT — адаптивная цена Поставщика и докуп Валидатора."""
from __future__ import annotations

import time
import uuid

import pytest

from core import auto_market
from core.models import Order, OrderType, Role
from core.state import GENESIS_VALIDATOR_ADDR


def _mk_order(sm, owner: str, side: OrderType, price: float, amount: float) -> str:
    oid = uuid.uuid4().hex
    sm.orders[oid] = Order(
        id=oid, owner=owner, order_type=side,
        price=price, amount=amount, filled=0.0, timestamp=time.time(),
    )
    return oid


def test_miner_reservation_is_reward_share_over_burn(engine, mk_account):
    """Предел покупателя = награда за блок / ANT, сжигаемый активными майнерами."""
    sm = engine.state
    gv = sm.accounts[GENESIS_VALIDATOR_ADDR]
    gv.ant_balance = 1000.0
    expected = auto_market.BLOCK_REWARD_WRT / (
        auto_market.BURN_CAP_LAMBDA * gv.lzn_frozen_mining
    )
    assert auto_market.miner_reservation_price(sm, gv) == pytest.approx(expected)


def test_reservation_rises_when_rivals_go_idle(engine, mk_account):
    """Майнер без ANT выпадает из L_active — предел оставшихся растёт.

    Это и есть саморегуляция: внешнего потолка цены нет, его двигает рынок.
    """
    sm = engine.state
    gv = sm.accounts[GENESIS_VALIDATOR_ADDR]
    gv.ant_balance = 1000.0
    rival = mk_account("v_rival", role=Role.VALIDATOR, frozen=6667.0, zkp=True)

    rival.ant_balance = 1000.0
    with_rival = auto_market.miner_reservation_price(sm, gv)
    rival.ant_balance = 0.0  # соперник встал — жечь ему нечем
    alone = auto_market.miner_reservation_price(sm, gv)

    assert alone > with_rival


def test_demand_pressure_reflects_book_imbalance(engine, mk_account):
    sm = engine.state
    v = mk_account("v_p", role=Role.VALIDATOR, frozen=10.0, wrt=500.0, zkp=True)
    p = mk_account("p_p", role=Role.PROVIDER, ant=500.0, zkp=True)

    assert auto_market.demand_pressure(sm) == 0.0
    _mk_order(sm, p.address, OrderType.SELL, 0.02, 100.0)
    assert auto_market.demand_pressure(sm) == pytest.approx(0.0)
    _mk_order(sm, v.address, OrderType.BUY, 0.02, 300.0)
    assert auto_market.demand_pressure(sm) == pytest.approx(0.75)


def test_ask_probes_upward_when_lot_is_taken(engine, mk_account):
    """За окно пересмотра была выручка → Поставщик пробует дороже."""
    auto_market.reset_quotes()
    sm = engine.state
    sm.last_price = 0.02
    sm.current_height = 100
    p = mk_account("p_up", role=Role.PROVIDER, ant=500.0, zkp=True)
    p.wrt_balance = 0.0

    first = auto_market.provider_ask_price(sm, p, bootstrap=0.02)

    # Внутри окна котировка не шевелится, даже если уже что-то продано.
    p.wrt_balance = 5.0
    sm.current_height = 100 + auto_market.REVIEW_INTERVAL_BLOCKS - 1
    assert auto_market.provider_ask_price(sm, p, bootstrap=0.02) == first

    sm.current_height = 100 + auto_market.REVIEW_INTERVAL_BLOCKS
    assert auto_market.provider_ask_price(sm, p, bootstrap=0.02) > first


def test_ask_has_no_ceiling(engine, mk_account):
    """Цена продавца ничем не ограничена — ни здесь, ни в консенсусе."""
    sm = engine.state
    p = mk_account("p_free2", role=Role.PROVIDER, ant=500.0, zkp=True)
    reservation = auto_market.miner_reservation_price(
        sm, sm.accounts[GENESIS_VALIDATOR_ADDR]
    )

    sm.last_price = reservation * 100.0
    ask = auto_market.provider_ask_price(sm, p, bootstrap=0.02)
    assert ask > reservation


def test_miner_never_bids_above_own_reservation(engine, mk_account):
    """Регулятор цены — отказ покупателя, а не правило у продавца."""
    sm = engine.state
    p = mk_account("p_greedy", role=Role.PROVIDER, ant=10_000.0, zkp=True)
    gv = sm.accounts[GENESIS_VALIDATOR_ADDR]
    gv.ant_balance = 0.0
    gv.wrt_balance = 1_000_000.0
    reservation = auto_market.miner_reservation_price(sm, gv)
    # Продавец просит втрое выше предела майнера.
    _mk_order(sm, p.address, OrderType.SELL, reservation * 3.0, 5_000.0)

    auto_market.step_once(sm, engine)
    buys = [
        tx for tx in sm.mempool
        if tx.tx_type.value == "create_order" and tx.order_type.value == "buy"
    ]
    assert buys
    assert all(tx.price <= reservation + 1e-9 for tx in buys)


def test_ask_steps_down_until_the_offer_is_taken(engine, mk_account):
    """Выручки за окно не было → цена идёт вниз. Противостояния не возникает.

    Себестоимость ANT нулевая, а непроданный остаток сгорает на границе §5.5,
    поэтому не продать всегда хуже, чем продать дёшево: Поставщик уступает первым.
    Шаг вниз симметричен шагу вверх и не зависит от размера запаса — скидка не
    увеличивает объём, спрос по количеству неэластичен.
    """
    auto_market.reset_quotes()
    sm = engine.state
    sm.last_price = 0.02
    sm.current_height = 100
    p = mk_account("p_step", role=Role.PROVIDER, ant=500.0, zkp=True)
    p.wrt_balance = 0.0
    _mk_order(sm, p.address, OrderType.SELL, 0.02, 1000.0)

    first = auto_market.provider_ask_price(sm, p, bootstrap=0.02)
    sm.current_height = 100 + auto_market.REVIEW_INTERVAL_BLOCKS
    down = auto_market.provider_ask_price(sm, p, bootstrap=0.02)
    assert down < first

    # Размер запаса на шаг не влияет.
    p.ant_balance *= 1000.0
    sm.current_height += auto_market.REVIEW_INTERVAL_BLOCKS
    assert auto_market.provider_ask_price(sm, p, bootstrap=0.02) == pytest.approx(
        down * (1.0 - auto_market.ASK_STEP), rel=1e-3
    )


def test_quote_holds_through_requote_when_sales_continue(engine, mk_account):
    """Перестановка заявки не стирает память о продажах.

    `filled` у ордера обнуляется при перевыставлении котировки — если читать
    сигнал спроса оттуда, каждая удачная сделка сразу после requote выглядит как
    «не берут», и цена уезжает вниз храповиком.
    """
    auto_market.reset_quotes()
    sm = engine.state
    sm.last_price = 0.02
    sm.current_height = 10
    p = mk_account("p_ratchet", role=Role.PROVIDER, ant=500.0, zkp=True)
    p.wrt_balance = 0.0

    prev = auto_market.provider_ask_price(sm, p, bootstrap=0.02)
    for _ in range(5):
        # Заявку сняли и выставили заново — свежий ордер, filled = 0.
        for o in [o for o in sm.orders.values() if o.owner == p.address]:
            sm.orders.pop(o.id, None)
        p.wrt_balance += 1.0  # но выручка шла
        _mk_order(sm, p.address, OrderType.SELL, prev, 1000.0)
        sm.current_height += auto_market.REVIEW_INTERVAL_BLOCKS
        nxt = auto_market.provider_ask_price(sm, p, bootstrap=0.02)
        assert nxt > prev
        prev = nxt


def test_step_once_posts_both_sides(engine, mk_account):
    """Поставщик выставляет запас, валидатор с пустым буфером — докуп."""
    sm = engine.state
    mk_account("p_two", role=Role.PROVIDER, ant=10_000.0, zkp=True)
    sm.accounts[GENESIS_VALIDATOR_ADDR].ant_balance = 0.0
    sm.accounts[GENESIS_VALIDATOR_ADDR].wrt_balance = 100_000.0

    n = auto_market.step_once(sm, engine)
    assert n >= 2
    sides = {tx.order_type.value for tx in sm.mempool if tx.tx_type.value == "create_order"}
    assert sides == {"buy", "sell"}


def test_validator_with_full_buffer_does_not_buy(engine, mk_account):
    sm = engine.state
    mk_account("p_full", role=Role.PROVIDER, ant=10_000.0, zkp=True)
    gv = sm.accounts[GENESIS_VALIDATOR_ADDR]
    l_total = engine._network_lzn_total_validators()
    gv.ant_balance = auto_market.BURN_CAP_LAMBDA * l_total * auto_market.ANT_BUFFER_BLOCKS
    gv.wrt_balance = 100_000.0

    auto_market.step_once(sm, engine)
    buys = [
        tx for tx in sm.mempool
        if tx.tx_type.value == "create_order" and tx.order_type.value == "buy"
    ]
    assert buys == []


def test_provider_requotes_when_price_moves(engine, mk_account):
    """Устаревший ask снимается, иначе он навсегда висит на вершине книги."""
    sm = engine.state
    p = mk_account("p_req", role=Role.PROVIDER, ant=10_000.0, zkp=True)
    target = auto_market.provider_ask_price(sm, p, bootstrap=0.02)
    stale_id = _mk_order(sm, p.address, OrderType.SELL, target * 5.0, 100.0)

    auto_market.step_once(sm, engine)
    cancels = [
        tx for tx in sm.mempool
        if tx.tx_type.value == "cancel_order" and tx.order_id == stale_id
    ]
    assert cancels
