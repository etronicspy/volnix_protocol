"""AutoDeclareDaemon: один тик подаёт canon-корректный declare, блок принимается."""
from __future__ import annotations

import pytest

from core import auto_declare
from core.engine import BURN_CAP_LAMBDA, SimulationEngine
from core.models import Role, TransactionType
from core.state import GENESIS_VALIDATOR_ADDR, StateManager


def test_step_once_submits_declare_for_genesis_validator(state_manager: StateManager, engine: SimulationEngine):
    assert len(state_manager.mempool) == 0
    n = auto_declare.step_once(state_manager, engine)
    assert n == 1
    # ровно один declare в мемпуле, на genesis-валидатора с b = λ·L
    decls = [tx for tx in state_manager.mempool if tx.tx_type == TransactionType.DECLARE_PARTICIPATION]
    assert len(decls) == 1
    gv = state_manager.accounts[GENESIS_VALIDATOR_ADDR]
    expected_b = BURN_CAP_LAMBDA * gv.lzn_frozen_mining
    assert decls[0].sender == GENESIS_VALIDATOR_ADDR
    assert decls[0].amount == pytest.approx(expected_b, rel=1e-9, abs=1e-9)


def test_step_once_skips_duplicates(state_manager: StateManager, engine: SimulationEngine):
    auto_declare.step_once(state_manager, engine)
    # повторный тик не должен дублировать declare
    n2 = auto_declare.step_once(state_manager, engine)
    assert n2 == 0


@pytest.mark.asyncio
async def test_auto_declare_enables_block_reward(state_manager: StateManager, engine: SimulationEngine):
    """§5.4: без declare блока нет; с AutoDeclareDaemon — блок и награда §5.1."""
    h0 = state_manager.current_height
    # без declare сжигать нечего → высота не растёт
    with pytest.raises(RuntimeError, match="min-burn"):
        await engine.produce_block()
    assert state_manager.current_height == h0

    auto_declare.step_once(state_manager, engine)
    await engine.produce_block()
    assert state_manager.current_height == h0 + 1
    txs = state_manager.blocks[-1]["transactions"]
    assert any(t.get("tx_type") == "block_reward" for t in txs)


def test_step_once_independent_of_validator_set(
    state_manager: StateManager, engine: SimulationEngine
):
    """§5.4: declare подаётся по активированному LZN, а не по текущему ValidatorSet."""
    state_manager.consensus_validator_set = []
    assert auto_declare.step_once(state_manager, engine) == 1


def test_step_once_covers_validator_outside_validator_set(
    state_manager: StateManager, engine: SimulationEngine
):
    """Валидатор вне ValidatorSet тоже жжёт ANT — иначе набор не пополняется."""
    acc = state_manager.create_account("fresh_val")
    acc.role = Role.VALIDATOR
    acc.zkp_verified = True
    acc.lzn_frozen_mining = 30.0
    acc.ant_balance = 500.0

    n = auto_declare.step_once(state_manager, engine)
    senders = {
        tx.sender
        for tx in state_manager.mempool
        if tx.tx_type == TransactionType.DECLARE_PARTICIPATION
    }
    assert n == 2
    assert senders == {GENESIS_VALIDATOR_ADDR, "fresh_val"}


def test_step_once_burns_remaining_ant_when_below_lambda(
    state_manager: StateManager, engine: SimulationEngine
):
    """Мало ANT — объявляем остаток: b_i=0 означало бы нулевой доход (§5.1)."""
    gv = state_manager.accounts[GENESIS_VALIDATOR_ADDR]
    gv.ant_balance = 0.1  # < λ·L_i = 1/3
    assert auto_declare.step_once(state_manager, engine) == 1
    decl = next(
        tx
        for tx in state_manager.mempool
        if tx.tx_type == TransactionType.DECLARE_PARTICIPATION
    )
    assert decl.amount == pytest.approx(0.1)


def test_step_once_skips_when_insufficient_ant(state_manager: StateManager, engine: SimulationEngine):
    gv = state_manager.accounts[GENESIS_VALIDATOR_ADDR]
    gv.ant_balance = 0.0
    assert auto_declare.step_once(state_manager, engine) == 0


def test_step_once_validator_with_zero_lzn_skipped(state_manager: StateManager, engine: SimulationEngine):
    addr = "second_val"
    acc = state_manager.create_account(addr)
    acc.role = Role.VALIDATOR
    acc.lzn_frozen_mining = 0.0
    acc.ant_balance = 1000.0
    state_manager.consensus_validator_set = [
        *state_manager.consensus_validator_set,
        {"address": addr, "power": 1e-9},
    ]
    n = auto_declare.step_once(state_manager, engine)
    # только genesis-валидатор → 1 declare
    assert n == 1
