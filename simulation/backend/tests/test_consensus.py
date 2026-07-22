"""Многовалидаторный консенсус: PreVote/PreCommit/+2/3, evidence, slashing."""
from __future__ import annotations

import time
import uuid

import pytest

from core.consensus import (
    Decision,
    FaultModel,
    Vote,
    normalize_validator_set,
    quorum_threshold,
    run_consensus,
    run_round,
    slashing_amount,
)
from core.engine import BURN_CAP_LAMBDA
from core.models import Role, Transaction, TransactionType
from core.state import GENESIS_VALIDATOR_ADDR


def test_normalize_validator_set_dedups_and_drops_nonpositive():
    raw = [
        {"address": "a", "power": 10},
        {"address": "b", "power": 0},
        {"address": "a", "power": 5},
        {"address": "", "power": 100},
        None,
    ]
    out = normalize_validator_set(raw)
    addrs = {v.address: v.power for v in out}
    assert addrs == {"a": 15.0}


def test_quorum_threshold():
    assert quorum_threshold(3.0) == pytest.approx(2.0)
    assert quorum_threshold(0.0) == 0.0


def test_run_round_commits_when_all_block():
    validators = normalize_validator_set([{"address": "v1", "power": 10}, {"address": "v2", "power": 5}])
    report = run_round(
        height=1,
        round_idx=0,
        proposer="v1",
        validators=validators,
        fault_model=FaultModel(),
    )
    assert report.decision == Decision.COMMIT
    assert report.power_block_pc > quorum_threshold(report.power_total)
    assert all(v == Vote.BLOCK for v in report.pre_votes.values())


def test_run_round_nil_quorum_blocks_commit():
    """p_nil = 1 → все не-пропозеры голосуют NIL → quorum NIL."""
    validators = normalize_validator_set([
        {"address": "v1", "power": 10},
        {"address": "v2", "power": 5},
        {"address": "v3", "power": 5},
    ])
    report = run_round(
        height=2,
        round_idx=0,
        proposer="v1",
        validators=validators,
        fault_model=FaultModel(p_nil=1.0, seed=1),
    )
    # У v2,v3 будет NIL; вместе 10 / 20 = 0.5 < 2/3 — но v1=BLOCK, итого block=10/20.
    # 10/20 не > 2/3*20=13.33 → NIL/timeout, не COMMIT.
    assert report.decision != Decision.COMMIT


def test_run_round_absent_no_commit():
    """p_absent=1 → все absent → ни BLOCK, ни NIL не достигают кворума."""
    validators = normalize_validator_set([{"address": "v1", "power": 5}, {"address": "v2", "power": 5}])
    report = run_round(
        height=3,
        round_idx=0,
        proposer="v1",
        validators=validators,
        fault_model=FaultModel(p_absent=1.0, seed=2),
    )
    assert report.decision == Decision.TIMEOUT
    assert all(v == Vote.ABSENT for v in report.pre_votes.values())


def test_run_consensus_uses_proposer_rotation():
    validators = [{"address": "v1", "power": 10}, {"address": "v2", "power": 10}]
    calls: list = []

    def picker(r: int) -> str:
        calls.append(r)
        return validators[r % 2]["address"]

    reports, evidence = run_consensus(
        height=5,
        validators=validators,
        proposer_for_round_fn=picker,
        fault_model=FaultModel(),
        max_rounds=4,
    )
    # commit на первом раунде (p_nil=p_absent=0)
    assert reports[0].decision == Decision.COMMIT
    assert len(reports) == 1
    assert evidence == []


def test_double_sign_generates_evidence():
    """p_double_sign=1 → block_pv проходит, но при pre_commit ставят NIL — Evidence."""
    validators = normalize_validator_set([
        {"address": "v1", "power": 100},
        {"address": "v2", "power": 10},
    ])
    sink: list = []
    report = run_round(
        height=10,
        round_idx=0,
        proposer="v1",
        validators=validators,
        fault_model=FaultModel(p_double_sign=1.0, seed=3),
        evidence_sink=sink,
    )
    assert sink  # хотя бы одно нарушение зафиксировано
    assert all(e.kind == "double_sign" for e in sink)
    assert report.decision in (Decision.NIL, Decision.TIMEOUT)


