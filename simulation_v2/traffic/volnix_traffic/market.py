"""Reference AutoMarket — keep ANT book alive for PoVB fuel; LZN asks when float exists."""

from __future__ import annotations

import logging
import math
from typing import Any, Optional

from volnix_traffic import actions
from volnix_traffic.client import NodeClient
from volnix_traffic.registry import BotRegistry

log = logging.getLogger("volnix_traffic.market")

ANT_BUFFER_BLOCKS = 64
REFILL_TRIGGER = 0.75
ASK_STEP = 0.04
REVIEW_INTERVAL_BLOCKS = 32
REQUOTE_THRESHOLD = 0.03
PROVIDER_LOT_FRACTION = 0.10
MIN_LOT = 10_000  # 0.01 ANT/LZN in micro
SCALE = 1_000_000


def floor_price(value: float) -> int:
    return max(1, int(math.floor(float(value))))


def miner_reservation_price(
    *,
    l_i: int,
    l_active: int,
    block_reward: int,
    lambda_f: float,
) -> int:
    if l_active <= 0 or lambda_f <= 0:
        return 1
    return max(1, int(block_reward / (lambda_f * l_active)))


def _open_by_owner(orderbook: dict[str, Any]) -> dict[str, list[dict[str, Any]]]:
    asks = orderbook.get("asks") or []
    bids = orderbook.get("bids") or []
    out: dict[str, list[dict[str, Any]]] = {}
    for side_list, side in ((asks, "SELL"), (bids, "BUY")):
        for o in side_list:
            out.setdefault(o["owner"], []).append({**o, "side": side})
    return out


