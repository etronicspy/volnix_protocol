"""Profit-driven role selection for traffic bots (canon 5.2-sim, micro-units)."""

from __future__ import annotations

from dataclasses import dataclass, field
from fractions import Fraction
from typing import Any, Optional, Sequence

ROLE_CITIZEN = "citizen"
ROLE_SUPPLIER = "supplier"
ROLE_VALIDATOR = "validator"

# Defaults when chain summary not yet loaded (display WRT, then * SCALE in estimates).
SCALE = 1_000_000
DEFAULT_BLOCK_REWARD = 50 * SCALE
DEFAULT_EPOCH_BLOCKS = 10_080
DEFAULT_LAMBDA = 1.0 / 3.0
FEE_PER_TX = 0.02 * SCALE
AVG_TXS_PER_BLOCK = 4.0
SWITCH_MARGIN_ABS = 5.0 * SCALE
SWITCH_MARGIN_REL = 0.10


@dataclass
class RoleROI:
    role: str
    income_wrt: float = 0.0
    cost_wrt: float = 0.0
    risk_wrt: float = 0.0

    @property
    def net(self) -> float:
        return self.income_wrt - self.cost_wrt - self.risk_wrt


@dataclass
class MarketSnapshot:
    height: int = 0
    epoch_blocks: int = DEFAULT_EPOCH_BLOCKS
    block_reward: int = DEFAULT_BLOCK_REWARD
    lambda_f: float = DEFAULT_LAMBDA
    # Book mid only (0 = empty book — not a synthetic start price).
    ant_price: int = 0
    lzn_price: int = 0
    # Last executed trade (0 = none yet).
    last_ant_price: int = 0
    last_lzn_price: int = 0
    bid_vol: int = 0
    ask_vol: int = 0
    l_total: int = SCALE
    n_validators: int = 1
    n_suppliers: int = 0
    ant_sold_hint: int = 50 * SCALE


@dataclass
class AccountView:
    address: str
    role: str = ROLE_CITIZEN
    wrt: int = 0
    lzn: int = 0
    lzn_activated: int = 0
    ant: int = 0

    @property
    def lzn_total(self) -> int:
        return int(self.lzn) + int(self.lzn_activated)


