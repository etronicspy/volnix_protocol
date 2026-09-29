"""Fan-out snapshot to independent EnrichmentAgents; genesis adopt; one flip/tick."""

from __future__ import annotations

import asyncio
import logging
from fractions import Fraction
from typing import Any, Dict, List, Optional

from volnix_traffic import actions
from volnix_traffic.agent import (
    AgentTickResult,
    EnrichmentAgent,
    FlipRequest,
    StrategyCard,
    balanced_profile,
)
from volnix_traffic.client import NodeClient
from volnix_traffic.compete import Rival
from volnix_traffic.profit import (
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    MarketSnapshot,
    ProfitStrategy,
    account_view_from_row,
)
from volnix_traffic.registry import (
    KIND_CITIZEN,
    KIND_ENRICHMENT,
    KIND_RETIRED,
    BotRegistry,
    BotWallet,
)
from volnix_traffic.settings import TrafficSettings

log = logging.getLogger("volnix_traffic.supervisor")

MIN_ORDER = 10_000


class EnrichmentSupervisor:
    def __init__(self, settings: TrafficSettings, registry: BotRegistry) -> None:
        self.settings = settings
        self.registry = registry
        self.agents: Dict[str, EnrichmentAgent] = {}
        self.board: List[StrategyCard] = []
        self.last_errors: List[str] = []
        self._next_index = 0
        self._forced_supplier_done = False
        self.profit = ProfitStrategy()

    def _note(self, msg: str) -> None:
        self.last_errors.append(msg)
        self.last_errors = self.last_errors[-20:]
        log.debug("%s", msg)

    def sync_index(self, hint: int) -> None:
        self._next_index = max(self._next_index, int(hint))

    def agent_statuses(self) -> List[Dict[str, Any]]:
        out: List[Dict[str, Any]] = []
        for ag in self.agents.values():
            pick = ag.last_pick
            out.append(
                {
                    "address": ag.wallet.address,
                    "role": ag.wallet.role,
                    "genesis": ag.wallet.genesis,
                    "expected_wrt": ag.expected_wrt,
                    "growth_wrt": ag.growth_wrt,
                    "b_i": pick.b_i if pick else 0,
                    "s_i": pick.s_i if pick else 0,
                    "enter": pick.enter if pick else False,
                    "adopted_from": ag.adopted_from,
                }
            )
        return out

    def _ensure_agent(self, wallet: BotWallet) -> EnrichmentAgent:
        key = wallet.seed
        if key not in self.agents:
            self.agents[key] = EnrichmentAgent(
                wallet,
                horizon=self.settings.horizon_blocks,
                peer_sample_min=self.settings.peer_sample_min,
                peer_sample_max=self.settings.peer_sample_max,
                role_flip_margin_wrt=self.settings.role_flip_margin_wrt,
                flip_confirm_ticks=self.settings.flip_confirm_ticks,
            )
            self.agents[key].strategy = balanced_profile()
            if wallet.role in (ROLE_SUPPLIER, ROLE_VALIDATOR):
                self.agents[key].strategy.desired_role = wallet.role
        else:
            self.agents[key].wallet = wallet
        return self.agents[key]

    def prune_retired(self) -> None:
        live = {b.seed for b in self.registry.enrichment()}
        for seed in list(self.agents):
            if seed not in live:
                del self.agents[seed]

    async def adopt_genesis(self, client: NodeClient) -> Optional[BotWallet]:
        seed = self.settings.genesis_seed
        existing = self.registry.get_by_seed(seed)
        if existing is not None:
            existing.kind = KIND_ENRICHMENT
            existing.genesis = True
            existing.role = existing.role or ROLE_VALIDATOR
            existing.verified = True
            self._ensure_agent(existing)
            return existing
        bot = BotWallet(
            seed=seed,
            role=ROLE_VALIDATOR,
            desired_role=ROLE_VALIDATOR,
            verified=True,
            kind=KIND_ENRICHMENT,
            genesis=True,
        )
        try:
            await actions.create_wallet(client, bot)
        except Exception as exc:
            self._note(f"genesis adopt: {exc}")
            return None
        if not bot.address:
            return None
        self.registry.add(bot)
        self._ensure_agent(bot)
        return bot

    async def grow_enrichment(
        self,
        client: NodeClient,
        snap: MarketSnapshot,
        accounts: List[Dict[str, Any]],
        *,
        max_new: int,
    ) -> int:
        created = 0
        n_sup = len(self.registry.suppliers())
        while len(self.registry.enrichment()) < self.settings.target_enrichment_bots and created < max_new:
            seed = BotRegistry.new_seed(self._next_index)
            self._next_index += 1
            bot = BotWallet(seed=seed, kind=KIND_ENRICHMENT)
            try:
                await actions.create_wallet(client, bot)
            except Exception as exc:
                self._note(f"create enrichment {seed}: {exc}")
                break
            self.registry.add(bot)
            try:
                await actions.fund_wrt(client, bot, self.settings.bootstrap_wrt)
            except Exception as exc:
                self._note(f"fund {bot.address}: {exc}")
            role = ROLE_SUPPLIER
            if self._forced_supplier_done or n_sup >= 1:
                view = account_view_from_row(
                    {"address": bot.address, "role": "citizen", "wrt": self.settings.bootstrap_wrt}
                )
                role = self.profit.best_first_role(view, snap)
                if role not in (ROLE_SUPPLIER, ROLE_VALIDATOR):
                    role = ROLE_VALIDATOR if n_sup >= 1 else ROLE_SUPPLIER
            if await actions.verify_identity(client, bot, role):
                bot.kind = KIND_ENRICHMENT
                if role == ROLE_SUPPLIER:
                    self._forced_supplier_done = True
                    n_sup += 1
                self._ensure_agent(bot)
            else:
                bot.kind = KIND_CITIZEN
            created += 1
        return created

    def build_rivals(
        self,
        accounts_by_addr: Dict[str, Dict[str, Any]],
        exclude: str,
    ) -> List[Rival]:
        rivals: List[Rival] = []
        for bot in self.registry.validators():
            if bot.address == exclude:
                continue
            row = accounts_by_addr.get(bot.address) or {}
            l_i = int(row.get("lzn_activated") or 0)
            ag = self.agents.get(bot.seed)
            if ag and ag.last_pick is not None:
                rivals.append(
                    Rival(address=bot.address, l_i=l_i, b_i=ag.last_pick.b_i, s_i=ag.last_pick.s_i)
                )
            elif l_i > 0:
                rivals.append(Rival(address=bot.address, l_i=l_i, b_i=l_i // 2, s_i=max(1, (2 * l_i) // 5)))
        return rivals

    async def tick_agents(
        self,
        client: NodeClient,
        *,
        height: int,
        snap: MarketSnapshot,
        accounts_by_addr: Dict[str, Dict[str, Any]],
        alpha: Fraction,
        lam: Fraction,
        k: int,
        ant_orderbook: Optional[Dict[str, Any]] = None,
        lzn_orderbook: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, int]:
        self.prune_retired()
        for bot in self.registry.enrichment():
            self._ensure_agent(bot)

        board = list(self.board)
        n_sup = max(1, int(snap.n_suppliers or 0))
        n_val = max(1, int(snap.n_validators or 0))
        # Use live enrichment counts for floor (more accurate than chain during bootstrap)
        live_sup = len(self.registry.suppliers()) or n_sup
        live_val = len(self.registry.validators()) or n_val

        async def run_one(ag: EnrichmentAgent) -> AgentTickResult:
            row = accounts_by_addr.get(ag.wallet.address) or {
                "address": ag.wallet.address,
                "role": ag.wallet.role,
            }
            rivals = self.build_rivals(accounts_by_addr, ag.wallet.address)
            return await ag.tick(
                client,
                height=height,
                snap=snap,
                row=row,
                rivals=rivals,
                board=board,
                alpha=alpha,
                lam=lam,
                k=k,
                n_suppliers=live_sup,
                n_validators=live_val,
                min_suppliers=self.settings.min_suppliers,
                min_validators=self.settings.min_validators,
                enable_market=self.settings.enable_market,
                enable_declare=self.settings.enable_declare,
                enable_role_flip=self.settings.enable_role_flip,
                ant_orderbook=ant_orderbook,
                lzn_orderbook=lzn_orderbook,
            )

        agents = list(self.agents.values())
        results: List[AgentTickResult] = []
        if agents:
            gathered = await asyncio.gather(
                *(run_one(ag) for ag in agents),
                return_exceptions=True,
            )
            for item in gathered:
                if isinstance(item, Exception):
                    self._note(f"agent: {item}")
                    results.append(AgentTickResult(error=str(item)))
                else:
                    results.append(item)
                    if item.error:
                        self._note(item.error)

        flip: Optional[FlipRequest] = None
        for res in results:
            if res.flip is not None:
                flip = res.flip
                break
        if flip is not None:
            ok = await self.execute_flip(client, flip, snap, accounts_by_addr)
            if not ok:
                flip.agent.flip_streak = 0

        self.board = [ag.card() for ag in self.agents.values()]
        return {
            "agents": len(agents),
            "declare": sum(1 for r in results if r.declared),
            "market": sum(1 for r in results if r.market),
            "adopted": sum(1 for r in results if r.adopted_from),
            "flips": 1 if flip is not None else 0,
        }

    async def execute_flip(
        self,
        client: NodeClient,
        req: FlipRequest,
        snap: MarketSnapshot,
        accounts_by_addr: Dict[str, Dict[str, Any]],
    ) -> bool:
        old = req.agent.wallet
        if old.genesis:
            return False
        live_sup = len(self.registry.suppliers())
        live_val = len(self.registry.validators())
        if old.role == ROLE_SUPPLIER and live_sup <= self.settings.min_suppliers:
            return False
        if old.role == ROLE_VALIDATOR and live_val <= self.settings.min_validators:
            return False

        row = accounts_by_addr.get(old.address) or {}
        # Cancel open orders if listed
        open_n = int(row.get("open_orders") or 0)
        if open_n > 0:
            try:
                detail = await client.account(old.address)
                for o in detail.get("open_orders") or []:
                    oid = str(o.get("order_id") or "")
                    if oid:
                        await actions.cancel_order(client, old, oid)
            except Exception as exc:
                self._note(f"flip cancel: {exc}")

        ant = int(row.get("ant") or 0)
        lzn = int(row.get("lzn") or 0)
        price_ant = max(1, snap.ant_price)
        price_lzn = max(1, snap.lzn_price)
        if old.role == ROLE_SUPPLIER and ant >= MIN_ORDER:
            await actions.place_order(
                client, old, market="ANT/WRT", side="SELL", amount=ant, price=price_ant
            )
        if lzn >= MIN_ORDER:
            await actions.place_order(
                client, old, market="LZN/WRT", side="SELL", amount=lzn, price=price_lzn
            )

        seed = BotRegistry.new_seed(self._next_index)
        self._next_index += 1
        newbie = BotWallet(seed=seed, kind=KIND_ENRICHMENT, desired_role=req.new_role)
        try:
            await actions.create_wallet(client, newbie)
        except Exception as exc:
            self._note(f"flip create: {exc}")
            return False
        wrt = int(row.get("wrt") or 0)
        send_amt = max(0, wrt - 10_000)
        if send_amt > 0:
            await actions.send_wrt(client, old, newbie.address, send_amt)
        if not await actions.verify_identity(client, newbie, req.new_role):
            newbie.kind = KIND_CITIZEN
            self.registry.add(newbie)
            self._note(f"flip verify {req.new_role} rejected (slot/gate)")
            return False

        old.kind = KIND_RETIRED
        self.registry.add(newbie)
        self.agents.pop(old.seed, None)
        self._ensure_agent(newbie)
        return True
