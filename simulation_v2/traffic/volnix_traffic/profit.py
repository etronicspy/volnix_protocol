"""Profit-driven role selection for traffic bots (canon 5.2-sim, micro-units)."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Optional

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
    ant_price: int = 1  # micro-WRT per micro-ANT (ratio)
    lzn_price: int = 1  # micro-WRT per micro-LZN (ratio)
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

    def best_first_role(self, account: AccountView, snap: MarketSnapshot) -> str:
        """Role to take on first ZKP (with hysteresis vs staying citizen)."""
        est = self.estimates(account, snap)
        self._roi_cache[account.address] = dict(est)
        # Prefer verified roles; citizen only if both nets are worse within margin.
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
        return float(max(1, snap.ant_price))

    def _estimate_citizen(self, account: AccountView, snap: MarketSnapshot) -> RoleROI:
        roi = RoleROI(role=ROLE_CITIZEN)
        if account.role != ROLE_CITIZEN and account.ant > 0:
            roi.cost_wrt = account.ant * self._ant_price(snap)
        return roi

    def _estimate_validator(self, account: AccountView, snap: MarketSnapshot) -> RoleROI:
        roi = RoleROI(role=ROLE_VALIDATOR)
        L_i = account.lzn_total or SCALE  # assume can buy ~1 LZN if none yet
        L_total = max(snap.l_total, L_i)
        if account.role != ROLE_VALIDATOR:
            L_total = max(L_total, snap.l_total + L_i)
        share = L_i / max(1, L_total)
        ant_price = self._ant_price(snap)
        ant_budget = float(account.ant)
        if account.role != ROLE_VALIDATOR:
            ant_budget += max(0.0, float(account.wrt)) * 0.15 / ant_price

        target_blocks = snap.epoch_blocks * 0.50
        cap_b = snap.lambda_f * L_i
        b_per_block = min(cap_b, ant_budget / target_blocks) if target_blocks > 0 else 0.0
        if b_per_block <= 0:
            return roi
        blocks_active = min(target_blocks, ant_budget / b_per_block)
        roi.income_wrt = float(snap.block_reward) * share * blocks_active
        n_decl = max(1, snap.n_validators + (0 if account.role == ROLE_VALIDATOR else 1))
        roi.income_wrt += FEE_PER_TX * AVG_TXS_PER_BLOCK * (1.0 / n_decl) * blocks_active
        roi.cost_wrt = b_per_block * ant_price * blocks_active
        return roi

    def _estimate_supplier(self, account: AccountView, snap: MarketSnapshot) -> RoleROI:
        roi = RoleROI(role=ROLE_SUPPLIER)
        n = max(1, snap.n_suppliers + (0 if account.role == ROLE_SUPPLIER else 1))
        ant_from_epoch = snap.ant_sold_hint / n
        own = float(account.ant) if account.role != ROLE_CITIZEN else 0.0
        inventory = own + ant_from_epoch
        ant_price = self._ant_price(snap)
        sell_eff = self._sell_efficiency(snap)
        roi.income_wrt = inventory * ant_price * sell_eff
        roi.risk_wrt = inventory * ant_price * (1.0 - sell_eff)
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


def reservation_bootstrap(block_reward: int, lambda_f: float, l_total: int) -> int:
    """Shared ANT/LZN empty-book bootstrap (WRT per micro-unit)."""
    if l_total <= 0 or lambda_f <= 0:
        return 1
    return max(1, int(block_reward / (lambda_f * l_total)))


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
    bootstrap = reservation_bootstrap(reward, lam, l_total)
    if mid <= 0:
        mid = bootstrap
    lzn_mid = 0
    if lzn_orderbook is not None:
        lzn_mid, _, _ = _book_mid(lzn_orderbook)
    lzn_price = max(1, lzn_mid) if lzn_mid > 0 else bootstrap
    return MarketSnapshot(
        height=int(summary.get("height") or 0),
        epoch_blocks=int(params.get("epoch_blocks") or DEFAULT_EPOCH_BLOCKS),
        block_reward=reward,
        lambda_f=lam,
        ant_price=max(1, mid),
        lzn_price=lzn_price,
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
