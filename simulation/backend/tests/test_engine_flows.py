"""DeliverTx-сценарии в produce_block: set_role, activate_lzn, ZKP, ордеры, матч, отмена."""
from __future__ import annotations

import time
import uuid

import pytest

from core.engine import BURN_CAP_LAMBDA
from core.models import Order, OrderType, Role, Transaction, TransactionType
from core.state import GENESIS_VALIDATOR_ADDR


def _declare_for_gv(engine):
    gv = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    L_total = engine._network_lzn_total_validators()
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=gv.address,
        amount=BURN_CAP_LAMBDA * L_total,
        stake_amount=0.0,
        asset_type="ant",
        timestamp=time.time(),
    )


def _mk(tx_type, **kw):
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=tx_type,
        timestamp=time.time(),
        **kw,
    )


@pytest.mark.asyncio
async def test_set_role_validator_requires_zkp_and_lzn(engine, mk_account):
    """set_role → VALIDATOR без ZKP — отклонение в DeliverTx."""
    a = mk_account("no_zkp", role=Role.CITIZEN, zkp=False)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.SET_ROLE, sender=a.address, receiver=a.address, role=Role.VALIDATOR))

    await engine.produce_block()
    a_after = engine.state.accounts[a.address]
    assert a_after.role == Role.CITIZEN  # не применилась


@pytest.mark.asyncio
async def test_set_role_provider_with_zkp_applies(engine, mk_account):
    a = mk_account("with_zkp", role=Role.CITIZEN, zkp=True)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.SET_ROLE, sender=a.address, receiver=a.address, role=Role.PROVIDER))
    await engine.produce_block()
    assert engine.state.accounts[a.address].role == Role.PROVIDER


@pytest.mark.asyncio
async def test_set_role_to_citizen_burns_ant_and_releases_orders(engine, mk_account):
    """Возврат к CITIZEN сжигает ANT и снимает ордера."""
    p = mk_account("ex_provider", role=Role.PROVIDER, ant=33.0, wrt=0.0, zkp=True)
    oid = uuid.uuid4().hex
    engine.state.orders[oid] = Order(
        id=oid,
        owner=p.address,
        order_type=OrderType.SELL,
        price=5.0,
        amount=4.0,
        filled=0.0,
        timestamp=time.time(),
    )
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.SET_ROLE, sender=p.address, receiver=p.address, role=Role.CITIZEN))
    await engine.produce_block()
    p_after = engine.state.accounts[p.address]
    assert p_after.role == Role.CITIZEN
    assert p_after.ant_balance == 0.0
    assert oid not in engine.state.orders


@pytest.mark.asyncio
async def test_zkp_verify_flag_flips(engine, mk_account):
    a = mk_account("u_zkp", role=Role.CITIZEN, zkp=False)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.ZKP_VERIFY, sender=a.address, receiver=a.address))
    await engine.produce_block()
    assert engine.state.accounts[a.address].zkp_verified is True


@pytest.mark.asyncio
async def test_activate_lzn_validator_freezes_balance(engine, mk_account):
    """activate_lzn у Валидатора → перевод из ликвидного LZN в frozen_mining.

    activate_lzn увеличивает L_total в том же блоке; считаем declare-target по
    пост-активации, чтобы Σb_i попало в коридор λ·L_total.
    """
    v = mk_account("v_act", role=Role.VALIDATOR, frozen=0.0, lzn=100.0, zkp=True)
    # Cимулируем post-activation L_total для расчёта declare
    L_post = engine._network_lzn_total_validators() + 20.0
    declare = Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=GENESIS_VALIDATOR_ADDR,
        amount=BURN_CAP_LAMBDA * L_post,
        stake_amount=0.0,
        asset_type="ant",
        timestamp=time.time(),
    )
    # activate_lzn должен пройти раньше declare-batch (mempool обрабатывается до declare)
    engine.state.mempool.append(_mk(TransactionType.ACTIVATE_LZN, sender=v.address, amount=20.0, asset_type="lzn"))
    engine.state.mempool.append(declare)

    await engine.produce_block()
    v_after = engine.state.accounts[v.address]
    assert v_after.lzn_balance == 80.0
    assert v_after.lzn_frozen_mining == 20.0