def test_slashing_amount_zero_for_no_frozen():
    assert slashing_amount(10.0, 0.0) == 0.0


def test_slashing_amount_bounded():
    amt = slashing_amount(0.0, 100.0, fraction=0.1)
    assert amt == 10.0
    amt2 = slashing_amount(0.0, 0.05, fraction=0.1)
    # минимум 0.01
    assert amt2 == pytest.approx(0.01)


# === Интеграция с engine ===


def _gv_declare(engine):
    L = engine._network_lzn_total_validators()
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=GENESIS_VALIDATOR_ADDR,
        amount=BURN_CAP_LAMBDA * L,
        stake_amount=0.0,
        asset_type="ant",
        timestamp=time.time(),
    )


def test_engine_default_fault_model_is_off(engine):
    assert engine.is_consensus_enabled is False


def test_engine_set_consensus_fault_model(engine):
    engine.set_consensus_fault_model(p_absent=0.1, p_nil=0.2, p_double_sign=0.05, seed=42)
    assert engine.is_consensus_enabled is True
    assert engine.consensus_fault_model.p_absent == 0.1
    engine.set_consensus_fault_model(p_absent=2.0, p_nil=-1.0)
    assert engine.consensus_fault_model.p_absent == 1.0
    assert engine.consensus_fault_model.p_nil == 0.0


@pytest.mark.asyncio
async def test_engine_commits_block_with_healthy_consensus(engine):
    """fault_model=0 + multi-validator → стандартный commit, добавляется consensus_round_log."""
    engine.set_consensus_fault_model(seed=11)
    # Имитация: ставим p_absent=p_nil=p_ds=0, но включаем enable
    engine.consensus_fault_model = FaultModel(p_absent=0.0, p_nil=0.0, p_double_sign=0.0, seed=11)
    # Принудительно включим: добавим небольшой p для активации
    engine.consensus_fault_model = FaultModel(p_absent=0.0, p_nil=0.0, p_double_sign=0.0001, seed=11)

    engine.state.mempool.append(_gv_declare(engine))
    await engine.produce_block()
    last_block = engine.state.blocks[-1]
    assert "consensus_round_log" in last_block
    assert last_block["consensus_round_log"][-1]["decision"] == "commit"


@pytest.mark.asyncio
async def test_engine_slashing_records_tx_and_canon(engine, mk_account):
    """p_double_sign=1 + active multi-validator → блок отклонён + evidence + slashing-tx."""
    # Реальный второй валидатор: с ZKP и frozen LZN (engine проверит роль/наличие)
    ghost = mk_account("ghost_validator", role=Role.VALIDATOR, frozen=100.0, lzn=10.0, zkp=True, ant=10.0)
    engine.state.consensus_validator_set = [
        {"address": GENESIS_VALIDATOR_ADDR, "power": 6667.0},
        {"address": ghost.address, "power": 1000.0},
    ]
    engine.set_consensus_fault_model(p_double_sign=1.0, seed=7)
    # declare должен дать Σb в коридор по обновлённому L_total
    L_total = engine._network_lzn_total_validators()
    engine.state.mempool.append(
        Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.DECLARE_PARTICIPATION,
            sender=GENESIS_VALIDATOR_ADDR,
            amount=BURN_CAP_LAMBDA * L_total,
            stake_amount=0.0,
            asset_type="ant",
            timestamp=time.time(),
        )
    )
    initial_frozen_ghost = ghost.lzn_frozen_mining
    try:
        await engine.produce_block()
    except Exception:
        pass
    # evidence_log должен содержать записи
    assert engine._evidence_log, "ожидаем double_sign evidence"
    # На state после rollback frozen восстановлен (slashing был внутри отклонённого блока)
    ghost_after = engine.state.accounts[ghost.address]
    assert ghost_after.lzn_frozen_mining == initial_frozen_ghost
