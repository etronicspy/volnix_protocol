"""§5.5 (ruleset v2): _epoch_boundary — cancel SELL → wipe → emission → coeff ∈ [0.75, 1.5]."""
from __future__ import annotations

import time
import uuid

import pytest

from core.engine import COEFF_MAX, COEFF_MIN
from core.models import Order, OrderType, Role
from core.state import BLOCKS_PER_EPOCH
from tests.conftest import strip_bootstrap_economy


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
    # Без активированного LZN спроса на «электричество» нет: и потолок §5.5,
    # и нижняя граница эмиссии равны нулю — после wipe балансы остаются пустыми.
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.0)
    p1 = mk_account("p1", role=Role.PROVIDER, ant=500.0, zkp=True)
    p2 = mk_account("p2", role=Role.PROVIDER, ant=300.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 0.0
    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    assert p1.ant_balance == 0.0
    assert p2.ant_balance == 0.0
    types = [t["tx_type"] for t in txs_in_block]
    assert "epoch_ant_wipe" in types
    assert "epoch_emission" in types


def test_epoch_boundary_emission_eq_sold_times_coeff(engine, mk_account):
    """Шаг 2: emission = sold × coeff; делится поровну между Поставщиками."""
    # L_total=0.02 → спрос эпохи 67.2, потолок 201.6: рыночная эмиссия 100 попадает
    # внутрь коридора и не подменяется ни нижней границей, ни потолком §5.5.
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.02)
    mk_account("p1", role=Role.PROVIDER, ant=0.0, zkp=True)
    providers_count = sum(1 for a in engine.state.accounts.values() if a.role == Role.PROVIDER)
    assert providers_count == 1

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
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.02)
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
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.02)
    mk_account("p1", role=Role.PROVIDER, ant=10.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 42.0
    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)
    assert engine.state.epoch_ant_sold_volume == 0.0
    assert engine.state.epoch_ant_sold_last == 42.0
    assert engine.state.current_epoch_burn == 0.0


def test_epoch_wipe_includes_sell_escrow_and_keeps_buy_orders(engine, mk_account):
    """v2 step 1–2: эскроу SELL возвращается и сгорает в wipe; BUY-ордера переживают границу."""
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.0)
    p = mk_account("p_esc", role=Role.PROVIDER, ant=100.0, zkp=True)
    v = mk_account("v_esc", role=Role.VALIDATOR, frozen=10.0, wrt=50.0, zkp=True)
    v.lzn_frozen_mining = 0.0

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
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.02)
    mk_account("p1", role=Role.PROVIDER, ant=0.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 500.0
    engine.state.epoch_ant_sold_last = 0.0
    engine.state.epoch_emission_coefficient = 1.25

    engine._epoch_boundary(BLOCKS_PER_EPOCH, [])
    assert engine.state.epoch_emission_coefficient == 1.25


def test_epoch_coeff_ema_smoothing(engine, mk_account):
    """v2: coeff движется к coeff/ratio с весом EMA (α=0.5), а не скачком."""
    from core.engine import EMISSION_COEFF_ALPHA

    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.02)
    mk_account("p1", role=Role.PROVIDER, ant=0.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 80.0
    engine.state.epoch_ant_sold_last = 100.0  # ratio=0.8 → target = 1/0.8 = 1.25
    engine.state.epoch_emission_coefficient = 1.0

    engine._epoch_boundary(BLOCKS_PER_EPOCH, [])
    expected = EMISSION_COEFF_ALPHA * 1.25 + (1 - EMISSION_COEFF_ALPHA) * 1.0
    assert abs(engine.state.epoch_emission_coefficient - expected) < 1e-9


def test_epoch_no_providers_no_credit(engine):
    """Если Поставщиков нет — emission не начисляется."""
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.02)
    engine.state.epoch_ant_sold_volume = 100.0
    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)
    # Должна быть запись epoch_emission, но per=0
    summary = [t for t in txs_in_block if t["tx_type"] == "epoch_emission"]
    assert summary


def test_epoch_emission_capped_by_ant_max_epoch(engine, mk_account):
    """§5.5: эмиссия не превышает ANT_max_epoch = EpochBlocks × L_total."""
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.001)
    p = mk_account("p_cap", role=Role.PROVIDER, ant=0.0, zkp=True)

    ant_max_epoch = BLOCKS_PER_EPOCH * 0.001  # 10.08
    engine.state.epoch_ant_sold_volume = 10_000.0
    engine.state.epoch_emission_coefficient = 1.0

    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    summary = next(t for t in txs_in_block if t["tx_type"] == "epoch_emission")
    assert summary["amount"] == pytest.approx(ant_max_epoch)
    assert p.ant_balance == pytest.approx(ant_max_epoch)


def test_epoch_emission_not_capped_when_below_ceiling(engine, mk_account):
    """Внутри коридора §5.5 эмиссия проходит целиком (sold × coeff)."""
    strip_bootstrap_economy(engine.state, keep_seed_lzn=0.02)
    p = mk_account("p_free", role=Role.PROVIDER, ant=0.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 200.0
    engine.state.epoch_emission_coefficient = 1.0

    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    summary = next(t for t in txs_in_block if t["tx_type"] == "epoch_emission")
    assert summary["amount"] == pytest.approx(200.0)
    assert p.ant_balance == pytest.approx(200.0)


def test_epoch_emission_floored_by_epoch_burn_demand(engine, mk_account):
    """v2: при мёртвом рынке эмиссия не падает в ноль, а покрывает спрос эпохи.

    Иначе майнить нечем, сжигание падает до нуля и sold уже не восстановится
    (CANON_PROBLEMS §5): нижняя граница = EpochBlocks × λ × L_total.
    """
    from core.engine import BURN_CAP_LAMBDA

    strip_bootstrap_economy(engine.state, keep_seed_lzn=3.0)
    p = mk_account("p_floor", role=Role.PROVIDER, ant=0.0, zkp=True)
    engine.state.epoch_ant_sold_volume = 0.0  # рынок стоит

    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    expected = BLOCKS_PER_EPOCH * 3.0 * BURN_CAP_LAMBDA
    summary = next(
        t for t in txs_in_block
        if t["tx_type"] == "epoch_emission" and t.get("asset_type") == "ant"
    )
    assert summary["amount"] == pytest.approx(expected)
    assert p.ant_balance == pytest.approx(expected)


def test_epoch_lzn_credit_from_sold(engine, mk_account):
    """§5.5 5.0-sim: LZN_emit = sold_lzn × coeff, делится Поставщикам."""
    strip_bootstrap_economy(engine.state, keep_seed_lzn=1.0)
    p = mk_account("p_lzn_ep", role=Role.PROVIDER, ant=0.0, lzn=0.0, zkp=True)
    engine.state.epoch_lzn_sold_volume = 80.0
    engine.state.epoch_lzn_emission_coefficient = 1.0
    engine.state.epoch_lzn_sold_last = 0.0

    txs_in_block: list = []
    engine._epoch_boundary(BLOCKS_PER_EPOCH, txs_in_block)

    credits = [t for t in txs_in_block if t["tx_type"] == "epoch_lzn_credit"]
    assert credits and credits[0]["amount"] == pytest.approx(80.0)
    assert p.lzn_balance == pytest.approx(80.0)
    assert engine.state.epoch_lzn_sold_volume == 0.0
    assert engine.state.epoch_lzn_sold_last == 80.0
