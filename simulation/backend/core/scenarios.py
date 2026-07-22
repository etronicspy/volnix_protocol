"""YAML-сценарии: детерминированный прогон шагов + ассерты на состояние.

Сценарий описывается YAML-файлом:

    name: epoch_wipe_stress
    seed: 42
    description: "Резкий поворот эпохи: §5.5 wipe → emission"
    steps:
      - mint: { receiver: "p1", amount: 100, asset: "ant" }
      - set_role: { address: "p1", role: "provider" }
      - bot_start: { intensity: 5, probe_ratio: 0.2 }
      - wait_blocks: 50
      - bot_stop: {}
    asserts:
      - balance:
          address: "p1"
          asset: "ant"
          op: ">="
          value: 0

Запуск:
  CLI:   python -m core.scenarios run path/to/scenario.yaml
  REST:  POST /api/scenarios/run  body={"path": "scenarios/epoch.yaml"}

Шаги поддерживают единый admission (`validate_and_build_tx`/`validate_treasury_mint`).
Repository scenarios — в каталоге `simulation/scenarios/`.
"""
from __future__ import annotations

import asyncio
import os
import random
import time
import uuid
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Callable, Dict, List, Optional, Union

import yaml

from core.bot_engine import BotEngine
from core.engine import SimulationEngine
from core.models import Role
from core.state import SIM_TREASURY_ADDR, StateManager
from core.wallet_validate import validate_and_build_tx, validate_treasury_mint

SCENARIOS_DIR = Path(__file__).resolve().parents[2] / "scenarios"


@dataclass
class AssertResult:
    description: str
    passed: bool
    expected: Any = None
    actual: Any = None
    detail: str = ""


@dataclass
class ScenarioReport:
    name: str
    steps_executed: int
    blocks_produced: int
    asserts: List[AssertResult] = field(default_factory=list)
    duration_sec: float = 0.0
    error: Optional[str] = None

    @property
    def passed(self) -> bool:
        return self.error is None and all(a.passed for a in self.asserts)

    def to_dict(self) -> dict:
        return {
            "name": self.name,
            "passed": self.passed,
            "error": self.error,
            "steps_executed": self.steps_executed,
            "blocks_produced": self.blocks_produced,
            "duration_sec": round(self.duration_sec, 3),
            "asserts": [
                {
                    "description": a.description,
                    "passed": a.passed,
                    "expected": a.expected,
                    "actual": a.actual,
                    "detail": a.detail,
                }
                for a in self.asserts
            ],
        }


def _cmp(op: str, actual: float, expected: float) -> bool:
    if op == "==":
        return abs(actual - expected) < 1e-9
    if op == "!=":
        return abs(actual - expected) >= 1e-9
    if op == ">=":
        return actual + 1e-9 >= expected
    if op == "<=":
        return actual - 1e-9 <= expected
    if op == ">":
        return actual > expected + 1e-9
    if op == "<":
        return actual < expected - 1e-9
    raise ValueError(f"Unknown comparator: {op}")


def _account_balance(sm: StateManager, address: str, asset: str) -> float:
    acc = sm.accounts.get(address)
    if not acc:
        return 0.0
    a = asset.lower()
    if a == "wrt":
        return float(acc.wrt_balance)
    if a == "lzn":
        return float(acc.lzn_balance)
    if a == "ant":
        return float(acc.ant_balance)
    if a == "frozen":
        return float(acc.lzn_frozen_mining)
    raise ValueError(f"Unknown asset: {asset}")


