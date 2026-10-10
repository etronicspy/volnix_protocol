"""One enrichment wallet — own strategy state, own P&L, own txs."""

from __future__ import annotations

import random
from dataclasses import dataclass, replace
from fractions import Fraction
from typing import Any, Dict, Optional, Sequence

from volnix_traffic import actions
from volnix_traffic.client import NodeClient
from volnix_traffic.compete import DeclarePick, Rival, entry_burn, pick_declare
from volnix_traffic.enrichment import (
    HorizonForecast,
    card_score,
    forecast_roles,
    forecast_validator,
    sample_peer_indices,
    wealth_wrt,
)
from volnix_traffic.market import miner_reservation_price
from volnix_traffic.profit import (
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    AccountView,
    MarketSnapshot,
    account_view_from_row,
    effective_ant_price,
    qty_affordable,
    quote_ant_ref,
    quote_cost,
    quote_lzn_ref,
)
from volnix_traffic.registry import BotWallet

MIN_ORDER = 10_000
SCALE = 1_000_000
FEE_PER_TX = 0.02 * SCALE
AVG_TXS_PER_BLOCK = 4.0
# §4.1: per-address activated ≤ ⌊new_L_total/3⌋ (sole/first activator uncapped).
_ACTIVATION_UNLIMITED = 1 << 62
ANT_BUFFER_BLOCKS = 16
WRT_FEE_RESERVE = int(FEE_PER_TX * 4)
# Supplier ask inventory pressure (unsold → markdown). Faster than legacy AutoMarket 32.
ASK_STEP = 0.04
REVIEW_INTERVAL_BLOCKS = 8
REQUOTE_THRESHOLD = 0.03


