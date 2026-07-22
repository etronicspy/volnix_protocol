"""Multi-validator consensus: при len(ValidatorSet) >= 2 автоматически включается
PreVote/PreCommit без обязательных faults; decision = COMMIT, evidence = []."""
from __future__ import annotations

import pytest

from core import auto_declare
from core.engine import BURN_CAP_LAMBDA, SimulationEngine
from core.models import Role
from core.state import GENESIS_VALIDATOR_ADDR, StateManager


def _seed_secondary_validator(sm: StateManager, addr: str, frozen: float = 1500.0) -> None:
    acc = sm.create_account(addr)
    acc.role = Role.VALIDATOR
    acc.zkp_verified = True
    acc.lzn_frozen_mining = float(frozen)
    acc.ant_balance = 10_000.0
    sm.consensus_validator_set = [
        *sm.consensus_validator_set,
        {"address": addr, "power": float(frozen)},
    ]


def test_is_consensus_enabled_false_with_single_validator(state_manager: StateManager, engine: SimulationEngine):
    assert len(state_manager.consensus_validator_set) == 1
    assert engine.is_consensus_enabled is False


def test_is_consensus_enabled_true_when_ge_two_validators(state_manager: StateManager, engine: SimulationEngine):
    _seed_secondary_validator(state_manager, "val_2", frozen=1000.0)
    assert len(state_manager.consensus_validator_set) == 2
    assert engine.is_consensus_enabled is True


def test_is_consensus_enabled_true_with_faults_single_validator(state_manager: StateManager, engine: SimulationEngine):
    engine.set_consensus_fault_model(p_absent=0.5)
    assert engine.is_consensus_enabled is True


@pytest.mark.asyncio
async def test_multi_validator_block_commits(state_manager: StateManager, engine: SimulationEngine):
    """ValidatorSet из 2+ узлов без faults → decision=COMMIT, блок принят, evidence пусто."""
    _seed_secondary_validator(state_manager, "val_2", frozen=1000.0)
    _seed_secondary_validator(state_manager, "val_3", frozen=900.0)

    # каждый валидатор подаёт корректный declare через AutoDeclareDaemon
    n = auto_declare.step_once(state_manager, engine)
    assert n == 3  # три валидатора

    h0 = state_manager.current_height
    await engine.produce_block()
    assert state_manager.current_height == h0 + 1

    block = state_manager.blocks[-1]
    round_log = block.get("consensus_round_log", []) or []
    assert round_log, "consensus_round_log must be populated when ValidatorSet >= 2"
    last_round = round_log[-1]
    assert last_round["decision"] == "commit"
    votes = last_round.get("votes") or {}
    pv = votes.get("pre_vote") or {}
    pc = votes.get("pre_commit") or {}
    # все 3 валидатора участвовали в голосовании
    assert set(pv.keys()) == {GENESIS_VALIDATOR_ADDR, "val_2", "val_3"}
    assert set(pc.keys()) == {GENESIS_VALIDATOR_ADDR, "val_2", "val_3"}
