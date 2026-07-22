"""Интеграция: produce_block — accept / rollback / persistence."""
from __future__ import annotations

import time
import uuid

import pytest

from core.engine import BURN_CAP_LAMBDA
from core.models import Role, Transaction, TransactionType
from core.state import GENESIS_PROVIDER_ADDR, GENESIS_VALIDATOR_ADDR, SIM_TREASURY_ADDR


def _mk_transfer(sender, receiver, amount, asset="wrt"):
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.TRANSFER,
        sender=sender,
        receiver=receiver,
        amount=float(amount),
        asset_type=asset,
        timestamp=time.time(),
    )


def _mk_mint(receiver, amount, asset):
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.MINT,
        sender=SIM_TREASURY_ADDR,
        receiver=receiver,
        amount=float(amount),
        asset_type=asset,
        timestamp=time.time(),
    )


def _mk_declare(sender, b, s):
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=sender,
        amount=float(b),
        stake_amount=float(s),
        asset_type="ant",
        timestamp=time.time(),
    )


@pytest.mark.asyncio
async def test_produce_block_advances_height_on_accepted_burn(engine, mk_account):
    """Σb_i ≈ λ·L_total → блок принимается, высота +1."""
    # genesis-валидатор уже есть с L=6667, casting declare с b ≈ λ·L
    gv = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    L = gv.lzn_frozen_mining
    b = BURN_CAP_LAMBDA * L
    tx = _mk_declare(gv.address, b=b, s=0.0)
    engine.state.mempool.append(tx)

    h0 = engine.state.current_height
    await engine.produce_block()
    assert engine.state.current_height == h0 + 1


@pytest.mark.asyncio
async def test_produce_block_accepts_small_burn_below_cap(engine):
    """v2: любой Σb_i ≤ λ·L_total валиден — маленький burn принимается, блок растёт."""
    gv0 = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    initial_ant = gv0.ant_balance
    h0 = engine.state.current_height

    tx = _mk_declare(gv0.address, b=1.0, s=0.0)
    engine.state.mempool.append(tx)

    await engine.produce_block()

    assert engine.state.current_height == h0 + 1
    gv_after = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    # Сожжён ровно b_i = 1.0 (v2: s_i не сжигается, здесь s=0)
    assert gv_after.ant_balance == pytest.approx(initial_ant - 1.0)
    # Базовая награда начислена (b_i > 0)
    last_block = engine.state.blocks[-1]
    assert any(t.get("tx_type") == "block_reward" for t in last_block["transactions"])


@pytest.mark.asyncio
async def test_produce_block_without_declare_commits_without_reward(engine):
    """v2: без declare (Σb_i = 0) блок валиден, но базовая WRT не эмитируется."""
    h0 = engine.state.current_height
    # пустой мемпул
    await engine.produce_block()
    assert engine.state.current_height == h0 + 1
    last_block = engine.state.blocks[-1]
    txs = last_block["transactions"]
    assert not any(t.get("tx_type") == "block_reward" for t in txs)
    assert any(t.get("tx_type") == "block_reward_skipped" for t in txs)


@pytest.mark.asyncio
async def test_produce_block_stake_not_burned(engine):
    """v2: при declare с s_i > 0 списывается только b_i; ставка остаётся на балансе."""
    gv0 = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    initial_ant = gv0.ant_balance
    b, s = 100.0, 500.0
    engine.state.mempool.append(_mk_declare(gv0.address, b=b, s=s))

    await engine.produce_block()

    gv_after = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    assert gv_after.ant_balance == pytest.approx(initial_ant - b)


def _declare_for_gv(engine):
    """Соберём declare от genesis-валидатора с b ≈ λ·L_total по текущему состоянию."""
    gv = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    L_total = engine._network_lzn_total_validators()
    target = BURN_CAP_LAMBDA * L_total
    return _mk_declare(gv.address, b=target, s=0.0)