def _activation_headroom(activated: int, l_total: int) -> int:
    """Additional micro-LZN activate-able under ⌊L_total/3⌋ (mirrors node lizenz)."""
    activated = int(activated)
    l_total = int(l_total)
    if l_total == 0 and activated == 0:
        return _ACTIVATION_UNLIMITED
    if activated > 0 and l_total == activated:
        return _ACTIVATION_UNLIMITED
    t0 = max(0, l_total - activated)
    best = 0
    for x in range(0, t0 // 2 + 2):
        if x <= (t0 + x) // 3:
            best = x
    return max(0, best - activated)


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


def _best_ask(orderbook: Optional[Dict[str, Any]]) -> int:
    asks = (orderbook or {}).get("asks") or []
    if not asks:
        return 0
    return max(1, int(asks[0].get("price") or 0))


def _best_bid(orderbook: Optional[Dict[str, Any]]) -> int:
    bids = (orderbook or {}).get("bids") or []
    if not bids:
        return 0
    return max(1, int(bids[0].get("price") or 0))


def _ant_reservation(snap: MarketSnapshot, l_i: int) -> int:
    """Max WRT/ANT a validator can profitably pay for declare fuel."""
    l_active = max(1, int(snap.l_total), int(l_i))
    return miner_reservation_price(
        l_i=max(1, int(l_i)),
        l_active=l_active,
        block_reward=int(snap.block_reward),
        lambda_f=float(snap.lambda_f),
    )


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
        # market -> (reviewed_at, ask_price, wrt_at_review, rem_at_review)
        self._ask_quotes: Dict[str, tuple[int, int, int, int]] = {}

    def _next_ask(
        self,
        market: str,
        *,
        height: int,
        wrt: int,
        seed: int,
        rem: int,
        sold: bool,
        force_floor: bool = False,
    ) -> int:
        """Seed once from ref, then markdown on unsold / bump slightly on fill."""
        seed_px = max(1, int(seed))
        if force_floor:
            ask = 1
            self._ask_quotes[market] = (height, ask, int(wrt), max(0, int(rem)))
            return ask
        prev = self._ask_quotes.get(market)
        if prev is None or height < prev[0]:
            ask = seed_px
            self._ask_quotes[market] = (height, ask, int(wrt), max(0, int(rem)))
            return ask
        reviewed_at, ask, _wrt0, _rem0 = prev
        if height - reviewed_at < REVIEW_INTERVAL_BLOCKS:
            return max(1, int(ask))
        if sold:
            adj = ASK_STEP
        else:
            adj = -ASK_STEP
        ask = max(1, int(float(ask) * (1.0 + adj)))
        self._ask_quotes[market] = (height, ask, int(wrt), max(0, int(rem)))
        return ask

    def _ask_sold_signal(self, market: str, wrt: int, rem: int) -> bool:
        prev = self._ask_quotes.get(market)
        if prev is None:
            return False
        _h, _px, wrt0, rem0 = prev
        if int(wrt) > int(wrt0):
            return True
        if rem0 > 0 and int(rem) < rem0:
            return True
        return False

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

    def _update_growth(self, wealth: int, height: int) -> None:
        """Track Δ mark-to-market WRT wealth over the scout horizon."""
        if self._mark_height < 0:
            self.wrt_mark = wealth
            self._mark_height = height
            self.growth_wrt = 0
            return
        self.growth_wrt = wealth - self.wrt_mark
        if height - self._mark_height >= self.horizon:
            self.wrt_mark = wealth
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
        self._update_growth(wealth_wrt(view, snap), height)
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

        # Market + declare before flip so validators never skip mining for a role change.
        try:
            if enable_market:
                out.market = await self._maybe_market(
                    client,
                    view,
                    snap,
                    row,
                    rivals=rivals,
                    alpha=alpha,
                    lam=lam,
                    k=k,
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
                    ant_price=effective_ant_price(snap),
                    b_frac=self.strategy.b_frac,
                    s_frac=self.strategy.s_frac,
                )
                self.last_pick = pick
                out.pick = pick
                # Chain liveness: declare whenever a valid package is affordable — no NPV veto.
                if pick is not None:
                    out.declared = await actions.declare(client, self.wallet, pick.b_i, pick.s_i)
        except Exception as exc:
            out.error = str(exc)

        if (
            enable_role_flip
            and not self.wallet.genesis
            and self.flip_streak >= self.flip_confirm_ticks
            and self._floor_ok(alt_role, n_suppliers, n_validators, min_suppliers, min_validators)
        ):
            out.flip = FlipRequest(agent=self, new_role=alt_role)
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
        rivals: Sequence[Rival] = (),
        alpha: Fraction = Fraction(1, 50),
        lam: Fraction = Fraction(1, 3),
        k: int = 150,
        ant_orderbook: Optional[Dict[str, Any]] = None,
        lzn_orderbook: Optional[Dict[str, Any]] = None,
    ) -> bool:
        del row  # freeze/deactivate no longer consulted
        if self.wallet.role == ROLE_VALIDATOR:
            return await self._validator_market(
                client,
                view,
                snap,
                rivals=rivals,
                alpha=alpha,
                lam=lam,
                k=k,
                ant_orderbook=ant_orderbook,
                lzn_orderbook=lzn_orderbook,
            )
        if self.wallet.role != ROLE_SUPPLIER:
            return False
        return await self._supplier_market(
            client,
            view,
            snap,
            ant_orderbook=ant_orderbook,
            lzn_orderbook=lzn_orderbook,
        )

    async def _supplier_market(
        self,
        client: NodeClient,
        view: AccountView,
        snap: MarketSnapshot,
        *,
        ant_orderbook: Optional[Dict[str, Any]] = None,
        lzn_orderbook: Optional[Dict[str, Any]] = None,
    ) -> bool:
        """SELL ANT/LZN: join bid, else seed ask once and markdown when unsold."""
        did = False
        epoch = max(1, int(snap.epoch_blocks))
        blocks_to_epoch = epoch - (snap.height % epoch) if snap.height > 0 else epoch
        near_wipe = self.horizon >= blocks_to_epoch
        height = int(snap.height)
        wrt = int(view.wrt)

        # A1: ANT ask
        ant_owned = _side_orders(ant_orderbook, self.wallet.address, "SELL")
        ant_rem = sum(int(o.get("remaining") or 0) for o in ant_owned)
        best_ant_bid = _best_bid(ant_orderbook)
        if best_ant_bid > 0:
            ant_price = best_ant_bid
            self._ask_quotes["ANT/WRT"] = (height, ant_price, wrt, ant_rem)
        else:
            seed = max(1, int(quote_ant_ref(snap) * self.strategy.ask_shade))
            sold = self._ask_sold_signal("ANT/WRT", wrt, ant_rem)
            ant_price = self._next_ask(
                "ANT/WRT",
                height=height,
                wrt=wrt,
                seed=seed,
                rem=ant_rem,
                sold=sold,
                force_floor=near_wipe,
            )
        if view.ant >= MIN_ORDER:
            stale = bool(
                ant_owned and int(ant_owned[0].get("price") or 0) != ant_price
            )
            need_post = ant_rem <= 0 or stale or len(ant_owned) > 1
            if need_post:
                # Fresh lot when posting/requoting; consolidate multi-asks even if rem>0.
                lot = max(MIN_ORDER, int(view.ant * 0.10))
                if await self._merge_place(
                    client,
                    market="ANT/WRT",
                    side="SELL",
                    amount=min(lot, view.ant),
                    price=ant_price,
                    existing=ant_owned,
                ):
                    did = True
                    self._ask_quotes["ANT/WRT"] = (
                        height,
                        ant_price,
                        wrt,
                        max(lot, ant_rem),
                    )

        # B1: free LZN ask
        free_lzn = int(view.lzn)
        lzn_owned = _side_orders(lzn_orderbook, self.wallet.address, "SELL")
        lzn_rem = sum(int(o.get("remaining") or 0) for o in lzn_owned)
        best_lzn_bid = _best_bid(lzn_orderbook)
        if best_lzn_bid > 0:
            lzn_price = best_lzn_bid
            self._ask_quotes["LZN/WRT"] = (height, lzn_price, wrt, lzn_rem)
        else:
            seed = max(1, int(quote_lzn_ref(snap) * self.strategy.ask_shade))
            sold = self._ask_sold_signal("LZN/WRT", wrt, lzn_rem)
            lzn_price = self._next_ask(
                "LZN/WRT",
                height=height,
                wrt=wrt,
                seed=seed,
                rem=lzn_rem,
                sold=sold,
                force_floor=near_wipe,
            )
        if free_lzn >= MIN_ORDER:
            stale = bool(
                lzn_owned and int(lzn_owned[0].get("price") or 0) != lzn_price
            )
            need_post = lzn_rem <= 0 or stale or len(lzn_owned) > 1
            if need_post:
                lot = max(MIN_ORDER, int(free_lzn * 0.10))
                if await self._merge_place(
                    client,
                    market="LZN/WRT",
                    side="SELL",
                    amount=min(lot, free_lzn),
                    price=lzn_price,
                    existing=lzn_owned,
                ):
                    did = True
                    self._ask_quotes["LZN/WRT"] = (
                        height,
                        lzn_price,
                        wrt,
                        max(lot, lzn_rem),
                    )
        return did

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
                quote_back += quote_cost(rem, old_px)
        total = extra + released
        if side == "BUY" and price > 0:
            total = min(total, qty_affordable(max(0, quote_free) + quote_back, price))
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

    def _declare_npv(
        self,
        *,
        address: str,
        l_i: int,
        ant: int,
        rivals: Sequence[Rival],
        alpha: Fraction,
        lam: Fraction,
        k: int,
        snap: MarketSnapshot,
        ant_price: int,
        wealth: int = 0,
    ) -> bool:
        """Speculative ANT/LZN: allow when declare pick exists and expected_wrt >= 0.

        Any non-negative expected WRT is enough — no set-entry or loss-tolerance gate.
        Not used for MsgDeclareParticipation or fuel-for-mine / free-LZN activate.
        """
        del wealth  # kept for call-site compat
        if l_i <= 0:
            return False
        pick = pick_declare(
            address=address,
            l_i=l_i,
            ant=ant,
            rivals=rivals,
            alpha=alpha,
            lam=lam,
            k=k,
            subsidy=snap.block_reward,
            fees=int(FEE_PER_TX * AVG_TXS_PER_BLOCK),
            ant_price=max(1, ant_price),
            b_frac=self.strategy.b_frac,
            s_frac=self.strategy.s_frac,
        )
        return pick is not None and pick.expected_wrt >= 0

    def _lzn_buy_npv(
        self,
        view: AccountView,
        snap: MarketSnapshot,
        rivals: Sequence[Rival],
        *,
        alpha: Fraction,
        lam: Fraction,
        k: int,
        delta_lzn: int,
        lzn_price: int,
    ) -> bool:
        """Buy ΔLZN only if horizon WRT net rises (any improvement is enough)."""
        if delta_lzn < MIN_ORDER:
            return False
        fc_now = forecast_validator(
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
        after = AccountView(
            address=view.address,
            role=view.role,
            wrt=max(0, int(view.wrt) - quote_cost(delta_lzn, max(1, lzn_price))),
            lzn=int(view.lzn) + int(delta_lzn),
            lzn_activated=int(view.lzn_activated),
            ant=int(view.ant),
        )
        fc_after = forecast_validator(
            after,
            snap,
            rivals,
            horizon=self.horizon,
            alpha=alpha,
            lam=lam,
            k=k,
            b_frac=self.strategy.b_frac,
            s_frac=self.strategy.s_frac,
        )
        return fc_after.net > fc_now.net

    def _lzn_joint_npv(
        self,
        *,
        view: AccountView,
        snap: MarketSnapshot,
        rivals: Sequence[Rival],
        alpha: Fraction,
        lam: Fraction,
        k: int,
        delta_lzn: int,
        lzn_price: int,
        ant_price: int,
        wealth: int,
    ) -> bool:
        """BUY LZN when net improves, or planned activate+fuel has expected_wrt >= 0."""
        if delta_lzn < MIN_ORDER:
            return False
        cost_lzn = quote_cost(delta_lzn, max(1, lzn_price))
        if int(view.wrt) < cost_lzn + WRT_FEE_RESERVE:
            return False
        L_new = int(view.lzn_activated) + int(view.lzn) + int(delta_lzn)
        remaining = int(view.wrt) - cost_lzn - WRT_FEE_RESERVE
        ant_bought = qty_affordable(remaining, max(1, ant_price))
        ant_total = int(view.ant) + ant_bought
        if self._lzn_buy_npv(
            view,
            snap,
            rivals,
            alpha=alpha,
            lam=lam,
            k=k,
            delta_lzn=delta_lzn,
            lzn_price=lzn_price,
        ):
            return True
        return self._declare_npv(
            address=view.address,
            l_i=L_new,
            ant=ant_total,
            rivals=rivals,
            alpha=alpha,
            lam=lam,
            k=k,
            snap=snap,
            ant_price=ant_price,
            wealth=wealth,
        )

    async def _validator_market(
        self,
        client: NodeClient,
        view: AccountView,
        snap: MarketSnapshot,
        *,
        rivals: Sequence[Rival] = (),
        alpha: Fraction = Fraction(1, 50),
        lam: Fraction = Fraction(1, 3),
        k: int = 150,
        ant_orderbook: Optional[Dict[str, Any]] = None,
        lzn_orderbook: Optional[Dict[str, Any]] = None,
    ) -> bool:
        """Activate free LZN → BUY ANT (mine fuel always; buffer if profitable) → BUY LZN."""
        did = False
        free = int(view.lzn)
        activated = int(view.lzn_activated)
        ant = int(view.ant)
        wrt = int(view.wrt)

        room = _activation_headroom(activated, int(snap.l_total))

        # C1: activate free LZN always — mining capacity must not wait on NPV.
        if free >= MIN_ORDER and room >= MIN_ORDER:
            amt = min(free, room)
            if await actions.activate_lzn(client, self.wallet, amt):
                did = True
                free -= amt
                activated += amt
                room = _activation_headroom(activated, int(snap.l_total) + amt)

        L_i = max(activated, 1)
        ant_target = max(MIN_ORDER, int(snap.lambda_f * L_i * ANT_BUFFER_BLOCKS))
        f_i = entry_burn(activated, alpha) if activated > 0 else 0
        corridor_fuel = f_i + max(1, int(float(lam) * activated)) + 1 if activated > 0 else 0
        reservation = _ant_reservation(snap, L_i)
        best_ask = _best_ask(ant_orderbook)
        ant_buys = _side_orders(ant_orderbook, self.wallet.address, "BUY")
        ant_open = sum(int(o.get("remaining") or 0) for o in ant_buys)
        held = ant + ant_open
        mine_fuel = max(f_i + 1, corridor_fuel) if activated > 0 and corridor_fuel > 0 else (
            (f_i + 1) if activated > 0 else 0
        )
        need_mine = max(0, mine_fuel - held) if activated > 0 else 0
        # Fuel-for-mine: cross best ask (no reservation ceiling). Buffer keeps private cap.
        if need_mine >= MIN_ORDER and best_ask > 0:
            ant_price = best_ask
        elif best_ask > 0:
            ant_price = best_ask if best_ask <= reservation else reservation
        else:
            ant_price = max(1, int(quote_ant_ref(snap) * self.strategy.bid_shade))
            if need_mine < MIN_ORDER and ant_price > reservation:
                ant_price = reservation
        lzn_book_ask = _best_ask(lzn_orderbook)
        if lzn_book_ask > 0:
            lzn_price = lzn_book_ask
        else:
            lzn_price = max(1, int(quote_lzn_ref(snap) * self.strategy.bid_shade))

        # A2: fuel-for-mine always (no NPV); buffer only when expected_wrt >= 0.
        lot = 0
        spare_wrt = wrt - WRT_FEE_RESERVE
        capital = wealth_wrt(
            AccountView(
                address=view.address,
                role=view.role,
                wrt=wrt,
                lzn=free,
                lzn_activated=activated,
                ant=ant,
            ),
            snap,
        )
        if activated > 0 and spare_wrt > quote_cost(MIN_ORDER, ant_price):
            need_all = max(0, max(ant_target, mine_fuel) - held)
            if need_all >= MIN_ORDER:
                affordable = qty_affordable(spare_wrt, ant_price)
                full_lot = min(need_all, affordable)
                mine_lot = min(need_mine, affordable) if need_mine >= MIN_ORDER else 0
                if full_lot >= MIN_ORDER:
                    if mine_lot >= full_lot:
                        lot = full_lot
                    else:
                        npv_ok = self._declare_npv(
                            address=self.wallet.address,
                            l_i=activated,
                            ant=ant + full_lot,
                            rivals=rivals,
                            alpha=alpha,
                            lam=lam,
                            k=k,
                            snap=snap,
                            ant_price=ant_price,
                            wealth=capital,
                        )
                        lot = full_lot if npv_ok else mine_lot
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
                wrt -= quote_cost(extra, ant_price)

        # B2: LZN buy via joint NPV (net improves or expected_wrt >= 0).
        lzn_buys = _side_orders(lzn_orderbook, self.wallet.address, "BUY")
        lzn_open = sum(int(o.get("remaining") or 0) for o in lzn_buys)
        room_left = room - lzn_open
        lzn_lot = 0
        lzn_spare = wrt - WRT_FEE_RESERVE
        # Reserve corridor ANT only when we already have activated capacity.
        if activated > 0:
            ant_reserve = quote_cost(max(0, max(ant_target, corridor_fuel) - ant), ant_price)
            lzn_spare -= ant_reserve
        if room_left >= MIN_ORDER and lzn_spare >= quote_cost(MIN_ORDER, lzn_price):
            n_v = max(1, int(snap.n_validators))
            fair_share = max(MIN_ORDER, int(snap.l_total) // n_v)
            denom = float(alpha) + float(lam)
            # Effective ANT after converting spare WRT (joint plan).
            effective_ant = ant + max(0, qty_affordable(lzn_spare, max(1, ant_price)))
            max_l_for_ant = int(effective_ant / denom) if denom > 0 and effective_ant > 0 else 0
            room_for_ant = max(0, max_l_for_ant - activated - free - lzn_open)
            # Bootstrap at least 1 LZN when empty even if room_for_ant tiny after buy cost.
            if activated + free <= 0:
                room_for_ant = max(
                    room_for_ant, min(SCALE, room_left, qty_affordable(lzn_spare, lzn_price))
                )
            if room_for_ant >= MIN_ORDER:
                view_now = AccountView(
                    address=view.address,
                    role=view.role,
                    wrt=wrt,
                    lzn=free,
                    lzn_activated=activated,
                    ant=ant,
                )
                trials = sorted(
                    {
                        MIN_ORDER,
                        SCALE,
                        min(max(fair_share - lzn_open, MIN_ORDER), room_for_ant),
                        room_for_ant,
                    }
                )
                for trial in trials:
                    candidate = min(room_left, qty_affordable(lzn_spare, lzn_price), trial)
                    if candidate < MIN_ORDER:
                        continue
                    if self._lzn_joint_npv(
                        view=view_now,
                        snap=snap,
                        rivals=rivals,
                        alpha=alpha,
                        lam=lam,
                        k=k,
                        delta_lzn=candidate,
                        lzn_price=lzn_price,
                        ant_price=ant_price,
                        wealth=capital,
                    ):
                        lzn_lot = candidate
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
