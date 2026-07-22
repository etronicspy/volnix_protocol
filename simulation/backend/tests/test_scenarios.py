"""Тесты YAML-сценарного движка."""
from __future__ import annotations

import time
import uuid

import pytest
import yaml

from core.bot_engine import BotEngine
from core.engine import BURN_CAP_LAMBDA
from core.models import Transaction, TransactionType
from core.scenarios import (
    ScenarioRunner,
    list_scenarios,
    load_scenario_file,
)
from core.state import GENESIS_VALIDATOR_ADDR


def _enqueue_canonical_declare(state_manager, engine):
    """Хелпер: добавляет declare от genesis-валидатора с b = λ·L_total."""
    gv = state_manager.accounts[GENESIS_VALIDATOR_ADDR]
    L_total = engine._network_lzn_total_validators()
    state_manager.mempool.append(
        Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.DECLARE_PARTICIPATION,
            sender=gv.address,
            amount=BURN_CAP_LAMBDA * L_total,
            stake_amount=0.0,
            asset_type="ant",
            timestamp=time.time(),
        )
    )


def test_list_scenarios_returns_yaml_files():
    names = list_scenarios()
    assert "epoch_wipe_stress.yaml" in names
    assert all(n.endswith(".yaml") for n in names)


def test_load_scenario_file_by_name():
    data = load_scenario_file("epoch_wipe_stress.yaml")
    assert "steps" in data
    assert isinstance(data["steps"], list)


@pytest.mark.asyncio
async def test_runner_mint_and_wait(state_manager, engine):
    runner = ScenarioRunner(state_manager, engine)
    scenario = yaml.safe_load(
        """
        name: smoke
        steps:
          - create_account: { address: alice }
          - mint: { receiver: alice, amount: 100, asset: wrt }
        asserts:
          - balance: { address: alice, asset: wrt, op: ">=", value: 100 }
          - block_height: { op: ">=", value: 1 }
        """
    )
    # Канон §5.4: чтобы блок принялся, нужен declare валидатора с b=λ·L_total
    _enqueue_canonical_declare(state_manager, engine)
    # Запускаем шаги, потом сами производим блок (вызывать wait_blocks внутри
    # после declare не получится — мемпул заполняется ДО шага).
    report = await runner.run(scenario)
    assert report.error is None
    await engine.produce_block()
    # Re-run asserts after block — но в текущем API нет такого. Просто проверяем напрямую.
    assert state_manager.current_height >= 1
    assert state_manager.accounts["alice"].wrt_balance >= 100


@pytest.mark.asyncio
async def test_runner_unknown_step_records_error(state_manager, engine):
    runner = ScenarioRunner(state_manager, engine)
    scenario = {"name": "bad", "steps": [{"unknown_step": {}}]}
    report = await runner.run(scenario)
    assert report.error is not None
    assert "Unknown" in report.error or "unknown" in report.error.lower()
    assert not report.passed


@pytest.mark.asyncio
async def test_runner_consensus_step(state_manager, engine):
    runner = ScenarioRunner(state_manager, engine)
    scenario = {
        "name": "consensus_fault",
        "steps": [{"consensus": {"p_double_sign": 1.0, "seed": 42}}],
    }
    report = await runner.run(scenario)
    assert report.error is None
    fm = engine.consensus_fault_model
    assert fm.p_double_sign == 1.0


@pytest.mark.asyncio
async def test_runner_bot_start_requires_attached_bot(state_manager, engine):
    runner = ScenarioRunner(state_manager, engine, bot_engine=None)
    scenario = {"name": "needs_bot", "steps": [{"bot_start": {"intensity": 1.0}}]}
    report = await runner.run(scenario)
    assert report.error is not None


@pytest.mark.asyncio
async def test_runner_bot_start_with_bot(state_manager, engine):
    bot = BotEngine(state_manager)
    runner = ScenarioRunner(state_manager, engine, bot_engine=bot)
    scenario = {
        "name": "with_bot",
        "steps": [
            {"bot_start": {"intensity": 2.0, "probe_ratio": 0.1, "enable_probes": True}},
            {"bot_stop": {}},
        ],
    }
    report = await runner.run(scenario)
    assert report.error is None
    assert bot.tx_per_second == 2.0  # intensity was set; stop() only flips is_running
    assert bot.is_running is False
    assert bot.probe_ratio == 0.1


@pytest.mark.asyncio
async def test_runner_assert_failure_in_report(state_manager, engine):
    runner = ScenarioRunner(state_manager, engine)
    scenario = {
        "name": "assert_fail",
        "steps": [],
        "asserts": [{"block_height": {"op": ">=", "value": 99999}}],
    }
    report = await runner.run(scenario)
    assert report.error is None
    assert not report.passed
    assert len(report.asserts) == 1
    assert not report.asserts[0].passed