class ScenarioRunner:
    """Запуск YAML-сценария над уже инициализированной симуляцией.

    Создаётся одноразово на каждый прогон; не пересоздаёт state.
    """

    def __init__(
        self,
        state_manager: StateManager,
        engine: SimulationEngine,
        bot_engine: Optional[BotEngine] = None,
    ) -> None:
        self.sm = state_manager
        self.engine = engine
        self.bot = bot_engine
        self.blocks_produced = 0

    async def run(self, scenario: dict) -> ScenarioReport:
        name = scenario.get("name") or "unnamed"
        seed = scenario.get("seed")
        if seed is not None:
            random.seed(int(seed))
        steps = scenario.get("steps") or []
        asserts_def = scenario.get("asserts") or []
        report = ScenarioReport(name=name, steps_executed=0, blocks_produced=0)
        t0 = time.time()
        try:
            for step in steps:
                if not isinstance(step, dict) or not step:
                    continue
                await self._run_step(step)
                report.steps_executed += 1
            for a in asserts_def:
                report.asserts.append(self._run_assert(a))
        except Exception as e:
            report.error = f"{type(e).__name__}: {e}"
        report.blocks_produced = self.blocks_produced
        report.duration_sec = time.time() - t0
        return report

    async def _run_step(self, step: dict) -> None:
        # step имеет вид {action_name: {...args}} (один ключ)
        op, args = next(iter(step.items()))
        args = args or {}
        handler: Optional[Callable[..., Any]] = getattr(self, f"_step_{op}", None)
        if handler is None:
            raise ValueError(f"Unknown scenario step: {op}")
        result = handler(**args)
        if asyncio.iscoroutine(result):
            await result

    # === Step implementations ===

    def _ensure_account(self, address: str, role: Optional[str] = None, zkp: bool = False) -> None:
        acc = self.sm.accounts.get(address) or self.sm.create_account(address)
        if role:
            acc.role = Role(role)
        if zkp:
            acc.zkp_verified = True

    def _push_tx(self, tx, sender: str = "") -> None:
        """Route tx via NetworkSim (if attached) or legacy mempool."""
        net = getattr(self.sm, "network", None)
        if net is not None:
            addr = sender or getattr(tx, "sender", "") or ""
            try:
                if addr:
                    net.submit_from_addr(addr, tx)
                else:
                    net.submit_to("node_0", tx)
                return
            except Exception:
                pass
        self.sm.mempool.append(tx)

    def _step_create_account(self, address: str, role: Optional[str] = None, zkp: bool = False, **_) -> None:
        self._ensure_account(address, role=role, zkp=zkp)

    def _step_mint(
        self,
        receiver: str,
        amount: float,
        asset: str = "wrt",
        **_: Any,
    ) -> None:
        self._ensure_account(receiver)
        ok, msg, tx = validate_treasury_mint(self.sm, receiver, float(amount), asset)
        if not ok or tx is None:
            raise RuntimeError(f"mint rejected: {msg}")
        self._push_tx(tx)

    def _step_set_role(self, address: str, role: str, **_: Any) -> None:
        self._ensure_account(address)
        ok, msg, tx = validate_and_build_tx(
            self.sm, "set_role", address, role=Role(role)
        )
        if not ok or tx is None:
            raise RuntimeError(f"set_role rejected: {msg}")
        self._push_tx(tx, address)

    def _step_verify_zkp(self, address: str, **_: Any) -> None:
        self._ensure_account(address)
        ok, msg, tx = validate_and_build_tx(self.sm, "verify_zkp", address)
        if not ok or tx is None:
            raise RuntimeError(f"verify_zkp rejected: {msg}")
        self._push_tx(tx, address)

    def _step_transfer(
        self,
        sender: str,
        to: str,
        amount: float,
        asset: str = "wrt",
        **_: Any,
    ) -> None:
        ok, msg, tx = validate_and_build_tx(
            self.sm, "transfer", sender, to_address=to, amount=float(amount), asset=asset
        )
        if not ok or tx is None:
            raise RuntimeError(f"transfer rejected: {msg}")
        self._push_tx(tx, sender)

    def _step_order(
        self,
        address: str,
        side: str,
        amount: float,
        price: Optional[float] = None,
        market: bool = False,
        max_wrt: Optional[float] = None,
        **_: Any,
    ) -> None:
        ok, msg, tx = validate_and_build_tx(
            self.sm,
            "create_order",
            address,
            side=side,
            amount=float(amount),
            price=float(price or 0.0),
            market=bool(market),
            max_wrt=max_wrt,
        )
        if not ok or tx is None:
            raise RuntimeError(f"order rejected: {msg}")
        self._push_tx(tx, address)

    def _step_cancel_order(self, address: str, order_id: str, **_: Any) -> None:
        ok, msg, tx = validate_and_build_tx(
            self.sm, "cancel_order", address, order_id=order_id
        )
        if not ok or tx is None:
            raise RuntimeError(f"cancel rejected: {msg}")
        self._push_tx(tx, address)

    def _step_declare(
        self,
        address: str,
        burn: float,
        stake: float = 0.0,
        **_: Any,
    ) -> None:
        ok, msg, tx = validate_and_build_tx(
            self.sm, "declare", address, burn_b=float(burn), stake_s=float(stake)
        )
        if not ok or tx is None:
            raise RuntimeError(f"declare rejected: {msg}")
        self._push_tx(tx, address)

    async def _step_wait_blocks(self, n: int = 1, **_: Any) -> None:
        net = getattr(self.sm, "network", None)
        # Если NetworkSim активен с латентностью > 0, нужно дать gossip-таскам
        # отработать: иначе flush_to_global ничего не отдаст (call_later ждёт latency).
        gossip_delay = 0.0
        if net is not None:
            gossip_delay = max(0.0, float(net.gossip_latency_ms) / 1000.0)
        for _ in range(int(n)):
            if gossip_delay > 0:
                # 2 × latency + epsilon, чтобы покрыть одиночный hop из любого узла.
                await asyncio.sleep(gossip_delay * 2 + 0.01)
            await self.engine.produce_block()
            self.blocks_produced += 1

    def _step_set_block_time(self, seconds: float, **_: Any) -> None:
        self.engine.set_block_time(float(seconds))

    def _step_bot_start(
        self,
        intensity: float = 1.0,
        probe_ratio: Optional[float] = None,
        enable_probes: Optional[bool] = None,
        **_: Any,
    ) -> None:
        if self.bot is None:
            raise RuntimeError("Bot engine is not attached to scenario runner")
        self.bot.set_intensity(float(intensity))
        if enable_probes is not None or probe_ratio is not None:
            self.bot.set_probe_settings(enable=enable_probes, ratio=probe_ratio)

    def _step_bot_stop(self, **_: Any) -> None:
        if self.bot is None:
            return
        self.bot.stop()

    def _step_bot_tick(self, n: int = 1, **_: Any) -> None:
        """Прогнать N итераций bot.generate_traffic() — детерминированно для тестов."""
        if self.bot is None:
            raise RuntimeError("Bot engine is not attached to scenario runner")
        for _ in range(int(n)):
            self.bot.generate_traffic()

    def _step_consensus(
        self,
        p_absent: float = 0.0,
        p_nil: float = 0.0,
        p_double_sign: float = 0.0,
        seed: Optional[int] = None,
        **_: Any,
    ) -> None:
        self.engine.set_consensus_fault_model(
            p_absent=p_absent, p_nil=p_nil, p_double_sign=p_double_sign, seed=seed
        )

    def _step_auto_declare(self, **_: Any) -> None:
        """Один тик AutoDeclareDaemon: подаёт canon-declare за каждого активного валидатора.

        Эквивалент `auto_declare.step_once(state, engine)`. Полезно перед `wait_blocks`,
        чтобы валидаторы получали награды §5.1/§5.4 (v2: без declare блок валиден,
        но b_i = 0 → доход = 0).
        """
        from core.auto_declare import step_once

        step_once(self.sm, self.engine)

    def _step_network_set(
        self,
        num_nodes: Optional[int] = None,
        latency_ms: Optional[int] = None,
        loss_pct: Optional[float] = None,
        quorum_pct: Optional[float] = None,
        **_: Any,
    ) -> None:
        """Включить / переконфигурировать NetworkSim прямо в сценарии.

        - Если `num_nodes >= 2` и `state.network is None` → создать NetworkSim и attach.
        - Прочие параметры применяются «горячо» к существующему network.
        """
        net = getattr(self.sm, "network", None)
        if net is None and num_nodes is not None and int(num_nodes) >= 2:
            from core.network import NetworkSim

            net = NetworkSim(
                num_nodes=int(num_nodes),
                gossip_latency_ms=int(latency_ms or 100),
                gossip_loss_pct=float(loss_pct or 0.0),
                quorum_pct=float(quorum_pct) if quorum_pct is not None else 2.0 / 3.0,
            )
            net.attach(self.sm)
            self.sm.network = net
            return
        if net is None:
            return
        net.set_config(
            gossip_latency_ms=latency_ms,
            gossip_loss_pct=loss_pct,
            quorum_pct=quorum_pct,
        )

    # === Asserts ===

    def _run_assert(self, a: dict) -> AssertResult:
        if "balance" in a:
            spec = a["balance"]
            addr = spec["address"]
            asset = spec.get("asset", "wrt")
            op = spec.get("op", "==")
            expected = float(spec.get("value", 0))
            actual = _account_balance(self.sm, addr, asset)
            passed = _cmp(op, actual, expected)
            return AssertResult(
                description=f"balance({addr}.{asset}) {op} {expected}",
                passed=passed,
                expected=expected,
                actual=actual,
                detail="",
            )
        if "block_height" in a:
            spec = a["block_height"]
            op = spec.get("op", ">=")
            expected = int(spec.get("value", 0))
            actual = int(self.sm.current_height)
            passed = _cmp(op, float(actual), float(expected))
            return AssertResult(
                description=f"block_height {op} {expected}",
                passed=passed,
                expected=expected,
                actual=actual,
            )
        if "canon_log" in a:
            spec = a["canon_log"]
            status = spec.get("status")
            canon = spec.get("canon")
            min_count = int(spec.get("min_count", 1))
            cnt = 0
            for entry in self.sm.canon_log.to_list_newest_first():
                if status and entry.get("status") != status:
                    continue
                if canon and entry.get("canon") != canon:
                    continue
                cnt += 1
            passed = cnt >= min_count
            return AssertResult(
                description=f"canon_log status={status} canon={canon} count>={min_count}",
                passed=passed,
                expected=min_count,
                actual=cnt,
            )
        if "mempool_size" in a:
            spec = a["mempool_size"]
            op = spec.get("op", "<=")
            expected = int(spec.get("value", 0))
            actual = len(self.sm.mempool)
            passed = _cmp(op, float(actual), float(expected))
            return AssertResult(
                description=f"mempool_size {op} {expected}",
                passed=passed,
                expected=expected,
                actual=actual,
            )
        return AssertResult(
            description=f"unknown assert: {list(a.keys())}",
            passed=False,
            detail="нет обработчика для этого assert-типа",
        )


