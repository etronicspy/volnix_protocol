"""§5.4 стенд: declare/burn — replace-by-sender в мемпуле; last-wins в блоке."""
from __future__ import annotations

import time
import uuid

import pytest

from core import auto_declare
from core.models import Role, Transaction, TransactionType
from core.state import GENESIS_VALIDATOR_ADDR, StateManager


def _mk_declare(sender: str, b: float, s: float = 0.0) -> Transaction:
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=sender,
        amount=float(b),
        stake_amount=float(s),
        asset_type="ant",
        timestamp=time.time(),
    )


def test_mempool_admit_replaces_declare_same_sender(state_manager: StateManager):
    a = _mk_declare(GENESIS_VALIDATOR_ADDR, 10.0)
    b = _mk_declare(GENESIS_VALIDATOR_ADDR, 20.0)
    assert state_manager.mempool_admit(a) is False
    assert state_manager.mempool_admit(b) is True
    decls = [
        t
        for t in state_manager.mempool
        if t.tx_type == TransactionType.DECLARE_PARTICIPATION
    ]
    assert len(decls) == 1
    assert decls[0].amount == 20.0
    assert decls[0].tx_hash == b.tx_hash


def test_compact_declare_mempool_keeps_latest_per_sender(state_manager: StateManager):
    state_manager.mempool = [
        _mk_declare(GENESIS_VALIDATOR_ADDR, 1.0),
        _mk_declare(GENESIS_VALIDATOR_ADDR, 2.0),
        _mk_declare(GENESIS_VALIDATOR_ADDR, 3.0),
    ]
    removed = state_manager.compact_declare_mempool()
    assert removed == 2
    assert len(state_manager.mempool) == 1
    assert state_manager.mempool[0].amount == 3.0


def test_finalize_duplicate_sender_last_wins_silent(engine, mk_account):
    """Дубли в пакете: последний побеждает, без declare_dropped spam."""
    v = mk_account("v_dup", role=Role.VALIDATOR, frozen=100.0, ant=80.0, zkp=True)
    tx1 = _mk_declare(v.address, 5.0)
    tx2 = _mk_declare(v.address, 15.0)
    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, n, _L = engine._finalize_declares_batch(
        [tx1, tx2], txs_in_block, participation, requeue
    )
    assert n == 1
    assert B == 15.0
    assert v.ant_balance == pytest.approx(65.0)
    assert not any(t.get("tx_type") == "declare_dropped" for t in txs_in_block)
    assert any(t.get("tx_hash") == tx2.tx_hash for t in txs_in_block)


@pytest.mark.asyncio
async def test_produce_block_compacts_bloated_declare_mempool(engine):
    """Даже при прямом append дублей compact на старте блока оставляет один."""
    gv = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    gv.lzn_frozen_mining = 100.0
    gv.ant_balance = 100.0
    for i in range(50):
        engine.state.mempool.append(_mk_declare(gv.address, 1.0 + i * 0.01))
    assert len(engine.state.mempool) == 50
    await engine.produce_block()
    # После успеха мемпул = только requeue; дубли не должны вернуться пачкой
    decls = [
        t
        for t in engine.state.mempool
        if t.tx_type == TransactionType.DECLARE_PARTICIPATION
        and t.sender == gv.address
    ]
    assert len(decls) <= 1


def test_auto_declare_then_manual_replace(state_manager: StateManager, engine):
    auto_declare.step_once(state_manager, engine)
    replacement = _mk_declare(GENESIS_VALIDATOR_ADDR, 1.0)
    state_manager.submit_tx(replacement)
    decls = [
        t
        for t in state_manager.mempool
        if t.tx_type == TransactionType.DECLARE_PARTICIPATION
        and t.sender == GENESIS_VALIDATOR_ADDR
    ]
    assert len(decls) == 1
    assert decls[0].tx_hash == replacement.tx_hash
