"""§4.1–4.2: _sanitize_ant_ineligible_roles — ANT и ордера только у Provider/Validator."""
from __future__ import annotations

import time
import uuid

from core.models import Order, OrderType, Role


def test_sanitize_burns_ant_on_citizen(engine, mk_account):
    """ANT на Гражданине → сжигается в новой ленте блока с пометкой §4.1."""
    c = mk_account("citizen_with_ant", role=Role.CITIZEN, ant=42.0)
    txs: list = []
    engine._sanitize_ant_ineligible_roles(txs)
    assert c.ant_balance == 0.0
    burn = [t for t in txs if t["tx_type"] == "protocol_ant_burn"]
    assert burn and burn[0]["receiver"] == c.address
    assert burn[0]["amount"] == 42.0


def test_sanitize_skips_treasury(engine):
    """Казне симуляции ANT не трогать (служебный адрес)."""
    from core.state import SIM_TREASURY_ADDR

    tr = engine.state.accounts[SIM_TREASURY_ADDR]
    tr.ant_balance = 1_000_000.0
    txs: list = []
    engine._sanitize_ant_ineligible_roles(txs)
    assert tr.ant_balance == 1_000_000.0


def test_sanitize_removes_orders_from_citizen(engine, mk_account):
    """Если Гражданин — владелец ордера → ордер снимается, эскроу возвращается."""
    c = mk_account("c_with_order", role=Role.CITIZEN, wrt=50.0, ant=5.0)
    oid = uuid.uuid4().hex
    engine.state.orders[oid] = Order(
        id=oid,
        owner=c.address,
        order_type=OrderType.BUY,
        price=2.0,
        amount=10.0,
        filled=0.0,
        timestamp=time.time(),
    )
    txs: list = []
    engine._sanitize_ant_ineligible_roles(txs)
    assert oid not in engine.state.orders
    # эскроу: цена × остаток = 20 → +20 к WRT
    assert abs(c.wrt_balance - (50.0 + 20.0)) < 1e-9
    cancel = [t for t in txs if t["tx_type"] == "protocol_order_cancel"]
    assert cancel and cancel[0]["receiver"] == c.address


def test_sanitize_keeps_provider_and_validator_ant(engine, mk_account):
    p = mk_account("p_keeps", role=Role.PROVIDER, ant=100.0, zkp=True)
    v = mk_account("v_keeps", role=Role.VALIDATOR, frozen=10.0, ant=200.0, zkp=True)
    txs: list = []
    engine._sanitize_ant_ineligible_roles(txs)
    assert p.ant_balance == 100.0
    assert v.ant_balance == 200.0


def test_release_orders_for_owner_buy(engine, mk_account):
    v = mk_account("v_rel", role=Role.VALIDATOR, frozen=10.0, ant=10.0, wrt=0.0, zkp=True)
    oid = uuid.uuid4().hex
    engine.state.orders[oid] = Order(
        id=oid,
        owner=v.address,
        order_type=OrderType.BUY,
        price=5.0,
        amount=4.0,
        filled=1.0,
        timestamp=time.time(),
    )
    engine._release_orders_for_owner(v.address, v)
    assert oid not in engine.state.orders
    # (amount - filled) * price = 3 * 5 = 15
    assert v.wrt_balance == 15.0


def test_release_orders_for_owner_sell(engine, mk_account):
    p = mk_account("p_rel", role=Role.PROVIDER, ant=0.0, zkp=True)
    oid = uuid.uuid4().hex
    engine.state.orders[oid] = Order(
        id=oid,
        owner=p.address,
        order_type=OrderType.SELL,
        price=5.0,
        amount=8.0,
        filled=3.0,
        timestamp=time.time(),
    )
    engine._release_orders_for_owner(p.address, p)
    assert oid not in engine.state.orders
    assert p.ant_balance == 5.0  # (8-3)
