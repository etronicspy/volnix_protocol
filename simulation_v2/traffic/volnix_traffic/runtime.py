"""Orchestrator: height-polled tick = citizens → independent enrichment agents."""

from __future__ import annotations

import asyncio
import logging
from fractions import Fraction
from pathlib import Path
from typing import Any, Dict, List, Optional

from volnix_traffic import actions
from volnix_traffic.citizens import CitizenTraffic
from volnix_traffic.client import NodeClient
from volnix_traffic.declare import suggest_declare_bs
from volnix_traffic.engine import BotEngine
from volnix_traffic.profit import snapshot_from_chain
from volnix_traffic.registry import KIND_CITIZEN, KIND_ENRICHMENT, KIND_RETIRED, BotRegistry
from volnix_traffic.settings import TrafficSettings
from volnix_traffic.supervisor import EnrichmentSupervisor

log = logging.getLogger("volnix_traffic.loop")

_DEFAULT_STATE = Path(__file__).resolve().parent.parent / "data" / "wallets.json"
MAX_CATCHUP = 20


def effective_poll_sec(produce_interval_sec: float, configured_poll: float) -> float:
    """Poll often enough to see new heights when the chain is faster than configured poll."""
    configured = max(0.001, float(configured_poll))
    produce = max(0.001, float(produce_interval_sec))
    return max(0.001, min(configured, produce * 0.25))


