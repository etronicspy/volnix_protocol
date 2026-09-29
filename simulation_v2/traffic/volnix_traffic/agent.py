"""One enrichment wallet — own strategy state, own P&L, own txs."""

from __future__ import annotations

import random
from dataclasses import dataclass, replace
from fractions import Fraction
from typing import Any, Dict, Optional, Sequence

from volnix_traffic import actions
from volnix_traffic.client import NodeClient
from volnix_traffic.compete import DeclarePick, Rival, pick_declare
from volnix_traffic.enrichment import (
    HorizonForecast,
    card_score,
    forecast_roles,
    sample_peer_indices,
)
from volnix_traffic.profit import (
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    AccountView,
    MarketSnapshot,
    account_view_from_row,
)
from volnix_traffic.registry import BotWallet

MIN_ORDER = 10_000
SCALE = 1_000_000
FEE_PER_TX = 0.02 * SCALE
AVG_TXS_PER_BLOCK = 4.0
# Canon §4.1: ⌊LZN_TOTAL_SUPPLY / 3⌋ in whole LZN, then × SCALE micro-units.
LZN_TOTAL_SUPPLY = 1_000_000_000
LZN_MAX_FROZEN = (LZN_TOTAL_SUPPLY // 3) * SCALE
ANT_BUFFER_BLOCKS = 16
WRT_FEE_RESERVE = int(FEE_PER_TX * 4)


def _side_orders(orderbook: Optional[Dict[str, Any]], address: str, side: str) -> list[Dict[str, Any]]:
    if not orderbook or not address:
        return []
    key = "bids" if side == "BUY" else "asks"
    out: list[Dict[str, Any]] = []
    for raw in orderbook.get(key) or []:
        if raw.get("owner") != address:
            continue
        if int(raw.get("remaining") or 0) <= 0:
            continue
        out.append(raw)
    return out


@dataclass
class StrategyProfile:
    b_frac: float = 0.50
    s_frac: float = 0.40
    activate_ratio: float = 1.0
    ask_shade: float = 1.00
    bid_shade: float = 0.95
    desired_role: str = ""

    def copy(self) -> "StrategyProfile":
        return replace(self)


@dataclass(frozen=True)
class StrategyCard:
    address: str
    role: str
    b_frac: float
    s_frac: float
    activate_ratio: float
    ask_shade: float
    bid_shade: float
    desired_role: str
    growth_wrt: int
    expected_wrt: int

    def score(self) -> tuple[int, int]:
        return card_score(self.growth_wrt, self.expected_wrt)


@dataclass
class FlipRequest:
    agent: "EnrichmentAgent"
    new_role: str


@dataclass
class AgentTickResult:
    declared: bool = False
    market: bool = False
    adopted_from: str = ""
    flip: Optional[FlipRequest] = None
    pick: Optional[DeclarePick] = None
    expected_wrt: int = 0
    error: str = ""


def balanced_profile() -> StrategyProfile:
    return StrategyProfile()


class EnrichmentAgent:
    """Independent strategy process for one supplier/validator wallet."""

    def __init__(
        self,
        wallet: BotWallet,
        *,
        horizon: int = 12,
        peer_sample_min: int = 5,
        peer_sample_max: int = 10,
        role_flip_margin_wrt: int = 5_000_000,
        flip_confirm_ticks: int = 3,
        rng: Optional[random.Random] = None,
    ) -> None:
        self.wallet = wallet
        self.horizon = int(horizon)
        self.peer_sample_min = int(peer_sample_min)
        self.peer_sample_max = int(peer_sample_max)
        self.role_flip_margin_wrt = int(role_flip_margin_wrt)
        self.flip_confirm_ticks = int(flip_confirm_ticks)
        self.rng = rng or random.Random()
        self.strategy = balanced_profile()
        if wallet.role in (ROLE_SUPPLIER, ROLE_VALIDATOR):
            self.strategy.desired_role = wallet.role
        self.wrt_mark: int = 0
        self.growth_wrt: int = 0
        self.expected_wrt: int = 0
        self.flip_streak: int = 0
        self.flip_target: str = ""
        self.last_pick: Optional[DeclarePick] = None
        self.adopted_from: str = ""
        self._mark_height: int = -1

    def card(self) -> StrategyCard:
        return StrategyCard(
            address=self.wallet.address,
            role=self.wallet.role,
            b_frac=self.strategy.b_frac,
            s_frac=self.strategy.s_frac,
            activate_ratio=self.strategy.activate_ratio,
            ask_shade=self.strategy.ask_shade,
            bid_shade=self.strategy.bid_shade,
            desired_role=self.strategy.desired_role,
            growth_wrt=self.growth_wrt,
            expected_wrt=self.expected_wrt,
        )

    def scout(self, board: Sequence[StrategyCard]) -> Optional[StrategyCard]:
        others = [c for c in board if c.address and c.address != self.wallet.address]
        if not others:
            return None
        idxs = sample_peer_indices(
            len(others), self.peer_sample_min, self.peer_sample_max, self.rng
        )
        if not idxs:
            return None
        sample = [others[i] for i in idxs]
        best = max(sample, key=lambda c: c.score())
        mine = self.card().score()
        if best.score() > mine:
            return best
        return None

    def adopt(self, card: StrategyCard) -> None:
        self.strategy.b_frac = card.b_frac
        self.strategy.s_frac = card.s_frac
        # Capacity race: always max-activate; peer cards cannot teach deactivation.
        self.strategy.activate_ratio = 1.0
        self.strategy.ask_shade = card.ask_shade
        self.strategy.bid_shade = card.bid_shade
        if card.desired_role in (ROLE_SUPPLIER, ROLE_VALIDATOR):
            self.strategy.desired_role = card.desired_role
        self.adopted_from = card.address

    def _update_growth(self, wrt: int, height: int) -> None:
        if self._mark_height < 0:
            self.wrt_mark = wrt
            self._mark_height = height
            self.growth_wrt = 0
            return
        self.growth_wrt = wrt - self.wrt_mark
        if height - self._mark_height >= self.horizon:
            self.wrt_mark = wrt
            self._mark_height = height

    async def tick(
        self,
        client: NodeClient,
        *,
        height: int,
        snap: MarketSnapshot,
        row: Dict[str, Any],
        rivals: Sequence[Rival],
        board: Sequence[StrategyCard],
        alpha: Fraction,
        lam: Fraction,
        k: int,
        n_suppliers: int,
        n_validators: int,
        min_suppliers: int,
        min_validators: int,
        enable_market: bool,
        enable_declare: bool,
        enable_role_flip: bool,
        ant_orderbook: Optional[Dict[str, Any]] = None,
        lzn_orderbook: Optional[Dict[str, Any]] = None,
    ) -> AgentTickResult:
        out = AgentTickResult()
        view = account_view_from_row(row)
        self._update_growth(view.wrt, height)
        self.adopted_from = ""

        peer = self.scout(board)
        if peer is not None:
            self.adopt(peer)
            out.adopted_from = peer.address

        forecasts = forecast_roles(
            view,
            snap,
            rivals,
            horizon=self.horizon,
            alpha=alpha,
            lam=lam,
            k=k,
            b_frac=self.strategy.b_frac,
            s_frac=self.strategy.s_frac,
        )
        current = forecasts.get(self.wallet.role) or HorizonForecast(
            role=self.wallet.role, income_wrt=0, cost_wrt=0, risk_wrt=0
        )
        self.expected_wrt = current.net
        out.expected_wrt = current.net

        alt_role = ROLE_SUPPLIER if self.wallet.role == ROLE_VALIDATOR else ROLE_VALIDATOR
        alt = forecasts[alt_role]
        if alt.net > current.net + self.role_flip_margin_wrt:
            if self.flip_target == alt_role:
                self.flip_streak += 1
            else:
                self.flip_target = alt_role
                self.flip_streak = 1
        else:
            self.flip_streak = 0
            self.flip_target = ""

        if (
            enable_role_flip
            and not self.wallet.genesis
            and self.flip_streak >= self.flip_confirm_ticks
            and self._floor_ok(alt_role, n_suppliers, n_validators, min_suppliers, min_validators)
        ):
            out.flip = FlipRequest(agent=self, new_role=alt_role)
            return out

        try:
            if enable_market:
                out.market = await self._maybe_market(
                    client,
                    view,
                    snap,
                    row,
                    ant_orderbook=ant_orderbook,
                    lzn_orderbook=lzn_orderbook,
                )
            if self.wallet.role == ROLE_VALIDATOR and enable_declare:
                pick = pick_declare(
                    address=self.wallet.address,
                    l_i=view.lzn_activated,
                    ant=view.ant,
                    rivals=rivals,
                    alpha=alpha,
                    lam=lam,
                    k=k,
                    subsidy=snap.block_reward,
                    fees=int(FEE_PER_TX * AVG_TXS_PER_BLOCK),
                    ant_price=max(1, snap.ant_price),
                    b_frac=self.strategy.b_frac,
                    s_frac=self.strategy.s_frac,
                )
                self.last_pick = pick
                out.pick = pick
                if pick is not None:
                    out.declared = await actions.declare(client, self.wallet, pick.b_i, pick.s_i)
        except Exception as exc:
            out.error = str(exc)
        return out

    def _floor_ok(
        self,
        new_role: str,
        n_suppliers: int,
        n_validators: int,
        min_suppliers: int,
        min_validators: int,
    ) -> bool:
        if self.wallet.role == ROLE_SUPPLIER and n_suppliers <= min_suppliers:
            return False
        if self.wallet.role == ROLE_VALIDATOR and n_validators <= min_validators:
            return False
        return new_role in (ROLE_SUPPLIER, ROLE_VALIDATOR)

    async def _maybe_market(
        self,
        client: NodeClient,
        view: AccountView,
        snap: MarketSnapshot,
        row: Dict[str, Any],
        *,
        ant_orderbook: Optional[Dict[str, Any]] = None,
        lzn_orderbook: Optional[Dict[str, Any]] = None,
    ) -> bool:
        del row  # freeze/deactivate no longer consulted
        if self.wallet.role == ROLE_VALIDATOR:
            return await self._validator_market(
                client,
                view,
                snap,
                ant_orderbook=ant_orderbook,
                lzn_orderbook=lzn_orderbook,
            )

        if self.wallet.role == ROLE_SUPPLIER and view.ant >= MIN_ORDER:
            owned = _side_orders(ant_orderbook, self.wallet.address, "SELL")
            lot = max(MIN_ORDER, int(view.ant * 0.10))
            price = max(1, int(snap.ant_price * self.strategy.ask_shade))
            return await self._merge_place(
                client,
                market="ANT/WRT",
                side="SELL",
                amount=min(lot, view.ant),
                price=price,
                existing=owned,
            )
        if self.wallet.role == ROLE_SUPPLIER:
            owned = _side_orders(ant_orderbook, self.wallet.address, "SELL")
            if len(owned) > 1:
                price = max(1, int(snap.ant_price * self.strategy.ask_shade))
                return await self._merge_place(
                    client,
                    market="ANT/WRT",
                    side="SELL",
                    amount=0,
                    price=price,
                    existing=owned,
                )
        return False

    async def _merge_place(
        self,
        client: NodeClient,
        *,
        market: str,
        side: str,
        amount: int,
        price: int,
        existing: Sequence[Dict[str, Any]],
        quote_free: int = 0,
    ) -> bool:
        """Cancel this wallet's open orders on the same side, then post one combined order."""
        extra = max(0, int(amount))
        if extra < MIN_ORDER and len(existing) <= 1:
            return False
        released = 0
        quote_back = 0
        for raw in existing:
            oid = str(raw.get("order_id") or "")
            rem = int(raw.get("remaining") or 0)
            old_px = int(raw.get("price") or 0)
            if not oid or rem <= 0:
                continue
            if await actions.cancel_order(client, self.wallet, oid):
                released += rem
                quote_back += rem * old_px
        total = extra + released
        if side == "BUY" and price > 0:
            total = min(total, (max(0, quote_free) + quote_back) // price)
        if total < MIN_ORDER:
            return released > 0
        return await actions.place_order(
            client,
            self.wallet,
            market=market,
            side=side,
            amount=total,
            price=price,
        )

    async def _validator_market(
        self,
        client: NodeClient,
        view: AccountView,
        snap: MarketSnapshot,
        *,
        ant_orderbook: Optional[Dict[str, Any]] = None,
        lzn_orderbook: Optional[Dict[str, Any]] = None,
    ) -> bool:
        """Activate max → ANT buffer → BUY LZN (capacity race vs L_total) in one tick."""
        did = False
        free = int(view.lzn)
        activated = int(view.lzn_activated)
        ant = int(view.ant)
        wrt = int(view.wrt)

        room = max(0, LZN_MAX_FROZEN - activated)
        # 1. Activate all free LZN up to per-address ceiling.
        if free >= MIN_ORDER and room >= MIN_ORDER:
            amt = min(free, room)
            if amt >= MIN_ORDER and await actions.activate_lzn(client, self.wallet, amt):
                did = True
                free -= amt
                activated += amt
                room = max(0, LZN_MAX_FROZEN - activated)

        L_i = max(activated, 1)
        ant_target = max(MIN_ORDER, int(snap.lambda_f * L_i * ANT_BUFFER_BLOCKS))
        ant_price = max(1, int(snap.ant_price * self.strategy.bid_shade))
        lzn_price = max(1, int(snap.lzn_price * self.strategy.bid_shade))
        ant_buys = _side_orders(ant_orderbook, self.wallet.address, "BUY")
        ant_open = sum(int(o.get("remaining") or 0) for o in ant_buys)
        held = ant + ant_open

        # 2. Refill ANT when below declare fuel buffer. Resting bids count toward the buffer.
        lot = 0
        spare_wrt = wrt - WRT_FEE_RESERVE
        if held < ant_target and spare_wrt > ant_price * MIN_ORDER:
            want = ant_target - held
            affordable = spare_wrt // ant_price
            lot = min(want, affordable, MIN_ORDER * 5)
        if lot >= MIN_ORDER or len(ant_buys) > 1:
            extra = lot if lot >= MIN_ORDER else 0
            if await self._merge_place(
                client,
                market="ANT/WRT",
                side="BUY",
                amount=extra,
                price=ant_price,
                existing=ant_buys,
                quote_free=spare_wrt,
            ):
                did = True
                ant += extra
                wrt -= extra * ant_price

        # 3. LZN demand sized by network mass (fair share of l_total) under freeze cap.
        lzn_buys = _side_orders(lzn_orderbook, self.wallet.address, "BUY")
        lzn_open = sum(int(o.get("remaining") or 0) for o in lzn_buys)
        room_left = room - lzn_open
        lzn_lot = 0
        lzn_spare = 0
        if room_left >= MIN_ORDER:
            lzn_spare = wrt - WRT_FEE_RESERVE
            ant_reserve = max(0, ant_target - ant) * ant_price
            lzn_spare -= ant_reserve
            n_v = max(1, int(snap.n_validators))
            fair_share = max(MIN_ORDER, int(snap.l_total) // n_v)
            if lzn_spare >= lzn_price * MIN_ORDER and lzn_open < fair_share:
                lzn_lot = min(room_left, lzn_spare // lzn_price, fair_share - lzn_open)
        if lzn_lot >= MIN_ORDER or len(lzn_buys) > 1:
            extra = lzn_lot if lzn_lot >= MIN_ORDER else 0
            if await self._merge_place(
                client,
                market="LZN/WRT",
                side="BUY",
                amount=extra,
                price=lzn_price,
                existing=lzn_buys,
                quote_free=max(0, lzn_spare),
            ):
                did = True
        return did