@dataclass
class ProfitStrategy:
    _roi_cache: dict[str, dict[str, RoleROI]] = field(default_factory=dict, repr=False)

    def estimates(self, account: AccountView, snap: MarketSnapshot) -> dict[str, RoleROI]:
        return {
            ROLE_CITIZEN: self._estimate_citizen(account, snap),
            ROLE_SUPPLIER: self._estimate_supplier(account, snap),
            ROLE_VALIDATOR: self._estimate_validator(account, snap),
        }

    def desired_role(self, account: AccountView, snap: MarketSnapshot) -> str:
        est = self.estimates(account, snap)
        self._roi_cache[account.address] = dict(est)
        return max(est, key=lambda r: est[r].net)

    def best_first_role(
        self,
        account: AccountView,
        snap: MarketSnapshot,
        *,
        rivals: Sequence[Any] = (),
        alpha: Fraction = Fraction(1, 50),
        lam: Fraction = Fraction(1, 3),
        k: int = 150,
        horizon: int = 12,
        b_frac: float = 0.50,
        s_frac: float = 0.40,
    ) -> str:
        """Role to take on first ZKP — same HorizonForecast WRT nets as enrichment tick."""
        # Lazy import avoids circular dependency (enrichment → profit).
        from volnix_traffic.enrichment import forecast_roles

        forecasts = forecast_roles(
            account,
            snap,
            rivals,
            horizon=horizon,
            alpha=alpha,
            lam=lam,
            k=k,
            b_frac=b_frac,
            s_frac=s_frac,
        )
        # Cache as RoleROI-shaped entries for any callers reading get_cached_roi.
        est = {
            role: RoleROI(
                role=role,
                income_wrt=float(fc.income_wrt),
                cost_wrt=float(fc.cost_wrt),
                risk_wrt=float(fc.risk_wrt),
            )
            for role, fc in forecasts.items()
        }
        self._roi_cache[account.address] = dict(est)
        best = max((ROLE_SUPPLIER, ROLE_VALIDATOR), key=lambda r: est[r].net)
        citizen_net = est[ROLE_CITIZEN].net
        best_net = est[best].net
        margin = max(SWITCH_MARGIN_ABS, abs(citizen_net) * SWITCH_MARGIN_REL)
        if best_net <= citizen_net + margin:
            return ROLE_CITIZEN
        return best

    def get_cached_roi(self, address: str) -> Optional[dict[str, RoleROI]]:
        return self._roi_cache.get(address)

    def _ant_price(self, snap: MarketSnapshot) -> float:
        return float(effective_ant_price(snap))

    def _lzn_price(self, snap: MarketSnapshot) -> float:
        return float(effective_lzn_price(snap))

    def _estimate_citizen(self, account: AccountView, snap: MarketSnapshot) -> RoleROI:
        # Citizen benefit is holding WRT; inventory left from a verified role is MTM risk.
        roi = RoleROI(role=ROLE_CITIZEN)
        if account.role != ROLE_CITIZEN and account.ant > 0:
            roi.cost_wrt = quote_cost(account.ant, int(self._ant_price(snap)))
        return roi

    def _estimate_validator(self, account: AccountView, snap: MarketSnapshot) -> RoleROI:
        """Equal-share proxy in WRT; LZN acquisition cost counted when capacity is missing."""
        roi = RoleROI(role=ROLE_VALIDATOR)
        ant_price = self._ant_price(snap)
        lzn_price = self._lzn_price(snap)
        owned = float(account.lzn_total)
        cost_lzn = 0.0
        L_i = owned
        if L_i <= 0:
            cost_lzn = float(quote_cost(SCALE, int(lzn_price)))
            if float(account.wrt) < cost_lzn:
                return roi
            L_i = float(SCALE)

        spare_wrt = max(0.0, float(account.wrt) - cost_lzn)
        ant_budget = float(account.ant)
        # Spend a fraction of spare WRT on declare fuel (price = micro-WRT / whole ANT).
        spend_frac = 0.15 if account.role != ROLE_VALIDATOR else 0.50
        if ant_price > 0:
            ant_budget += spare_wrt * spend_frac * float(SCALE) / ant_price

        target_blocks = snap.epoch_blocks * 0.50
        cap_b = snap.lambda_f * L_i
        b_per_block = min(cap_b, ant_budget / target_blocks) if target_blocks > 0 else 0.0
        if b_per_block <= 0:
            roi.cost_wrt = cost_lzn
            return roi
        blocks_active = min(target_blocks, ant_budget / b_per_block)
        n_decl = max(1, snap.n_validators + (0 if account.role == ROLE_VALIDATOR else 1))
        share = 1.0 / n_decl
        roi.income_wrt = float(snap.block_reward) * share * blocks_active
        roi.income_wrt += FEE_PER_TX * AVG_TXS_PER_BLOCK * share * blocks_active
        roi.cost_wrt = b_per_block * ant_price * blocks_active / float(SCALE) + cost_lzn
        return roi

    def _estimate_supplier(self, account: AccountView, snap: MarketSnapshot) -> RoleROI:
        """Supplier P&L in WRT at book ANT price."""
        roi = RoleROI(role=ROLE_SUPPLIER)
        n = max(1, snap.n_suppliers + (0 if account.role == ROLE_SUPPLIER else 1))
        ant_from_epoch = snap.ant_sold_hint / n
        own = float(account.ant) if account.role != ROLE_CITIZEN else 0.0
        inventory = own + ant_from_epoch
        ant_price = self._ant_price(snap)
        sell_eff = self._sell_efficiency(snap)
        mtm = inventory * ant_price / float(SCALE)
        roi.income_wrt = mtm * sell_eff
        roi.risk_wrt = mtm * (1.0 - sell_eff)
        return roi

    def _sell_efficiency(self, snap: MarketSnapshot) -> float:
        total = snap.bid_vol + snap.ask_vol
        if total <= 0:
            return 0.70
        tightness = snap.bid_vol / total
        return max(0.35, min(0.92, 0.45 + 0.50 * tightness))


def _book_mid(orderbook: dict[str, Any]) -> tuple[int, int, int]:
    """Return (mid_price, bid_vol, ask_vol). mid is 0 if book empty."""
    bids = orderbook.get("bids") or []
    asks = orderbook.get("asks") or []
    bid_vol = sum(int(b.get("remaining") or 0) for b in bids)
    ask_vol = sum(int(a.get("remaining") or 0) for a in asks)
    best_ask = int(asks[0]["price"]) if asks else 0
    best_bid = int(bids[0]["price"]) if bids else 0
    mid = 0
    if best_ask and best_bid:
        mid = (best_ask + best_bid) // 2
    elif best_ask:
        mid = best_ask
    elif best_bid:
        mid = best_bid
    return mid, bid_vol, ask_vol


FAIRNESS_FRAC = 0.50
# Canon §5.4: f_i+b_i+s_i ≤ L_i → worst-case burn is 1 ANT per 1 LZN (Fill ≤ L_decl).
WORST_ANT_PER_LZN = 1.0


def quote_cost(qty: int, price: int) -> int:
    """micro-WRT for ``qty`` micro-base at ``price`` (micro-WRT per whole token)."""
    if qty <= 0 or price <= 0:
        return 0
    return (int(qty) * int(price)) // SCALE


def qty_affordable(wrt: int, price: int) -> int:
    """Max micro-base affordable with ``wrt`` micro-WRT at ``price``."""
    if wrt <= 0 or price <= 0:
        return 0
    return (int(wrt) * SCALE) // int(price)


