"""Citizen bootstrap + spontaneous WRT spend (all traffic wallets)."""

from __future__ import annotations

import logging
import random
from typing import Any, Dict, List, Optional

from volnix_traffic import actions
from volnix_traffic.client import NodeClient
from volnix_traffic.registry import KIND_CITIZEN, BotRegistry, BotWallet
from volnix_traffic.settings import TrafficSettings

log = logging.getLogger("volnix_traffic.citizens")

MIN_TRANSFER = 10_000
TRANSFERS_FLOOR_MIN = 2
TRANSFERS_FLOOR_MAX = 4


def spend_probability(intensity: float) -> float:
    """Share of funded wallets that spend this height. Intensity 0 → none; 10 → all."""
    if intensity <= 0:
        return 0.0
    return min(1.0, 0.1 * float(intensity))


def transfers_floor(citizen_transfers_per_tick: int) -> int:
    """MsgSend floor per height when intensity > 0 (README clamp 2–4)."""
    return max(TRANSFERS_FLOOR_MIN, min(TRANSFERS_FLOOR_MAX, int(citizen_transfers_per_tick)))


class CitizenTraffic:
    def __init__(self, settings: TrafficSettings, registry: BotRegistry) -> None:
        self.settings = settings
        self.registry = registry
        self._next_index = 0

    def sync_index(self, hint: int) -> None:
        self._next_index = max(self._next_index, int(hint))

    async def ensure_pool(self, client: NodeClient, *, max_new: int) -> int:
        created = 0
        while len(self.registry.citizens()) < self.settings.target_citizens and created < max_new:
            seed = BotRegistry.new_seed(self._next_index)
            self._next_index += 1
            bot = BotWallet(seed=seed, kind=KIND_CITIZEN)
            try:
                await actions.create_wallet(client, bot)
            except Exception as exc:
                log.debug("create citizen %s: %s", seed, exc)
                break
            self.registry.add(bot)
            try:
                await actions.fund_wrt(client, bot, self.settings.bootstrap_wrt)
            except Exception as exc:
                log.debug("fund citizen %s: %s", bot.address, exc)
            created += 1
        return created

    def _eligible(
        self, accounts_by_addr: Dict[str, Dict[str, Any]]
    ) -> List[BotWallet]:
        """Every registered wallet with spare WRT — citizens, bots, genesis."""
        out: List[BotWallet] = []
        for b in self.registry.all():
            if not b.address:
                continue
            bal = int((accounts_by_addr.get(b.address) or {}).get("wrt") or 0)
            if bal > MIN_TRANSFER * 2:
                out.append(b)
        return out

    async def step(
        self,
        client: NodeClient,
        accounts_by_addr: Dict[str, Dict[str, Any]],
        *,
        n_transfers: Optional[int] = None,
        intensity: Optional[float] = None,
        rng: Optional[random.Random] = None,
    ) -> int:
        """Imitate everyday WRT spending: any wallet may MsgSend a slice of its balance."""
        rng = rng or random.Random()
        wallets = self._eligible(accounts_by_addr)
        if len(wallets) < 2:
            return 0

        if n_transfers is not None:
            picks = max(0, int(n_transfers))
            return await self._send_pairs(client, accounts_by_addr, wallets, picks, rng)

        level = float(self.settings.intensity if intensity is None else intensity)
        p = spend_probability(level)
        if p <= 0:
            return 0
        spenders = [b for b in wallets if rng.random() < p]
        # README: citizen_transfers_per_tick (clamped 2–4) is the floor when intensity > 0.
        floor = min(transfers_floor(self.settings.citizen_transfers_per_tick), len(wallets))
        if len(spenders) < floor:
            remaining = [b for b in wallets if b not in spenders]
            need = floor - len(spenders)
            if remaining:
                spenders.extend(rng.sample(remaining, min(need, len(remaining))))
        cap = max(floor, int(self.settings.max_actions_per_tick))
        if len(spenders) > cap:
            rng.shuffle(spenders)
            spenders = spenders[:cap]
        sent = 0
        for src in spenders:
            others = [b for b in wallets if b.address != src.address]
            if not others:
                continue
            dst = rng.choice(others)
            bal = int((accounts_by_addr.get(src.address) or {}).get("wrt") or 0)
            amount = rng.randint(MIN_TRANSFER, max(MIN_TRANSFER, bal // 10))
            if await actions.send_wrt(client, src, dst.address, amount):
                sent += 1
                row = accounts_by_addr.get(src.address)
                if row is not None:
                    row["wrt"] = bal - amount
        return sent

    async def _send_pairs(
        self,
        client: NodeClient,
        accounts_by_addr: Dict[str, Dict[str, Any]],
        wallets: List[BotWallet],
        n: int,
        rng: random.Random,
    ) -> int:
        sent = 0
        for _ in range(n):
            src, dst = rng.sample(wallets, 2)
            bal = int((accounts_by_addr.get(src.address) or {}).get("wrt") or 0)
            if bal <= MIN_TRANSFER * 2:
                continue
            amount = rng.randint(MIN_TRANSFER, max(MIN_TRANSFER, bal // 10))
            if await actions.send_wrt(client, src, dst.address, amount):
                sent += 1
                row = accounts_by_addr.get(src.address)
                if row is not None:
                    row["wrt"] = bal - amount
        return sent