@pytest.mark.asyncio
async def test_activate_lzn_non_validator_rejected(engine, mk_account):
    c = mk_account("c_act", role=Role.CITIZEN, lzn=10.0)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.ACTIVATE_LZN, sender=c.address, amount=1.0, asset_type="lzn"))
    await engine.produce_block()
    c_after = engine.state.accounts[c.address]
    assert c_after.lzn_balance == 10.0
    assert c_after.lzn_frozen_mining == 0.0


@pytest.mark.asyncio
async def test_activate_lzn_above_cap_rejected(engine, mk_account):
    from core.state import LZN_MAX_FROZEN_PER_ADDRESS

    v = mk_account("v_cap", role=Role.VALIDATOR, frozen=float(LZN_MAX_FROZEN_PER_ADDRESS - 1), lzn=100.0, zkp=True)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.ACTIVATE_LZN, sender=v.address, amount=50.0, asset_type="lzn"))
    await engine.produce_block()
    v_after = engine.state.accounts[v.address]
    assert v_after.lzn_frozen_mining == float(LZN_MAX_FROZEN_PER_ADDRESS - 1)
    assert v_after.lzn_balance == 100.0


@pytest.mark.asyncio
async def test_limit_buy_order_escrows_wrt(engine, mk_account):
    v = mk_account("v_buy", role=Role.VALIDATOR, frozen=10.0, wrt=100.0, zkp=True)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(
        _mk(TransactionType.CREATE_ORDER, sender=v.address, order_type=OrderType.BUY, price=5.0, amount=4.0)
    )
    await engine.produce_block()
    v_after = engine.state.accounts[v.address]
    assert v_after.wrt_balance == 100.0 - 20.0
    assert any(o.owner == v.address for o in engine.state.orders.values())


@pytest.mark.asyncio
async def test_limit_sell_order_escrows_ant(engine, mk_account):
    p = mk_account("p_sell", role=Role.PROVIDER, ant=10.0, zkp=True)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(
        _mk(TransactionType.CREATE_ORDER, sender=p.address, order_type=OrderType.SELL, price=3.0, amount=4.0)
    )
    await engine.produce_block()
    p_after = engine.state.accounts[p.address]
    assert p_after.ant_balance == 6.0
    assert any(o.owner == p.address for o in engine.state.orders.values())


@pytest.mark.asyncio
async def test_match_engine_settles_crossing_orders(engine, mk_account):
    """Один блок: BUY @ 5 пересекается с SELL @ 4 → сделка по 4 (price of maker)."""
    v = mk_account("v_m", role=Role.VALIDATOR, frozen=10.0, wrt=100.0, zkp=True)
    p = mk_account("p_m", role=Role.PROVIDER, ant=20.0, zkp=True)
    engine.state.mempool.append(_declare_for_gv(engine))
    # Сначала SELL (maker), потом BUY (taker)
    engine.state.mempool.append(
        _mk(TransactionType.CREATE_ORDER, sender=p.address, order_type=OrderType.SELL, price=4.0, amount=5.0)
    )
    engine.state.mempool.append(
        _mk(TransactionType.CREATE_ORDER, sender=v.address, order_type=OrderType.BUY, price=5.0, amount=5.0)
    )
    await engine.produce_block()
    v_after = engine.state.accounts[v.address]
    p_after = engine.state.accounts[p.address]
    # Покупатель получил ANT, у продавца его меньше
    assert v_after.ant_balance == 5.0
    assert p_after.ant_balance == 15.0  # 20 - 5
    # Цена сделки: лимит maker = 4 → продавец получил 5*4=20 WRT
    assert p_after.wrt_balance == 20.0
    # Покупатель: эскроу был 25, осталось 5 (5 - 4 = 1 за единицу скидки × 5)
    assert v_after.wrt_balance == 100.0 - 25.0 + (5.0 - 4.0) * 5.0
    # Last price = 4
    assert engine.state.last_price == 4.0