def load_scenario_file(path: Union[str, Path]) -> dict:
    p = Path(path)
    if not p.is_absolute():
        # ищем сначала в SCENARIOS_DIR, потом относительно CWD
        candidate = SCENARIOS_DIR / p
        if candidate.exists():
            p = candidate
    with open(p, encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}
    if not isinstance(data, dict):
        raise ValueError(f"scenario must be a YAML mapping: {path}")
    return data


def list_scenarios() -> List[str]:
    if not SCENARIOS_DIR.exists():
        return []
    return sorted(p.name for p in SCENARIOS_DIR.glob("*.yaml"))


async def run_scenario_file(
    path: Union[str, Path],
    *,
    state_manager: Optional[StateManager] = None,
    engine: Optional[SimulationEngine] = None,
    bot_engine: Optional[BotEngine] = None,
    reset_state: bool = True,
) -> ScenarioReport:
    sm = state_manager or StateManager()
    if reset_state:
        sm.reset_state()
    eng = engine or SimulationEngine(sm)
    bot = bot_engine or BotEngine(sm)
    runner = ScenarioRunner(sm, eng, bot)
    scenario = load_scenario_file(path)
    return await runner.run(scenario)


# === CLI ===

def _cli_main(argv: Optional[List[str]] = None) -> int:
    import argparse

    parser = argparse.ArgumentParser(prog="python -m core.scenarios")
    sub = parser.add_subparsers(dest="cmd", required=True)
    sub.add_parser("list", help="перечислить сценарии в simulation/scenarios/")
    p_run = sub.add_parser("run", help="запустить YAML-сценарий")
    p_run.add_argument("path", help="путь к YAML (абсолютный или относительно simulation/scenarios)")
    args = parser.parse_args(argv)
    if args.cmd == "list":
        for name in list_scenarios():
            print(name)
        return 0
    if args.cmd == "run":
        report = asyncio.run(run_scenario_file(args.path))
        import json

        print(json.dumps(report.to_dict(), ensure_ascii=False, indent=2))
        return 0 if report.passed else 1
    return 2


if __name__ == "__main__":
    raise SystemExit(_cli_main())
