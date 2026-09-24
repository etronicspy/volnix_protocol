"""Orchestrator: height-polled tick = market → bots → declare."""

from __future__ import annotations

import asyncio
import logging
from pathlib import Path
from typing import Any, Dict, List, Optional, Set

from volnix_traffic.client import NodeClient
from volnix_traffic.declare import AutoDeclare
from volnix_traffic.engine import BotEngine
from volnix_traffic.market import AutoMarket
from volnix_traffic.profit import snapshot_from_chain
from volnix_traffic.registry import BotRegistry
from volnix_traffic.settings import TrafficSettings

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
        # Single source of truth — always go through bots.registry
        self.bots = BotEngine(settings, BotRegistry())
        self.market = AutoMarket()
        self.declare = AutoDeclare()
        self.running = False
        self.last_height = -1
        self.last_tick: Dict[str, Any] = {}
        self.produce_interval_sec: float = 1.0
        self.effective_poll_sec: float = float(settings.poll_interval_sec)
        self._task: Optional[asyncio.Task] = None

    @property
    def registry(self) -> BotRegistry:
        return self.bots.registry

    def status(self) -> Dict[str, Any]:
        roles = {"citizen": 0, "supplier": 0, "validator": 0}
        for b in self.registry.all():
            roles[b.role] = roles.get(b.role, 0) + 1
        return {
            "running": self.running,
            "height": self.last_height,
            "intensity": self.bots.intensity,
            "wallets": len(self.registry),
            "roles": roles,
            "last_tick": self.last_tick,
            "last_errors": list(self.bots.last_errors),
            "node_url": self.settings.node_url,
            "produce_interval_sec": self.produce_interval_sec,
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
            self.bots.set_intensity(intensity)
        self.running = True
        self._ensure_loop()

    def stop(self) -> None:
        self.running = False

    def _ensure_loop(self) -> None:
        """Schedule the poll loop if an event loop is running."""
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
        raw = summary.get("produce_interval_sec")
        if raw is not None:
            try:
                self.produce_interval_sec = max(0.001, float(raw))
            except (TypeError, ValueError):
                pass
        self.effective_poll_sec = effective_poll_sec(
            self.produce_interval_sec,
            self.settings.poll_interval_sec,
        )

    async def ensure_started(self) -> None:
        """Call from FastAPI startup: load state, attach loop, optionally begin ticking."""
        loaded = self.registry.load(self.state_path)
        if loaded:
            self.bots._next_index = max(self.bots._next_index, loaded)
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
        summary = await self.client.chain_summary()
        self._sync_pace(summary)
        h = int(height if height is not None else (summary.get("height") or 0))
        params = await self.client.params()
        accounts = await self.client.accounts()
        orderbook = await self.client.orderbook("ANT/WRT")
        lzn_orderbook = await self.client.orderbook("LZN/WRT")
        by_addr = {a["address"]: a for a in accounts}
        reg = self.registry

        lam = float(params.get("lambda_num") or 1) / float(params.get("lambda_den") or 3)
        block_reward = int(
            (summary.get("params") or {}).get("current_block_reward")
            or (summary.get("params") or {}).get("base_block_reward")
            or 50_000_000
        )
        l_total = int(summary.get("l_total") or 0)
        snap = snapshot_from_chain(
            summary,
            orderbook,
            lzn_orderbook=lzn_orderbook,
            lambda_num=int(params.get("lambda_num") or 1),
            lambda_den=int(params.get("lambda_den") or 3),
        )

        market_n = bot_n = decl_n = 0
        if self.settings.enable_market:
            market_n = await self.market.step(
                self.client,
                reg,
                height=h,
                accounts_by_addr=by_addr,
                orderbook=orderbook,
                block_reward=block_reward,
                lambda_f=lam,
                l_total=l_total,
                lzn_orderbook=lzn_orderbook,
                lzn_price=snap.lzn_price,
            )
        if self.settings.enable_bots:
            bot_n = await self.bots.step(
                self.client,
                height=h,
                summary=summary,
                accounts=accounts,
                orderbook=orderbook,
                params=params,
                lzn_orderbook=lzn_orderbook,
            )
            self._persist()
        if self.settings.enable_declare:
            pending: Set[str] = set()
            try:
                unc = await self.client.unconfirmed()
                for tx in (unc.get("result") or {}).get("txs") or []:
                    if isinstance(tx, dict) and tx.get("signer"):
                        pending.add(str(tx["signer"]))
            except Exception:
                pass
            decl_n = await self.declare.step(
                self.client,
                reg,
                accounts_by_addr=by_addr,
                lambda_num=int(params.get("lambda_num") or 1),
                lambda_den=int(params.get("lambda_den") or 3),
                alpha_num=int(params.get("alpha_num") or 1),
                alpha_den=int(params.get("alpha_den") or 50),
                pending_signers=pending,
            )

        self.last_height = h
        self.last_tick = {
            "height": h,
            "market": market_n,
            "bots": bot_n,
            "declare": decl_n,
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
                    try:
                        await self.catch_up_to(height)
                    except Exception:
                        # Preserve last_height only on success inside catch_up/tick;
                        # on failure leave pointer so next loop retries.
                        raise
                else:
                    await asyncio.sleep(self.effective_poll_sec)
            except asyncio.CancelledError:
                raise
            except Exception as exc:
                log.warning("tick error: %s", exc)
                self.bots._note_error(f"loop: {exc}")
                await asyncio.sleep(max(1.0, self.effective_poll_sec))