@pytest.mark.asyncio
async def test_cancel_order_returns_escrow(engine, mk_account):
    v = mk_account("v_cancel", role=Role.VALIDATOR, frozen=10.0, wrt=100.0, zkp=True)
    oid = uuid.uuid4().hex
    engine.state.orders[oid] = Order(
        id=oid,
        owner=v.address,
        order_type=OrderType.BUY,
        price=5.0,
        amount=4.0,
        filled=0.0,
        timestamp=time.time(),
    )
    # Уменьшим WRT, как будто эскроу уже сняли
    v.wrt_balance = 80.0
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.CANCEL_ORDER, sender=v.address, order_id=oid))
    await engine.produce_block()
    v_after = engine.state.accounts[v.address]
    assert oid not in engine.state.orders
    assert v_after.wrt_balance == 80.0 + 20.0


@pytest.mark.asyncio
async def test_cancel_order_wrong_owner_rejected(engine, mk_account):
    v = mk_account("v_owner", role=Role.VALIDATOR, frozen=10.0, wrt=100.0, zkp=True)
    impostor = mk_account("imp", role=Role.CITIZEN, wrt=0.0)
    oid = uuid.uuid4().hex
    engine.state.orders[oid] = Order(
        id=oid,
        owner=v.address,
        order_type=OrderType.BUY,
        price=5.0,
        amount=4.0,
        filled=0.0,
        timestamp=time.time(),
    )
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk(TransactionType.CANCEL_ORDER, sender=impostor.address, order_id=oid))
    await engine.produce_block()
    assert oid in engine.state.orders


@pytest.mark.asyncio
async def test_create_order_buy_from_provider_rejected(engine, mk_account):
    """SELL — только Provider; BUY — только Validator (§5.2)."""
    p = mk_account("p_wrong", role=Role.PROVIDER, wrt=100.0, ant=10.0, zkp=True)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(
        _mk(TransactionType.CREATE_ORDER, sender=p.address, order_type=OrderType.BUY, price=5.0, amount=1.0)
    )
    await engine.produce_block()
    # Ордер от Provider в роли BUY не должен попасть в книгу
    assert not any(o.owner == p.address and o.order_type == OrderType.BUY for o in engine.state.orders.values())


@pytest.mark.asyncio
async def test_market_buy_consumes_best_ask(engine, mk_account):
    """MARKET BUY — Валидатор покупает по лучшим ask до lim WRT / amount ANT."""
    v = mk_account("v_mk", role=Role.VALIDATOR, frozen=10.0, wrt=100.0, zkp=True)
    p = mk_account("p_mk", role=Role.PROVIDER, ant=10.0, zkp=True)
    # Заранее ставим SELL ордер от p
    oid = uuid.uuid4().hex
    engine.state.orders[oid] = Order(
        id=oid,
        owner=p.address,
        order_type=OrderType.SELL,
        price=4.0,
        amount=5.0,
        filled=0.0,
        timestamp=time.time(),
    )
    p.ant_balance = 5.0  # 5 в эскроу

    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(
        _mk(
            TransactionType.CREATE_ORDER,
            sender=v.address,
            order_type=OrderType.BUY,
            price=0.0,
            amount=3.0,
            market=True,
            max_wrt=20.0,
        )
    )
    await engine.produce_block()
    v_after = engine.state.accounts[v.address]
    p_after = engine.state.accounts[p.address]
    assert v_after.ant_balance == 3.0
    assert v_after.wrt_balance == 100.0 - 12.0
    assert p_after.wrt_balance == 12.0
