"""§5.5 (ruleset v2): _epoch_boundary — cancel SELL → wipe → emission → coeff ∈ [0.75, 1.5]."""
from __future__ import annotations

import time
import uuid

from core.engine import COEFF_MAX, COEFF_MIN
from core.models import Order, OrderType, Role
from core.state import BLOCKS_PER_EPOCH


def test_epoch_boundary_not_called_off_boundary(engine, mk_account):
    """height не кратен BLOCKS_PER_EPOCH → ничего не происходит."""
    p = mk_account("p1", role=Role.PROVIDER, ant=500.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 100.0
    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH - 1, txs_in_block)
    assert p.ant_balance == 500.0
    assert engine.state.epoch_ant_sold_volume == 100.0
    assert txs_in_block == []


def test_epoch_boundary_wipes_provider_ant(engine, mk_account):
    """Шаг 1: на границе эпохи у Поставщиков ant_balance → 0."""
    p1 = mk_account("p1", role=Role.PROVIDER, ant=500.0, zkp=True)
    p2 = mk_account("p2", role=Role.PROVIDER, ant=300.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 0.0
    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    # Учитываем что genesis-provider тоже Provider; emission делится на всех
    # При sold=0 emission=0 → балансы после wipe+credit должны быть 0
    assert p1.ant_balance == 0.0
    assert p2.ant_balance == 0.0
    types = [t["tx_type"] for t in txs_in_block]
    assert "epoch_ant_wipe" in types
    assert "epoch_emission" in types


def test_epoch_boundary_emission_eq_sold_times_coeff(engine, mk_account):
    """Шаг 2: emission = sold × coeff; делится поровну между Поставщиками."""
    mk_account("p1", role=Role.PROVIDER, ant=0.0, zkp=True)
    # genesis-provider тоже Provider, поэтому смотрим counts
    providers_count = sum(1 for a in engine.state.accounts.values() if a.role == Role.PROVIDER)
    assert providers_count >= 2

    engine.state.epoch_ant_sold_volume = 100.0
    engine.state.epoch_emission_coefficient = 1.0
    engine.state.epoch_ant_sold_last = 0.0  # первая эпоха → coeff не меняется

    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    expected_per = 100.0 / providers_count
    for p in engine.state.accounts.values():
        if p.role == Role.PROVIDER:
            assert abs(p.ant_balance - expected_per) < 1e-9


def test_epoch_coeff_clamped_to_range(engine, mk_account):
    """coeff после обновления должен попасть в [COEFF_MIN, COEFF_MAX]."""
    mk_account("p1", role=Role.PROVIDER, ant=0.0, zkp=True)

    # sold резко упал → ratio < 1 → coeff растёт; проверим cap = COEFF_MAX
    engine.state.epoch_ant_sold_volume = 10.0
    engine.state.epoch_ant_sold_last = 1000.0
    engine.state.epoch_emission_coefficient = 1.0

    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)
    assert COEFF_MIN <= engine.state.epoch_emission_coefficient <= COEFF_MAX

    # Резкий рост sold → ratio > 1 → coeff падает; проверим cap = COEFF_MIN
    engine.state.epoch_ant_sold_volume = 1000.0
    engine.state.epoch_ant_sold_last = 10.0
    engine.state.epoch_emission_coefficient = 1.0
    engine._epoch_boundary(BLOCKS_PER_EPOCH * 2, [])
    assert COEFF_MIN <= engine.state.epoch_emission_coefficient <= COEFF_MAX


def test_epoch_resets_sold_counters(engine, mk_account):
    mk_account("p1", role=Role.PROVIDER, ant=10.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 42.0
    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)
    assert engine.state.epoch_ant_sold_volume == 0.0
    assert engine.state.epoch_ant_sold_last == 42.0
    assert engine.state.current_epoch_burn == 0.0


def test_epoch_wipe_includes_sell_escrow_and_keeps_buy_orders(engine, mk_account):
    """v2 step 1–2: эскроу SELL возвращается и сгорает в wipe; BUY-ордера переживают границу."""
    p = mk_account("p_esc", role=Role.PROVIDER, ant=100.0, zkp=True)
    v = mk_account("v_esc", role=Role.VALIDATOR, frozen=10.0, wrt=50.0, zkp=True)

    sell_id = uuid.uuid4().hex
    engine.state.orders[sell_id] = Order(
        id=sell_id, owner=p.address, order_type=OrderType.SELL,
        price=2.0, amount=40.0, filled=0.0, timestamp=time.time(),
    )
    p.ant_balance = 60.0  # 40 в эскроу

    buy_id = uuid.uuid4().hex
    engine.state.orders[buy_id] = Order(
        id=buy_id, owner=v.address, order_type=OrderType.BUY,
        price=1.0, amount=5.0, filled=0.0, timestamp=time.time(),
    )

    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    # SELL отменён, весь ANT поставщика (60 свободных + 40 эскроу) сожжён
    assert sell_id not in engine.state.orders
    assert p.ant_balance == 0.0
    cancels = [t for t in txs_in_block if t["tx_type"] == "epoch_order_cancel"]
    assert cancels and cancels[0]["amount"] == 40.0
    wipes = [t for t in txs_in_block if t["tx_type"] == "epoch_ant_wipe" and t["receiver"] == p.address]
    assert wipes and wipes[0]["amount"] == 100.0
    # BUY-ордер валидатора не тронут
    assert buy_id in engine.state.orders


def test_epoch_coeff_unchanged_when_sold_prev_zero(engine, mk_account):
    """v2: sold_prev = 0 → ratio := 1, коэффициент не меняется (нет деления на ноль)."""
    mk_account("p1", role=Role.PROVIDER, ant=0.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 500.0
    engine.state.epoch_ant_sold_last = 0.0
    engine.state.epoch_emission_coefficient = 1.25

    engine._epoch_boundary(BLOCKS_PER_EPOCH, [])
    assert engine.state.epoch_emission_coefficient == 1.25


def test_epoch_coeff_ema_smoothing(engine, mk_account):
    """v2: coeff движется к coeff/ratio с весом EMA (α=0.5), а не скачком."""
    from core.engine import EMISSION_COEFF_ALPHA

    mk_account("p1", role=Role.PROVIDER, ant=0.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 80.0
    engine.state.epoch_ant_sold_last = 100.0  # ratio=0.8 → target = 1/0.8 = 1.25
    engine.state.epoch_emission_coefficient = 1.0

    engine._epoch_boundary(BLOCKS_PER_EPOCH, [])
    expected = EMISSION_COEFF_ALPHA * 1.25 + (1 - EMISSION_COEFF_ALPHA) * 1.0
    assert abs(engine.state.epoch_emission_coefficient - expected) < 1e-9


def test_epoch_no_providers_no_credit(engine):
    """Если Поставщиков нет — emission не начисляется (delete genesis-provider)."""
    from core.state import GENESIS_PROVIDER_ADDR

    del engine.state.accounts[GENESIS_PROVIDER_ADDR]
    engine.state.epoch_ant_sold_volume = 100.0
    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)
    # Должна быть запись epoch_emission, но per=0
    summary = [t for t in txs_in_block if t["tx_type"] == "epoch_emission"]
    assert summary
