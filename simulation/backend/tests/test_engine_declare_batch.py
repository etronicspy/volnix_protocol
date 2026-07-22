"""§5.4: _finalize_declares_batch — λ-cap, K-cap, ordered_unique, requeue."""
from __future__ import annotations

import time
import uuid

from core.engine import BURN_CAP_LAMBDA, MAX_ACTIVE_VALIDATORS_K
from core.models import Role, Transaction, TransactionType


def _mk_declare(sender, b, s, ts=None):
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=sender,
        amount=float(b),
        stake_amount=float(s),
        asset_type="ant",
        timestamp=ts or time.time(),
    )


def test_finalize_declares_applies_within_cap(engine, mk_account):
    """Один валидатор с b ≤ λ·L_total → tx применяется, ANT списан."""
    L = 100.0
    v = mk_account("v_single", role=Role.VALIDATOR, frozen=L, ant=50.0, zkp=True)
    cap = BURN_CAP_LAMBDA * L
    b = cap / 2
    tx = _mk_declare(v.address, b=b, s=0.0)

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []

    B, n = engine._finalize_declares_batch([tx], txs_in_block, participation, requeue)

    assert n == 1
    assert B == b
    assert not requeue
    assert v.address in participation
    assert v.ant_balance == 50.0 - b
    # tx попала в ленту блока
    assert any(t["tx_hash"] == tx.tx_hash for t in txs_in_block)


def test_finalize_declares_drops_lowest_w_when_sum_exceeds_cap(engine, mk_account):
    """Σb > λ·L_total → выбрасываем наименьшие w_i = s/L_i, пока не уложимся."""
    # Удалим genesis-валидатор: в этом тесте только v1+v2 формируют L_total
    from core.state import GENESIS_VALIDATOR_ADDR

    del engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    L = 100.0
    v1 = mk_account("v_hi_w", role=Role.VALIDATOR, frozen=L, ant=100.0, zkp=True)
    v2 = mk_account("v_lo_w", role=Role.VALIDATOR, frozen=L, ant=100.0, zkp=True)
    cap = BURN_CAP_LAMBDA * (L + L)  # =66.67

    # v1: высокий стейк → высокий w_i
    tx1 = _mk_declare(v1.address, b=40.0, s=30.0)  # w=0.30
    # v2: низкий стейк → низкий w_i; Σb=80 > cap=66.67 → v2 (lower w_i) уходит
    tx2 = _mk_declare(v2.address, b=40.0, s=1.0)  # w=0.01

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, _n = engine._finalize_declares_batch(
        [tx1, tx2], txs_in_block, participation, requeue
    )

    # v2 (низкий w_i) должен попасть в requeue; v1 — применён
    assert v1.address in participation
    assert v2.address not in participation
    assert tx2 in requeue
    assert B <= cap + 0.5


def test_finalize_declares_burns_only_b_not_stake(engine, mk_account):
    """v2: s_i — возвращаемая ставка; с баланса списывается только b_i."""
    L = 100.0
    v = mk_account("v_stake", role=Role.VALIDATOR, frozen=L, ant=80.0, zkp=True)
    b, s = 10.0, 40.0  # b + s ≤ L и ≤ ant
    tx = _mk_declare(v.address, b=b, s=s)

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, n = engine._finalize_declares_batch([tx], txs_in_block, participation, requeue)

    assert n == 1
    assert B == b
    assert v.ant_balance == 80.0 - b  # ставка s не списана
    assert participation[v.address]["s"] == s
    assert participation[v.address]["w_i"] == s / L


def test_finalize_declares_stake_requires_available_ant(engine, mk_account):
    """v2: хотя s_i не сжигается, требование ANT ≥ b_i + s_i сохраняется."""
    v = mk_account("v_thin", role=Role.VALIDATOR, frozen=100.0, ant=20.0, zkp=True)
    tx = _mk_declare(v.address, b=5.0, s=30.0)  # b+s=35 > ant=20

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, n = engine._finalize_declares_batch([tx], txs_in_block, participation, requeue)

    assert n == 0
    assert tx in requeue
    assert v.ant_balance == 20.0