class TrafficRuntime:
    def __init__(self, settings: TrafficSettings, state_path: Optional[Path] = None) -> None:
        self.settings = settings
        self.state_path = state_path or _DEFAULT_STATE
        self.client = NodeClient(settings.node_url)
        self.registry = BotRegistry()
        # Kept for intensity + legacy BotEngine tests; step() is not on the default path.
        self.bots = BotEngine(settings, self.registry)
        self.supervisor = EnrichmentSupervisor(settings, self.registry)
        self.citizens = CitizenTraffic(settings, self.registry)
        self.running = False
        self.last_height = -1
        self.last_tick: Dict[str, Any] = {}
        self.produce_interval_sec: float = 60.0
        self.attempt_window_sec: float = 60.0
        self.pace_debt_blocks: int = 0
        self.effective_poll_sec: float = float(settings.poll_interval_sec)
        self._task: Optional[asyncio.Task] = None
        # Created on first use so it binds to uvicorn's loop (Python 3.9 locks capture the loop at init).
        self._op_lock: Optional[asyncio.Lock] = None
        self.release = False
        self.intensity = float(settings.intensity)

    def _lock(self) -> asyncio.Lock:
        if self._op_lock is None:
            self._op_lock = asyncio.Lock()
        return self._op_lock

    def set_intensity(self, value: float) -> None:
        # README + TrafficPanel: spend share 0–10 (0 = none, 10 = everyone with spare WRT).
        self.intensity = max(0.0, min(10.0, float(value)))
        self.bots.set_intensity(self.intensity)

    def status(self) -> Dict[str, Any]:
        roles = {"citizen": 0, "supplier": 0, "validator": 0}
        pools = {KIND_CITIZEN: 0, KIND_ENRICHMENT: 0, KIND_RETIRED: 0}
        genesis_addr = ""
        for b in self.registry.all():
            roles[b.role] = roles.get(b.role, 0) + 1
            pools[b.kind] = pools.get(b.kind, 0) + 1
            if b.genesis:
                genesis_addr = b.address
        floor = {
            "min_suppliers": self.settings.min_suppliers,
            "min_validators": self.settings.min_validators,
            "suppliers": len(self.registry.suppliers()),
            "validators": len(self.registry.validators()),
        }
        errors = list(self.supervisor.last_errors) + list(self.bots.last_errors)
        return {
            "running": self.running,
            "release": self.release,
            "height": self.last_height,
            "intensity": self.intensity,
            "wallets": len(self.registry),
            "roles": roles,
            "pools": pools,
            "floor": floor,
            "genesis": {"address": genesis_addr, "managed": bool(genesis_addr)},
            "agents": self.supervisor.agent_statuses(),
            "horizon_blocks": self.settings.horizon_blocks,
            "params": {
                "target_enrichment_bots": self.settings.target_enrichment_bots,
                "target_citizens": self.settings.target_citizens,
                "citizen_transfers_per_tick": self.settings.citizen_transfers_per_tick,
                "horizon_blocks": self.settings.horizon_blocks,
                "peer_sample_min": self.settings.peer_sample_min,
                "peer_sample_max": self.settings.peer_sample_max,
            },
            "last_tick": self.last_tick,
            "last_errors": errors[-20:],
            "node_url": self.settings.node_url,
            "produce_interval_sec": self.produce_interval_sec,
            "attempt_window_sec": self.attempt_window_sec,
            "pace_debt_blocks": self.pace_debt_blocks,
            "effective_poll_sec": self.effective_poll_sec,
            "flags": {
                "market": self.settings.enable_market,
                "bots": self.settings.enable_bots,
                "declare": self.settings.enable_declare,
            },
        }

    def wallets(self) -> List[Dict[str, Any]]:
        return [b.to_dict() for b in self.registry.all()]

    def start(self, intensity: Optional[float] = None) -> None:
        if intensity is not None:
            self.set_intensity(intensity)
        self.release = False
        self.running = True
        self._ensure_loop()

    def stop(self) -> None:
        self.running = False

    async def reset_bots(self) -> Dict[str, Any]:
        """Drop the local bot registry. Chain state is unchanged; the loop recruits again."""
        async with self._lock():
            self.registry.clear()
            self.supervisor.agents.clear()
            self.supervisor.board.clear()
            self.supervisor.last_errors.clear()
            self.supervisor._next_index = 0
            self.supervisor._forced_supplier_done = False
            self.citizens._next_index = 0
            self.bots._next_index = 0
            self.bots.last_errors.clear()
            self.last_tick = {}
            self.last_height = -1
            self.release = False
            self._persist()
            return self.status()

    async def release_bots(self) -> Dict[str, Any]:
        """Pause every bot except genesis. Genesis only posts a declare so heights can finalize."""
        async with self._lock():
            self.release = True
            self.running = True
            seed = self.settings.genesis_seed
            self.supervisor.agents = {
                key: agent for key, agent in self.supervisor.agents.items() if key == seed
            }
            self.supervisor.board = [agent.card() for agent in self.supervisor.agents.values()]
            self._ensure_loop()
            return self.status()

    async def _genesis_declare_only(self, accounts_by_addr: Dict[str, Dict[str, Any]], alpha: Fraction) -> bool:
        bot = await self.supervisor.adopt_genesis(self.client)
        if bot is None or not bot.address:
            return False
        row = accounts_by_addr.get(bot.address) or {}
        pair = suggest_declare_bs(int(row.get("lzn_activated") or 0), int(row.get("ant") or 0), alpha)
        if pair is None:
            return False
        return await actions.declare(self.client, bot, pair[0], pair[1])

    def _ensure_loop(self) -> None:
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            return
        if self._task is None or self._task.done():
            self._task = loop.create_task(self._loop(), name="traffic-loop")

    def _persist(self) -> None:
        try:
            self.registry.save(self.state_path)
        except OSError as exc:
            log.warning("persist wallets failed: %s", exc)

    def _sync_pace(self, summary: Dict[str, Any]) -> None:
        # Prefer wall-clock sleep (after stand time_scale); keep attempt_window for display.
        raw = summary.get("wall_sleep_sec")
        if raw is None:
            raw = summary.get("produce_interval_sec")
        if raw is None:
            raw = summary.get("attempt_window_sec")
        if raw is not None:
            try:
                wall = max(0.001, float(raw))
                self.produce_interval_sec = wall
            except (TypeError, ValueError):
                pass
        canon = summary.get("attempt_window_sec")
        if canon is not None:
            try:
                self.attempt_window_sec = max(0.001, float(canon))
            except (TypeError, ValueError):
                pass
        debt = summary.get("pace_debt_blocks")
        if debt is not None:
            try:
                self.pace_debt_blocks = int(debt)
            except (TypeError, ValueError):
                pass
        self.effective_poll_sec = effective_poll_sec(
            self.produce_interval_sec,
            self.settings.poll_interval_sec,
        )

    async def ensure_started(self) -> None:
        loaded = self.registry.load(self.state_path)
        if loaded:
            self.bots._next_index = max(self.bots._next_index, loaded)
            self.supervisor.sync_index(loaded)
            self.citizens.sync_index(loaded)
            log.info("loaded %s bot wallets from %s", loaded, self.state_path)
        if self.settings.autostart:
            self.running = True
        self._ensure_loop()

    async def aclose(self) -> None:
        self.running = False
        self._persist()
        if self._task and not self._task.done():
            self._task.cancel()
            try:
                await self._task
            except asyncio.CancelledError:
                pass
        await self.client.aclose()

    async def tick_once(self, height: Optional[int] = None) -> Dict[str, Any]:
        async with self._lock():
            return await self._tick_once(height)

    async def _tick_once(self, height: Optional[int] = None) -> Dict[str, Any]:
        summary = await self.client.chain_summary()
        self._sync_pace(summary)
        h = int(height if height is not None else (summary.get("height") or 0))
        params = await self.client.params()
        accounts = await self.client.accounts()
        orderbook = await self.client.orderbook("ANT/WRT")
        lzn_orderbook = await self.client.orderbook("LZN/WRT")
        by_addr = {a["address"]: a for a in accounts}
        snap = snapshot_from_chain(
            summary,
            orderbook,
            lzn_orderbook=lzn_orderbook,
            lambda_num=int(params.get("lambda_num") or 1),
            lambda_den=int(params.get("lambda_den") or 3),
        )
        snap.height = h

        citizen_n = bot_n = decl_n = market_n = 0
        if self.settings.enable_bots and self.release:
            declared = await self._genesis_declare_only(
                by_addr,
                Fraction(int(params.get("alpha_num") or 1), int(params.get("alpha_den") or 50)),
            )
            decl_n = 1 if declared else 0
            bot_n = 1
            self._persist()
        elif self.settings.enable_bots:
            await self.supervisor.adopt_genesis(self.client)
            max_new = self.settings.max_new_per_tick
            citizen_created = await self.citizens.ensure_pool(self.client, max_new=max_new)
            enr_created = await self.supervisor.grow_enrichment(
                self.client, snap, accounts, max_new=max_new
            )
            self.registry.sync_from_chain(accounts)
            # Re-fetch after creates so new addresses appear
            if citizen_created or enr_created:
                accounts = await self.client.accounts()
                by_addr = {a["address"]: a for a in accounts}
                self.registry.sync_from_chain(accounts)

            counts = await self.supervisor.tick_agents(
                self.client,
                height=h,
                snap=snap,
                accounts_by_addr=by_addr,
                alpha=Fraction(int(params.get("alpha_num") or 1), int(params.get("alpha_den") or 50)),
                lam=Fraction(int(params.get("lambda_num") or 1), int(params.get("lambda_den") or 3)),
                k=int(params.get("max_active_validators") or params.get("k") or 150),
                ant_orderbook=orderbook,
                lzn_orderbook=lzn_orderbook,
            )
            bot_n = counts.get("agents", 0)
            decl_n = counts.get("declare", 0)
            market_n = counts.get("market", 0)
            # Everyday spend after strategy txs: all wallets, not only citizens.
            citizen_n = await self.citizens.step(
                self.client, by_addr, intensity=self.intensity
            )
            self._persist()

        self.last_height = h
        self.last_tick = {
            "height": h,
            "market": market_n,
            "bots": bot_n,
            "declare": decl_n,
            "citizen_txs": citizen_n,
            "spend_txs": citizen_n,
        }
        return self.last_tick

    async def catch_up_to(self, height: int) -> int:
        """Tick each missed height up to MAX_CATCHUP; return number of ticks run."""
        if self.last_height < 0:
            await self.tick_once(height=height)
            return 1
        if height <= self.last_height:
            return 0
        target = min(height, self.last_height + MAX_CATCHUP)
        n = 0
        for h in range(self.last_height + 1, target + 1):
            await self.tick_once(height=h)
            n += 1
        return n

    async def _loop(self) -> None:
        log.info("traffic loop started → %s", self.settings.node_url)
        while True:
            try:
                if not self.running:
                    await asyncio.sleep(self.effective_poll_sec)
                    continue
                summary = await self.client.chain_summary()
                self._sync_pace(summary)
                height = int(summary.get("height") or 0)
                if height != self.last_height:
                    await self.catch_up_to(height)
                else:
                    await asyncio.sleep(self.effective_poll_sec)
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                log.warning("tick error: %s", exc)
                self.supervisor._note(f"loop: {exc}")
                await asyncio.sleep(max(1.0, self.effective_poll_sec))
