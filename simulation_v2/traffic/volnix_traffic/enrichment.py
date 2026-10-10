"""Per-agent horizon forecast (no shared cache). Called from EnrichmentAgent.

All benefit is measured in WRT (mark-to-market + declare net).
"""

from __future__ import annotations

from dataclasses import dataclass
from fractions import Fraction
from math import ceil
from typing import List, Sequence

from volnix_traffic.compete import Rival, entry_burn, pick_declare
from volnix_traffic.profit import (
    ROLE_CITIZEN,
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    SCALE,
    AccountView,
    MarketSnapshot,
    effective_ant_price,
    effective_lzn_price,
    qty_affordable,
    quote_cost,
)

FEE_PER_TX = 0.02 * SCALE
AVG_TXS_PER_BLOCK = 4.0


@dataclass(frozen=True)
class HorizonForecast:
    role: str
    income_wrt: int
    cost_wrt: int
    risk_wrt: int
    enter: bool = False

    @property
    def net(self) -> int:
        return int(self.income_wrt) - int(self.cost_wrt) - int(self.risk_wrt)


def wealth_wrt(account: AccountView, snap: MarketSnapshot) -> int:
    """Mark-to-market wealth in micro-WRT: cash + ANT + LZN."""
    ant_p = effective_ant_price(snap)
    lzn_p = effective_lzn_price(snap)
    lzn_units = int(account.lzn) + int(account.lzn_activated)
    return int(account.wrt) + quote_cost(int(account.ant), ant_p) + quote_cost(lzn_units, lzn_p)


def _sell_efficiency(snap: MarketSnapshot) -> float:
    total = snap.bid_vol + snap.ask_vol
    if total <= 0:
        return 0.70
    tightness = snap.bid_vol / total
    return max(0.35, min(0.92, 0.45 + 0.50 * tightness))


def forecast_validator(
    account: AccountView,
    snap: MarketSnapshot,
    rivals: Sequence[Rival],
    *,
    horizon: int,
    alpha: Fraction,
    lam: Fraction,
    k: int,
    b_frac: float,
    s_frac: float,
) -> HorizonForecast:
    """Validator horizon net in WRT; LZN purchase cost counted when capacity is missing."""
    owned = int(account.lzn_activated) + int(account.lzn)
    ant_price = effective_ant_price(snap)
    lzn_price = effective_lzn_price(snap)
    wrt = int(account.wrt)
    ant = int(account.ant)
    lzn_buy_cost = 0
    L_i = owned

    if L_i <= 0:
        need = SCALE
        cost = quote_cost(need, lzn_price)
        if wrt < cost:
            return HorizonForecast(
                role=ROLE_VALIDATOR, income_wrt=0, cost_wrt=0, risk_wrt=0, enter=False
            )
        L_i = need
        lzn_buy_cost = cost
        wrt -= cost

    fees = int(FEE_PER_TX * AVG_TXS_PER_BLOCK)
    pick = pick_declare(
        address=account.address,
        l_i=L_i,
        ant=ant,
        rivals=rivals,
        alpha=alpha,
        lam=lam,
        k=k,
        subsidy=snap.block_reward,
        fees=fees,
        ant_price=ant_price,
        b_frac=b_frac,
        s_frac=s_frac,
    )
    # If no ANT for declare, assume buying fuel from remaining WRT (MTM-neutral until burn).
    if pick is None and wrt > 0:
        f_i = entry_burn(L_i, alpha)
        b_min = int(ceil(float(lam) * L_i)) if lam else 1
        need_ant = max(0, f_i + max(1, b_min) + 1 - ant)
        affordable = qty_affordable(wrt, ant_price)
        if need_ant > 0 and affordable >= need_ant:
            bought = min(affordable, max(need_ant, int(float(lam) * L_i * 2) if lam else need_ant))
            pick = pick_declare(
                address=account.address,
                l_i=L_i,
                ant=ant + bought,
                rivals=rivals,
                alpha=alpha,
                lam=lam,
                k=k,
                subsidy=snap.block_reward,
                fees=fees,
                ant_price=ant_price,
                b_frac=b_frac,
                s_frac=s_frac,
            )

    if pick is None:
        return HorizonForecast(
            role=ROLE_VALIDATOR,
            income_wrt=0,
            cost_wrt=int(lzn_buy_cost),
            risk_wrt=0,
            enter=False,
        )

    h = max(1, int(horizon))
    # pick.expected_wrt is per-block net (reward - ant_price * burned); may be negative.
    if pick.enter:
        income = int(pick.expected_wrt) * h
        cost = int(lzn_buy_cost)
    else:
        income = 0
        cost = int(lzn_buy_cost)
    return HorizonForecast(
        role=ROLE_VALIDATOR,
        income_wrt=int(income),
        cost_wrt=int(cost),
        risk_wrt=0,
        enter=pick.enter,
    )


def forecast_supplier(
    account: AccountView,
    snap: MarketSnapshot,
    *,
    horizon: int,
) -> HorizonForecast:
    """Supplier income/risk in WRT at book ANT price (epoch wipe = MTM loss)."""
    h = max(1, int(horizon))
    n = max(1, snap.n_suppliers + (0 if account.role == ROLE_SUPPLIER else 1))
    demand = int(snap.lambda_f * max(snap.l_total, 1) * h / n)
    inventory = int(account.ant) if account.role != ROLE_CITIZEN else 0
    # Supplier-role estimate always includes expected epoch ANT share (aspiring citizens too).
    inventory = max(inventory, int(snap.ant_sold_hint / n))
    eff = _sell_efficiency(snap)
    sold = int(min(inventory, max(demand, inventory)) * eff)
    price = effective_ant_price(snap)
    income = quote_cost(sold, price)
    risk = 0
    epoch = max(1, snap.epoch_blocks)
    blocks_to_epoch = epoch - (snap.height % epoch) if snap.height > 0 else epoch
    if h >= blocks_to_epoch:
        unsold = max(0, inventory - sold)
        risk = quote_cost(unsold, price)
    return HorizonForecast(role=ROLE_SUPPLIER, income_wrt=income, cost_wrt=0, risk_wrt=risk)


def forecast_roles(
    account: AccountView,
    snap: MarketSnapshot,
    rivals: Sequence[Rival],
    *,
    horizon: int,
    alpha: Fraction,
    lam: Fraction,
    k: int,
    b_frac: float,
    s_frac: float,
) -> dict[str, HorizonForecast]:
    # Citizen: benefit is already holding WRT; no extra income term.
    return {
        ROLE_VALIDATOR: forecast_validator(
            account,
            snap,
            rivals,
            horizon=horizon,
            alpha=alpha,
            lam=lam,
            k=k,
            b_frac=b_frac,
            s_frac=s_frac,
        ),
        ROLE_SUPPLIER: forecast_supplier(account, snap, horizon=horizon),
        ROLE_CITIZEN: HorizonForecast(role=ROLE_CITIZEN, income_wrt=0, cost_wrt=0, risk_wrt=0),
    }


def card_score(growth_wrt: int, expected_wrt: int) -> tuple[int, int]:
    return (int(growth_wrt), int(expected_wrt))


def sample_peer_indices(n_others: int, k_min: int, k_max: int, rng) -> List[int]:
    if n_others <= 0:
        return []
    lo = max(1, int(k_min))
    hi = max(lo, int(k_max))
    k = rng.randint(lo, hi)
    k = min(k, n_others)
    return rng.sample(range(n_others), k)