def test_finalize_declares_drops_non_validator(engine, mk_account):
    """Структурно невалидный declare (не валидатор) — drop, не вечный requeue."""
    a = mk_account("citizen_decl", role=Role.CITIZEN, ant=50.0)
    tx = _mk_declare(a.address, b=1.0, s=0.0)

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, n = engine._finalize_declares_batch([tx], txs_in_block, participation, requeue)

    assert n == 0
    assert B == 0.0
    assert tx not in requeue
    assert a.ant_balance == 50.0  # не списано
    assert any(
        t.get("tx_type") == "declare_dropped" and t.get("tx_hash") == tx.tx_hash
        for t in txs_in_block
    )


def test_finalize_declares_drops_zero_lzn(engine, mk_account):
    """Валидатор без активированного LZN — drop (иначе мемпул залипает навечно)."""
    v = mk_account("v_nolzn", role=Role.VALIDATOR, frozen=0.0, ant=50.0, zkp=True)
    tx = _mk_declare(v.address, b=1.0, s=0.0)

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, n = engine._finalize_declares_batch([tx], txs_in_block, participation, requeue)

    assert n == 0
    assert tx not in requeue
    assert any(t.get("tx_type") == "declare_dropped" for t in txs_in_block)


def test_finalize_declares_rejects_insufficient_ant(engine, mk_account):
    """Мало ANT — временно: requeue (валидатор может купить ANT)."""
    v = mk_account("v_poor", role=Role.VALIDATOR, frozen=100.0, ant=0.5, zkp=True)
    tx = _mk_declare(v.address, b=5.0, s=0.0)

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, n = engine._finalize_declares_batch([tx], txs_in_block, participation, requeue)

    assert n == 0
    assert tx in requeue


def test_finalize_declares_unique_sender_keeps_first(engine, mk_account):
    """Дубликат от одного sender: первый может примениться, второй — drop."""
    L = 100.0
    v = mk_account("v_dup", role=Role.VALIDATOR, frozen=L, ant=50.0, zkp=True)
    cap = BURN_CAP_LAMBDA * L
    tx1 = _mk_declare(v.address, b=cap / 4, s=0.0)
    tx2 = _mk_declare(v.address, b=cap / 4, s=0.0)

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    engine._finalize_declares_batch([tx1, tx2], txs_in_block, participation, requeue)

    applied = [
        t
        for t in txs_in_block
        if t.get("tx_type") == "declare_participation"
        and t.get("tx_hash") in {tx1.tx_hash, tx2.tx_hash}
    ]
    assert len(applied) == 1
    assert any(
        t.get("tx_type") == "declare_dropped" and t.get("tx_hash") == tx2.tx_hash
        for t in txs_in_block
    )
    assert tx2 not in requeue


def test_finalize_declares_empty_validator_set_returns_zero(engine):
    """Если L_total = 0 — все declare в requeue, B=0."""
    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    B, n = engine._finalize_declares_batch([], txs_in_block, participation, requeue)
    assert B == 0.0 and n == 0


def test_finalize_declares_caps_active_validators_k(engine, mk_account):
    """Не более MAX_ACTIVE_VALIDATORS_K declare в блоке."""
    txs = []
    K = MAX_ACTIVE_VALIDATORS_K
    # cap большой, чтобы λ-фильтр не отрезал заранее
    L = 10_000.0
    for i in range(K + 5):
        addr = f"v_mass_{i}"
        mk_account(addr, role=Role.VALIDATOR, frozen=L, ant=1000.0, zkp=True)
        # Очень маленький b, чтобы Σb << cap
        txs.append(_mk_declare(addr, b=0.001, s=1.0))

    txs_in_block: list = []
    participation: dict = {}
    requeue: list = []
    engine._finalize_declares_batch(txs, txs_in_block, participation, requeue)

    assert len(participation) <= MAX_ACTIVE_VALIDATORS_K
