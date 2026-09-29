"""Per-agent horizon forecast (no shared cache). Called from EnrichmentAgent."""

from __future__ import annotations

from dataclasses import dataclass
from fractions import Fraction
from typing import List, Sequence

from volnix_traffic.compete import Rival, pick_declare
from volnix_traffic.profit import (
    ROLE_CITIZEN,
    ROLE_SUPPLIER,
    ROLE_VALIDATOR,
    SCALE,
    AccountView,
    MarketSnapshot,
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
    # Capacity after same-horizon activate: activated + free (not L_i/L_total share).
    L_i = int(account.lzn_activated) + int(account.lzn)
    if L_i <= 0:
        L_i = SCALE  # assume can buy ~1 LZN if none yet (bootstrap)
    fees = int(FEE_PER_TX * AVG_TXS_PER_BLOCK)
    pick = pick_declare(
        address=account.address,
        l_i=L_i,
        ant=account.ant,
        rivals=rivals,
        alpha=alpha,
        lam=lam,
        k=k,
        subsidy=snap.block_reward,
        fees=fees,
        ant_price=max(1, snap.ant_price),
        b_frac=b_frac,
        s_frac=s_frac,
    )
    if pick is None:
        return HorizonForecast(role=ROLE_VALIDATOR, income_wrt=0, cost_wrt=0, risk_wrt=0, enter=False)
    h = max(1, int(horizon))
    # pick.expected_wrt is already net per block (reward - ant_price * burned) — by b_i.
    if pick.enter:
        income = max(0, pick.expected_wrt) * h
        cost = 0
    else:
        income = 0
        cost = max(1, snap.ant_price) * pick.f_i * h
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
    h = max(1, int(horizon))
    n = max(1, snap.n_suppliers + (0 if account.role == ROLE_SUPPLIER else 1))
    demand = int(snap.lambda_f * max(snap.l_total, 1) * h / n)
    inventory = int(account.ant) if account.role != ROLE_CITIZEN else 0
    if account.role == ROLE_SUPPLIER:
        inventory = max(inventory, int(snap.ant_sold_hint / n))
    eff = _sell_efficiency(snap)
    sold = int(min(inventory, max(demand, inventory)) * eff)
    price = max(1, snap.ant_price)
    income = sold * price
    risk = 0
    epoch = max(1, snap.epoch_blocks)
    blocks_to_epoch = epoch - (snap.height % epoch) if snap.height > 0 else epoch
    if h >= blocks_to_epoch:
        unsold = max(0, inventory - sold)
        risk = unsold * price
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