@pytest.mark.asyncio
async def test_produce_block_transfer_wrt_via_admission(engine, mk_account):
    """TRANSFER WRT исполняется в блоке."""
    sender = mk_account("alice", role=Role.CITIZEN, wrt=10.0)
    receiver = mk_account("bob", role=Role.CITIZEN, wrt=0.0)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk_transfer(sender.address, receiver.address, 3.5))

    await engine.produce_block()

    s = engine.state.accounts[sender.address]
    r = engine.state.accounts[receiver.address]
    assert s.wrt_balance == 10.0 - 3.5
    assert r.wrt_balance == 3.5


@pytest.mark.asyncio
async def test_produce_block_reject_transfer_ant_logged(engine, mk_account):
    """Transfer ANT по §4.1 запрещён — в блоке должна быть запись deliver_tx_reject."""
    v = mk_account("v_ant", role=Role.VALIDATOR, frozen=10.0, ant=50.0, zkp=True)
    p = mk_account("p_ant", role=Role.PROVIDER, ant=0.0, zkp=True)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk_transfer(v.address, p.address, 5.0, asset="ant"))

    await engine.produce_block()

    last_block = engine.state.blocks[-1]
    rejects = [t for t in last_block["transactions"] if t.get("tx_type") == "deliver_tx_reject"]
    assert rejects, "no deliver_tx_reject entries found in last block"
    assert any(
        "ANT" in (t.get("title") or "") or "asset=ant" in (t.get("details") or "")
        for t in rejects
    )
    v_after = engine.state.accounts[v.address]
    p_after = engine.state.accounts[p.address]
    assert v_after.ant_balance == 50.0
    assert p_after.ant_balance == 0.0


@pytest.mark.asyncio
async def test_produce_block_mint_wrt_via_engine(engine, mk_account):
    target = mk_account("target", role=Role.CITIZEN, wrt=0.0)
    engine.state.mempool.append(_declare_for_gv(engine))
    engine.state.mempool.append(_mk_mint(target.address, 25.0, "wrt"))

    await engine.produce_block()
    t_after = engine.state.accounts[target.address]
    assert t_after.wrt_balance == 25.0


@pytest.mark.asyncio
async def test_set_block_time_clamps_range(engine):
    engine.set_block_time(0.001)
    assert engine.block_time == pytest.approx(0.1)  # clamped low
    engine.set_block_time(99999.0)
    assert engine.block_time == pytest.approx(300.0)  # clamped high
    engine.set_block_time(5.0)
    assert engine.block_time == pytest.approx(5.0)


@pytest.mark.asyncio
async def test_set_speed_maps_to_block_time(engine):
    """Скорость симуляции: 1:1 → блок/60 с; 604800× (1 с = 1 нед) → блок/9.92e-5 с."""
    engine.set_speed(1.0)
    assert engine.block_time == pytest.approx(60.0)
    assert engine.state.sim_speed == pytest.approx(1.0)
    assert engine.state.sim_block_interval_sec == pytest.approx(60.0)

    engine.set_speed(604_800.0)
    assert engine.block_time == pytest.approx(60.0 / 604_800.0)

    engine.set_speed(10.0 ** 9)  # clamp high → 1 с = 1 нед
    assert engine.speed == pytest.approx(604_800.0)

    engine.set_speed(0.0)  # clamp low → 0.2× (блок раз в 300 с)
    assert engine.speed == pytest.approx(0.2)
    assert engine.block_time == pytest.approx(300.0)


@pytest.mark.asyncio
async def test_batch_tick_produces_multiple_blocks(engine):
    """Batch-режим высоких скоростей: за один тик производится пачка блоков."""
    engine.set_speed(604_800.0)
    engine.is_running = True
    h0 = engine.state.current_height
    await engine._produce_batch_tick()
    engine.is_running = False
    assert engine.state.current_height > h0


@pytest.mark.asyncio
async def test_batch_tick_calls_pre_block_hook(engine):
    """В batch-режиме движок зовёт pre_block_hook перед каждым блоком (auto-declare)."""
    calls = []
    engine.pre_block_hook = lambda: calls.append(1)
    engine.set_speed(604_800.0)
    engine.is_running = True
    h0 = engine.state.current_height
    await engine._produce_batch_tick()
    engine.is_running = False
    assert len(calls) == engine.state.current_height - h0
