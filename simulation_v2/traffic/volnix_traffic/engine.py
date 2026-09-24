"""BotEngine — bootstrap wallets, weighted traffic, profit prep/verify."""

from __future__ import annotations

import logging
import random
from typing import Any, Dict, List, Optional

from volnix_traffic import actions
from volnix_traffic.client import NodeClient
from volnix_traffic.profit import (
    AccountView,
    MarketSnapshot,
    ProfitStrategy,
    account_view_from_row,
    snapshot_from_chain,
)
from volnix_traffic.registry import (
    ROLE_CITIZEN,
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    BotRegistry,
    BotWallet,
)
from volnix_traffic.settings import TrafficSettings, Weights

log = logging.getLogger("volnix_traffic.engine")

SCALE = 1_000_000
MIN_TRANSFER = 10_000
MIN_ORDER = 10_000


class BotEngine:
    def __init__(self, settings: TrafficSettings, registry: Optional[BotRegistry] = None) -> None:
        self.settings = settings
        self.registry = registry or BotRegistry()
        self.profit = ProfitStrategy()
        self.intensity = float(settings.intensity)
        self._action_carry = 0.0
        self._next_index = 0
        self.last_errors: List[str] = []

    def set_intensity(self, value: float) -> None:
        self.intensity = max(0.0, min(100.0, float(value)))

    def _note_error(self, msg: str) -> None:
        self.last_errors.append(msg)
        self.last_errors = self.last_errors[-20:]
        log.debug("%s", msg)

    async def bootstrap_one(self, client: NodeClient, snap: MarketSnapshot) -> Optional[BotWallet]:
        seed = BotRegistry.new_seed(self._next_index)
        self._next_index += 1
        bot = BotWallet(seed=seed)
        try:
            await actions.create_wallet(client, bot)
        except Exception as exc:
            self._note_error(f"create {seed}: {exc}")
            return None
        self.registry.add(bot)
        # Fund from genesis subsidy
        try:
            ok = await actions.fund_wrt(client, bot, self.settings.bootstrap_wrt)
            if not ok:
                self._note_error(f"fund {bot.address}: rejected")
        except Exception as exc:
            self._note_error(f"fund {bot.address}: {exc}")

        # Decide whether to verify now
        if random.random() < (1.0 - self.settings.citizen_fraction):
            view = AccountView(address=bot.address, role=ROLE_CITIZEN, wrt=self.settings.bootstrap_wrt)
            role = self.profit.best_first_role(view, snap)
            if role in (ROLE_SUPPLIER, ROLE_VALIDATOR):
                if await actions.verify_identity(client, bot, role):
                    bot.desired_role = role
        return bot

    async def ensure_pool(
        self,
        client: NodeClient,
        snap: MarketSnapshot,
        *,
        max_new: int = 3,
    ) -> int:
        created = 0
        while len(self.registry) < self.settings.target_wallets and created < max_new:
            bot = await self.bootstrap_one(client, snap)
            if bot is None:
                break
            created += 1
        return created

    def _pick_action(self) -> str:
        w: Weights = self.settings.weights
        names = ["transfer", "trade_ant", "trade_lzn", "activate_lzn", "cancel_order", "prep"]
        weights = [w.transfer, w.trade_ant, w.trade_lzn, w.activate_lzn, w.cancel_order, w.prep]
        return random.choices(names, weights=weights, k=1)[0]

    async def step(
        self,
        client: NodeClient,
        *,
        height: int,
        summary: dict[str, Any],
        accounts: list[dict[str, Any]],
        orderbook: dict[str, Any],
        params: dict[str, Any],
        lzn_orderbook: Optional[dict[str, Any]] = None,
    ) -> int:
        snap = snapshot_from_chain(
            summary,
            orderbook,
            lzn_orderbook=lzn_orderbook,
            lambda_num=int(params.get("lambda_num") or 1),
            lambda_den=int(params.get("lambda_den") or 3),
        )
        self.registry.sync_from_chain(accounts)
        by_addr = {a["address"]: a for a in accounts}

        created = await self.ensure_pool(client, snap)
        n = created

        self._action_carry += self.intensity
        budget = min(int(self._action_carry), self.settings.max_actions_per_tick)
        self._action_carry -= budget

        bots = self.registry.all()
        if not bots:
            return n

        lzn_book = lzn_orderbook or {}
        for _ in range(budget):
            action = self._pick_action()
            try:
                did = await self._do_action(
                    client, action, bots, by_addr, snap, orderbook, lzn_book
                )
            except Exception as exc:
                self._note_error(f"{action}: {exc}")
                did = False
            if did:
                n += 1
        return n

    async def _do_action(
        self,
        client: NodeClient,
        action: str,
        bots: list[BotWallet],
        by_addr: dict[str, dict[str, Any]],
        snap: MarketSnapshot,
        orderbook: dict[str, Any],
        lzn_orderbook: dict[str, Any],
    ) -> bool:
        if action == "transfer":
            return await self._transfer(client, bots, by_addr)
        if action == "trade_ant":
            return await self._trade_ant(client, bots, by_addr, snap, orderbook)
        if action == "trade_lzn":
            return await self._trade_lzn(client, bots, by_addr, snap, lzn_orderbook)
        if action == "activate_lzn":
            return await self._activate(client, bots, by_addr)
        if action == "cancel_order":
            return await self._cancel(client, bots, by_addr)
        if action == "prep":
            return await self._prep(client, bots, by_addr, snap, lzn_orderbook)
        return False

    async def _transfer(
        self,
        client: NodeClient,
        bots: list[BotWallet],
        by_addr: dict[str, dict[str, Any]],
    ) -> bool:
        funded = [b for b in bots if int((by_addr.get(b.address) or {}).get("wrt") or 0) > MIN_TRANSFER * 2]
        if len(funded) < 2:
            return False
        src, dst = random.sample(funded, 2)
        bal = int((by_addr.get(src.address) or {}).get("wrt") or 0)
        amount = random.randint(MIN_TRANSFER, max(MIN_TRANSFER, bal // 10))
        return await actions.send_wrt(client, src, dst.address, amount)

    async def _trade_ant(
        self,
        client: NodeClient,
        bots: list[BotWallet],
        by_addr: dict[str, dict[str, Any]],
        snap: MarketSnapshot,
        orderbook: dict[str, Any],
    ) -> bool:
        suppliers = [b for b in bots if b.role == ROLE_SUPPLIER]
        validators = [b for b in bots if b.role == ROLE_VALIDATOR]
        if random.random() < 0.55 and suppliers:
            bot = random.choice(suppliers)
            ant = int((by_addr.get(bot.address) or {}).get("ant") or 0)
            if ant < MIN_ORDER:
                return False
            asks = orderbook.get("asks") or []
            price = int(asks[0]["price"]) if asks else snap.ant_price
            # Near epoch end, shade ask down (wipe pressure)
            lot = max(MIN_ORDER, int(ant * 0.15))
            return await actions.place_order(
                client, bot, market="ANT/WRT", side="SELL", amount=min(lot, ant), price=max(1, price)
            )
        if validators:
            bot = random.choice(validators)
            wrt = int((by_addr.get(bot.address) or {}).get("wrt") or 0)
            if wrt < snap.ant_price * MIN_ORDER:
                return False
            # Bid under reservation (~10% weekly margin anchor)
            price = max(1, int(snap.ant_price * 0.95))
            lot = min(MIN_ORDER * 5, wrt // price)
            if lot < MIN_ORDER:
                return False
            return await actions.place_order(
                client, bot, market="ANT/WRT", side="BUY", amount=lot, price=price
            )
        return False

    async def _trade_lzn(
        self,
        client: NodeClient,
        bots: list[BotWallet],
        by_addr: dict[str, dict[str, Any]],
        snap: MarketSnapshot,
        lzn_orderbook: dict[str, Any],
    ) -> bool:
        sellers = [
            b
            for b in bots
            if int((by_addr.get(b.address) or {}).get("lzn") or 0) >= MIN_ORDER
        ]
        buyers = [b for b in bots if b.role in (ROLE_SUPPLIER, ROLE_VALIDATOR)]
        asks = lzn_orderbook.get("asks") or []
        bids = lzn_orderbook.get("bids") or []
        best_ask = int(asks[0]["price"]) if asks else 0
        best_bid = int(bids[0]["price"]) if bids else 0
        quote = max(1, snap.lzn_price)

        # Prefer BUY when any buyer has zero LZN (seed demand before float exists).
        hungry = [
            b
            for b in buyers
            if int((by_addr.get(b.address) or {}).get("lzn") or 0)
            + int((by_addr.get(b.address) or {}).get("lzn_activated") or 0)
            == 0
        ]
        want_buy = bool(buyers) and (bool(hungry) or not sellers or random.random() < 0.55)
        if want_buy and buyers:
            pool = hungry or buyers
            bot = random.choice(pool)
            wrt = int((by_addr.get(bot.address) or {}).get("wrt") or 0)
            # Shade bid under ask / quote
            if best_ask > 0:
                price = max(1, int(best_ask * 0.95))
            else:
                price = max(1, int(quote * 0.95))
            lot = min(SCALE // 10, wrt // price if price > 0 else 0)
            if lot < MIN_ORDER:
                # Fall through to sell if possible
                want_buy = False
            else:
                return await actions.place_order(
                    client, bot, market="LZN/WRT", side="BUY", amount=lot, price=price
                )

        if sellers:
            bot = random.choice(sellers)
            free = int((by_addr.get(bot.address) or {}).get("lzn") or 0)
            lot = max(MIN_ORDER, free // 5)
            if best_bid > 0:
                price = max(1, int(best_bid * 1.05))
            else:
                price = max(1, int(quote * 1.05))
            return await actions.place_order(
                client, bot, market="LZN/WRT", side="SELL", amount=min(lot, free), price=price
            )
        return False

    async def _activate(
        self,
        client: NodeClient,
        bots: list[BotWallet],
        by_addr: dict[str, dict[str, Any]],
    ) -> bool:
        cands = [
            b
            for b in bots
            if b.role == ROLE_VALIDATOR
            and int((by_addr.get(b.address) or {}).get("lzn") or 0) >= MIN_ORDER
        ]
        if not cands:
            return False
        bot = random.choice(cands)
        free = int((by_addr.get(bot.address) or {}).get("lzn") or 0)
        amount = max(MIN_ORDER, free // 2)
        return await actions.activate_lzn(client, bot, min(amount, free))

    async def _cancel(
        self,
        client: NodeClient,
        bots: list[BotWallet],
        by_addr: dict[str, dict[str, Any]],
    ) -> bool:
        # Need account detail with open orders — fetch one bot with open_orders > 0
        cands = [b for b in bots if int((by_addr.get(b.address) or {}).get("open_orders") or 0) > 0]
        if not cands:
            return False
        bot = random.choice(cands)
        detail = await client.account(bot.address)
        orders = detail.get("open_orders") or []
        if not orders:
            return False
        oid = str(orders[0].get("order_id") or "")
        return await actions.cancel_order(client, bot, oid)

    async def _prep(
        self,
        client: NodeClient,
        bots: list[BotWallet],
        by_addr: dict[str, dict[str, Any]],
        snap: MarketSnapshot,
        lzn_orderbook: Optional[dict[str, Any]] = None,
    ) -> bool:
        # Unverified citizens → ROI role
        citizens = [b for b in bots if b.role == ROLE_CITIZEN and not b.verified]
        if citizens:
            bot = random.choice(citizens)
            row = by_addr.get(bot.address) or {"address": bot.address, "role": ROLE_CITIZEN}
            view = account_view_from_row(row)
            role = self.profit.desired_role(view, snap)
            if role in (ROLE_SUPPLIER, ROLE_VALIDATOR):
                # Ensure some WRT first
                wrt = int(row.get("wrt") or 0)
                if wrt < self.settings.min_bot_wrt:
                    await actions.fund_wrt(client, bot, self.settings.bootstrap_wrt)
                return await actions.verify_identity(client, bot, role)
            return False

        # Validators with zero LZN → bid LZN/WRT before activate
        price = max(1, snap.lzn_price)
        asks = (lzn_orderbook or {}).get("asks") or []
        if asks:
            price = max(1, int(asks[0]["price"]))
        for bot in bots:
            if bot.role != ROLE_VALIDATOR:
                continue
            row = by_addr.get(bot.address) or {}
            free = int(row.get("lzn") or 0)
            activated = int(row.get("lzn_activated") or 0)
            if free + activated > 0:
                continue
            wrt = int(row.get("wrt") or 0)
            bid = max(1, int(price * 0.95))
            if wrt < bid * MIN_ORDER:
                continue
            lot = min(SCALE // 10, wrt // bid)
            if lot < MIN_ORDER:
                continue
            return await actions.place_order(
                client, bot, market="LZN/WRT", side="BUY", amount=lot, price=bid
            )

        # Validators with free LZN → activate
        for bot in bots:
            if bot.role != ROLE_VALIDATOR:
                continue
            free = int((by_addr.get(bot.address) or {}).get("lzn") or 0)
            if free >= MIN_ORDER:
                return await actions.activate_lzn(client, bot, free // 2)
        # Top-up low WRT bots
        for bot in bots:
            wrt = int((by_addr.get(bot.address) or {}).get("wrt") or 0)
            if wrt < self.settings.min_bot_wrt:
                return await actions.fund_wrt(client, bot, self.settings.bootstrap_wrt)
        return False