class AutoMarket:
    """Supplier ANT asks + validator ANT refill + free-LZN asks for traffic wallets."""

    def __init__(self) -> None:
        # quote_key -> (reviewed_at, wrt_at_review, ask_price)
        # ANT keys: address; LZN keys: f"{address}|lzn"
        self._quotes: dict[str, tuple[int, int, int]] = {}

    def reset(self) -> None:
        self._quotes.clear()

    def provider_ask(
        self,
        address: str,
        *,
        height: int,
        wrt: int,
        bootstrap: int,
        sold_signal: bool,
        demand: float,
    ) -> int:
        prev = self._quotes.get(address)
        if prev is None or height < prev[0]:
            ask = floor_price(bootstrap)
            self._quotes[address] = (height, wrt, ask)
            return ask
        reviewed_at, _wrt_at, ask = prev
        if height - reviewed_at < REVIEW_INTERVAL_BLOCKS:
            return ask
        if sold_signal:
            adj = ASK_STEP * (1.0 + demand)
        else:
            adj = -ASK_STEP
        ask = floor_price(ask * (1.0 + adj))
        self._quotes[address] = (height, wrt, ask)
        return ask

    async def step(
        self,
        client: NodeClient,
        registry: BotRegistry,
        *,
        height: int,
        accounts_by_addr: dict[str, dict[str, Any]],
        orderbook: dict[str, Any],
        block_reward: int,
        lambda_f: float,
        l_total: int,
        lzn_orderbook: Optional[dict[str, Any]] = None,
        lzn_price: Optional[int] = None,
    ) -> int:
        submitted = 0
        if l_total <= 0:
            return 0
        bootstrap = max(1, int(block_reward / (lambda_f * l_total)))
        asks = orderbook.get("asks") or []
        bids = orderbook.get("bids") or []
        bid_vol = sum(int(b.get("remaining") or 0) for b in bids)
        ask_vol = sum(int(a.get("remaining") or 0) for a in asks)
        demand = bid_vol / (bid_vol + ask_vol) if (bid_vol + ask_vol) else 0.0
        best_ask = int(asks[0]["price"]) if asks else bootstrap

        open_by_owner = _open_by_owner(orderbook)

        # Suppliers sell ANT
        for bot in registry.suppliers():
            row = accounts_by_addr.get(bot.address)
            if not row:
                continue
            ant = int(row.get("ant") or 0)
            if ant < MIN_LOT:
                continue
            wrt = int(row.get("wrt") or 0)
            prev = self._quotes.get(bot.address)
            sold = bool(prev and wrt > prev[1])
            ask = self.provider_ask(
                bot.address,
                height=height,
                wrt=wrt,
                bootstrap=bootstrap,
                sold_signal=sold,
                demand=demand,
            )
            owned = open_by_owner.get(bot.address) or []
            stale = [
                o
                for o in owned
                if o.get("side") == "SELL" and abs(int(o["price"]) - ask) > ask * REQUOTE_THRESHOLD
            ]
            for o in stale:
                if await actions.cancel_order(client, bot, str(o["order_id"])):
                    submitted += 1
            sell_open = sum(int(o.get("remaining") or 0) for o in owned if o.get("side") == "SELL")
            if sell_open > 0 and not stale:
                continue
            lot = max(MIN_LOT, int(ant * PROVIDER_LOT_FRACTION))
            lot = min(lot, ant)
            if await actions.place_order(
                client, bot, market="ANT/WRT", side="SELL", amount=lot, price=ask
            ):
                submitted += 1

        # Validators refill ANT buffer
        validators = [
            b
            for b in registry.validators()
            if (accounts_by_addr.get(b.address) or {}).get("lzn_activated", 0)
        ]
        l_active = sum(
            int((accounts_by_addr.get(b.address) or {}).get("lzn_activated") or 0)
            for b in validators
        )
        for bot in validators:
            row = accounts_by_addr.get(bot.address) or {}
            l_i = int(row.get("lzn_activated") or 0)
            if l_i <= 0:
                continue
            ant = int(row.get("ant") or 0)
            wrt = int(row.get("wrt") or 0)
            owned = open_by_owner.get(bot.address) or []
            buy_open = sum(int(o.get("remaining") or 0) for o in owned if o.get("side") == "BUY")
            target = int(lambda_f * l_i * ANT_BUFFER_BLOCKS)
            held = ant + buy_open
            if held >= target * REFILL_TRIGGER:
                continue
            reservation = miner_reservation_price(
                l_i=l_i, l_active=max(l_i, l_active), block_reward=block_reward, lambda_f=lambda_f
            )
            bid = floor_price(min(best_ask, reservation))
            want = target - held
            affordable = wrt // bid if bid > 0 else 0
            lot = min(want, affordable)
            if lot < MIN_LOT:
                continue
            if await actions.place_order(
                client, bot, market="ANT/WRT", side="BUY", amount=lot, price=bid
            ):
                submitted += 1

        # Free-LZN holders post LZN/WRT asks (no LZN BUY market-maker)
        submitted += await self._lzn_asks(
            client,
            registry,
            height=height,
            accounts_by_addr=accounts_by_addr,
            lzn_orderbook=lzn_orderbook or {},
            bootstrap=max(1, int(lzn_price) if lzn_price else bootstrap),
        )
        return submitted

    async def _lzn_asks(
        self,
        client: NodeClient,
        registry: BotRegistry,
        *,
        height: int,
        accounts_by_addr: dict[str, dict[str, Any]],
        lzn_orderbook: dict[str, Any],
        bootstrap: int,
    ) -> int:
        submitted = 0
        asks = lzn_orderbook.get("asks") or []
        bids = lzn_orderbook.get("bids") or []
        bid_vol = sum(int(b.get("remaining") or 0) for b in bids)
        ask_vol = sum(int(a.get("remaining") or 0) for a in asks)
        demand = bid_vol / (bid_vol + ask_vol) if (bid_vol + ask_vol) else 0.0
        open_by_owner = _open_by_owner(lzn_orderbook)

        for bot in registry.all():
            row = accounts_by_addr.get(bot.address)
            if not row:
                continue
            free = int(row.get("lzn") or 0)
            if free < MIN_LOT:
                continue
            wrt = int(row.get("wrt") or 0)
            key = f"{bot.address}|lzn"
            prev = self._quotes.get(key)
            sold = bool(prev and wrt > prev[1])
            ask = self.provider_ask(
                key,
                height=height,
                wrt=wrt,
                bootstrap=bootstrap,
                sold_signal=sold,
                demand=demand,
            )
            owned = open_by_owner.get(bot.address) or []
            stale = [
                o
                for o in owned
                if o.get("side") == "SELL" and abs(int(o["price"]) - ask) > ask * REQUOTE_THRESHOLD
            ]
            for o in stale:
                if await actions.cancel_order(client, bot, str(o["order_id"])):
                    submitted += 1
            sell_open = sum(int(o.get("remaining") or 0) for o in owned if o.get("side") == "SELL")
            if sell_open > 0 and not stale:
                continue
            lot = max(MIN_LOT, int(free * PROVIDER_LOT_FRACTION))
            lot = min(lot, free)
            if await actions.place_order(
                client, bot, market="LZN/WRT", side="SELL", amount=lot, price=ask
            ):
                submitted += 1
        return submitted