def reservation_bootstrap(block_reward: int, lambda_f: float, l_total: int) -> int:
    """Fair private ANT value hint in micro-WRT per whole ANT (not a market mid).

    Worst-case declare burns up to 1 ANT per LZN of capacity (§5.4), so
    raw willingness = reward / L_total (WRT per ANT). Fairness splits that
    surplus 50/50 with the counterparty. ``lambda_f`` is unused for pricing
    (λ only caps Σ b_i, not total f+b+s) but kept for call-site compatibility.
    """
    del lambda_f  # corridor λ is not the Fill worst case
    if l_total <= 0:
        return 1
    raw_wrt_per_ant = max(1, int(block_reward / (WORST_ANT_PER_LZN * l_total)))
    return max(1, int(raw_wrt_per_ant * FAIRNESS_FRAC) * SCALE)


def private_ant_value(snap: MarketSnapshot, l_i: int = 0) -> int:
    """Validator willingness to pay for ANT (private; not book mid)."""
    l_active = max(1, int(snap.l_total), int(l_i))
    return reservation_bootstrap(int(snap.block_reward), float(snap.lambda_f), l_active)


def private_lzn_value(snap: MarketSnapshot) -> int:
    """Rough validator willingness for capacity when no LZN book/trade."""
    # Capacity share of block reward per whole LZN → micro-WRT per whole LZN.
    l_total = max(1, int(snap.l_total))
    return max(1, (int(snap.block_reward) // l_total) * SCALE)


def effective_ant_price(snap: MarketSnapshot) -> int:
    """MTM/forecast clearing: book mid → last trade → private opponent value."""
    if int(snap.ant_price) > 0:
        return int(snap.ant_price)
    if int(snap.last_ant_price) > 0:
        return int(snap.last_ant_price)
    return private_ant_value(snap)


def effective_lzn_price(snap: MarketSnapshot) -> int:
    """MTM/forecast clearing: book mid → last trade → private capacity value."""
    if int(snap.lzn_price) > 0:
        return int(snap.lzn_price)
    if int(snap.last_lzn_price) > 0:
        return int(snap.last_lzn_price)
    return private_lzn_value(snap)


def quote_ant_ref(snap: MarketSnapshot) -> int:
    """Seller/buyer reference for ANT when no opposite-side quote to join."""
    if int(snap.last_ant_price) > 0:
        return int(snap.last_ant_price)
    if int(snap.ant_price) > 0:
        return int(snap.ant_price)
    return private_ant_value(snap)


def quote_lzn_ref(snap: MarketSnapshot) -> int:
    """Seller/buyer reference for LZN when no opposite-side quote to join."""
    if int(snap.last_lzn_price) > 0:
        return int(snap.last_lzn_price)
    if int(snap.lzn_price) > 0:
        return int(snap.lzn_price)
    return private_lzn_value(snap)


def snapshot_from_chain(
    summary: dict[str, Any],
    orderbook: dict[str, Any],
    *,
    lzn_orderbook: Optional[dict[str, Any]] = None,
    lambda_num: int = 1,
    lambda_den: int = 3,
) -> MarketSnapshot:
    params = summary.get("params") or {}
    mid, bid_vol, ask_vol = _book_mid(orderbook)
    reward = int(params.get("current_block_reward") or params.get("base_block_reward") or DEFAULT_BLOCK_REWARD)
    l_total = int(summary.get("l_total") or SCALE)
    lam = float(lambda_num) / float(lambda_den) if lambda_den else DEFAULT_LAMBDA
    lzn_mid = 0
    if lzn_orderbook is not None:
        lzn_mid, _, _ = _book_mid(lzn_orderbook)
    last_ant = summary.get("last_ant_wrt_price")
    last_lzn = summary.get("last_lzn_wrt_price")
    return MarketSnapshot(
        height=int(summary.get("height") or 0),
        epoch_blocks=int(params.get("epoch_blocks") or DEFAULT_EPOCH_BLOCKS),
        block_reward=reward,
        lambda_f=lam,
        ant_price=max(0, mid),
        lzn_price=max(0, lzn_mid),
        last_ant_price=int(last_ant) if last_ant is not None else 0,
        last_lzn_price=int(last_lzn) if last_lzn is not None else 0,
        bid_vol=bid_vol,
        ask_vol=ask_vol,
        l_total=l_total,
        n_validators=int(summary.get("n_validators") or 1),
        n_suppliers=int(summary.get("n_suppliers") or 0),
        ant_sold_hint=max(50 * SCALE, int((summary.get("supply") or {}).get("ant") or 0) // 10),
    )


def account_view_from_row(row: dict[str, Any]) -> AccountView:
    return AccountView(
        address=str(row.get("address") or ""),
        role=str(row.get("role") or ROLE_CITIZEN),
        wrt=int(row.get("wrt") or 0),
        lzn=int(row.get("lzn") or 0),
        lzn_activated=int(row.get("lzn_activated") or 0),
        ant=int(row.get("ant") or 0),
    )
