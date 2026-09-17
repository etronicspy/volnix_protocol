"""
Profit-driven role selection for simulation bots.

Each bot estimates expected WRT net over one ANT epoch and switches
to the most profitable eligible role. Ineligible but more profitable
roles become prep targets (ZKP, activate LZN).

Economics (canon):
    Validator (§5.1, §5.4):
        income  = 50 WRT × (L_i / L_declaring) × blocks_with_declare
                + fees × (b_i / B) × those blocks
        cost    = ANT burned (b_i ≈ λ·L_i) × last_price

    Provider (§5.2, §5.5):
        income  = sellable ANT × last_price × sell_efficiency
        risk    = unsold ANT wiped at epoch boundary

    Citizen:
        income  = 0; switching here burns ANT (§4.2)
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Dict, List, Optional, Tuple

from core.models import OrderType, Role
from core.state import (
    BLOCKS_PER_EPOCH,
    BURN_CAP_LAMBDA,
    account_total_lzn,
    eligible_for_provider_role,
    eligible_for_validator_role,
)

if TYPE_CHECKING:
    from core.models import Account
    from core.state import StateManager

BLOCK_REWARD_WRT = 50.0
FEE_PER_TX_WRT = 0.02
AVG_TXS_PER_BLOCK = 4.0
# Hysteresis: do not flip roles for a tiny edge (stops oscillation).
SWITCH_MARGIN_ABS = 5.0
SWITCH_MARGIN_REL = 0.10


@dataclass
class RoleROI:
    """Estimated WRT profit per epoch for a given role."""

    role: Role
    income_wrt: float = 0.0
    cost_wrt: float = 0.0
    risk_wrt: float = 0.0

    @property
    def net(self) -> float:
        return self.income_wrt - self.cost_wrt - self.risk_wrt


@dataclass
class ProfitStrategy:
    """Evaluates which role maximises profit for a given bot account."""

    _roi_cache: Dict[str, Dict[str, RoleROI]] = field(default_factory=dict, repr=False)

    def estimates(
        self,
        account: "Account",
        state: "StateManager",
    ) -> Dict[Role, RoleROI]:
        return {
            Role.CITIZEN: self._estimate_citizen(account, state),
            Role.PROVIDER: self._estimate_provider(account, state),
            Role.VALIDATOR: self._estimate_validator(account, state),
        }

    def desired_role(
        self,
        address: str,
        account: "Account",
        state: "StateManager",
    ) -> Role:
        """Highest-ROI role ignoring eligibility (used for prep: ZKP / activate LZN)."""
        est = self.estimates(account, state)
        self._roi_cache[address] = {r.value: roi for r, roi in est.items()}
        return max(est, key=lambda r: est[r].net)

    def best_role(
        self,
        address: str,
        account: "Account",
        state: "StateManager",
    ) -> Optional[Role]:
        """Eligible role with highest expected ROI, or None if current is good enough."""
        estimates = self.estimates(account, state)
        self._roi_cache[address] = {r.value: roi for r, roi in estimates.items()}

        eligible: Dict[Role, RoleROI] = {}
        for role, roi in estimates.items():
            if role == account.role:
                eligible[role] = roi
                continue
            if role == Role.VALIDATOR and not eligible_for_validator_role(address, account):
                continue
            if role == Role.PROVIDER and not eligible_for_provider_role(address, account):
                continue
            eligible[role] = roi

        if not eligible:
            return None

        best = max(eligible, key=lambda r: eligible[r].net)
        current_net = eligible.get(account.role, estimates[account.role]).net
        best_net = eligible[best].net

        if best == account.role:
            return None

        margin = max(SWITCH_MARGIN_ABS, abs(current_net) * SWITCH_MARGIN_REL)
        if best_net <= current_net + margin:
            return None

        return best

    def rank_switch_delta(
        self,
        address: str,
        account: "Account",
        state: "StateManager",
    ) -> Tuple[Optional[Role], float]:
        """(new_role, net_delta) for batch reeval sorting."""
        new_role = self.best_role(address, account, state)
        if new_role is None:
            return None, 0.0
        cache = self._roi_cache.get(address) or {}
        cur = cache.get(account.role.value)
        nxt = cache.get(new_role.value)
        if not cur or not nxt:
            return new_role, 0.0
        return new_role, nxt.net - cur.net

    def get_cached_roi(self, address: str) -> Optional[Dict[str, RoleROI]]:
        return self._roi_cache.get(address)

    def _ant_price(self, state: "StateManager") -> float:
        return max(0.01, float(state.last_price)) if state.last_price > 0 else 5.0

    def _estimate_citizen(self, account: "Account", state: "StateManager") -> RoleROI:
        roi = RoleROI(role=Role.CITIZEN)
        # §4.2: becoming a citizen burns ANT — treat remaining ANT as a switching cost.
        if account.role != Role.CITIZEN and account.ant_balance > 1e-12:
            roi.cost_wrt = account.ant_balance * self._ant_price(state)
        return roi

    def _estimate_validator(
        self, account: "Account", state: "StateManager"
    ) -> RoleROI:
        roi = RoleROI(role=Role.VALIDATOR)

        L_i = account_total_lzn(account)
        if L_i <= 1e-12:
            return roi

        validators = [a for a in state.accounts.values() if a.role == Role.VALIDATOR]
        L_total = sum(account_total_lzn(v) for v in validators)
        if account.role != Role.VALIDATOR:
            L_total += L_i
        if L_total <= 1e-12:
            L_total = L_i

        share = L_i / L_total
        ant_price = self._ant_price(state)
        ant_budget = float(account.ant_balance)
        if account.role != Role.VALIDATOR:
            ant_budget += max(0.0, account.wrt_balance) * 0.15 / ant_price

        # Reward needs only b_i>0; a profit-seeker will not burn λ·L_i every height.
        target_blocks = BLOCKS_PER_EPOCH * 0.50
        cap_b = BURN_CAP_LAMBDA * L_i
        b_per_block = min(cap_b, ant_budget / target_blocks) if target_blocks > 0 else 0.0
        if b_per_block <= 1e-12:
            return roi
        blocks_active = min(target_blocks, ant_budget / b_per_block)

        roi.income_wrt = BLOCK_REWARD_WRT * share * blocks_active
        avg_fee_pool = FEE_PER_TX_WRT * AVG_TXS_PER_BLOCK
        n_declaring = max(
            1,
            len([v for v in validators if v.ant_balance > 0.1])
            + (0 if account.role == Role.VALIDATOR else 1),
        )
        roi.income_wrt += avg_fee_pool * (1.0 / n_declaring) * blocks_active
        roi.cost_wrt = b_per_block * ant_price * blocks_active
        return roi

    def _estimate_provider(
        self, account: "Account", state: "StateManager"
    ) -> RoleROI:
        roi = RoleROI(role=Role.PROVIDER)

        providers = [a for a in state.accounts.values() if a.role == Role.PROVIDER]
        n_providers = len(providers) + (0 if account.role == Role.PROVIDER else 1)
        n_providers = max(1, n_providers)

        coeff = float(getattr(state, "epoch_emission_coefficient", 1.0) or 1.0)
        sold_last = max(
            float(getattr(state, "epoch_ant_sold_last", 0.0) or 0.0),
            float(getattr(state, "epoch_ant_sold_volume", 0.0) or 0.0),
        )
        if sold_last <= 1e-12:
            # Cold start: assume a modest weekly book until the first epoch prints.
            sold_last = 50.0 * max(1, len(providers))

        expected_emission = sold_last * coeff
        ant_from_epoch = expected_emission / n_providers
        own_ant = float(account.ant_balance) if account.role != Role.CITIZEN else 0.0
        inventory = own_ant + ant_from_epoch

        ant_price = self._ant_price(state)
        sell_efficiency = self._provider_sell_efficiency(state)
        roi.income_wrt = inventory * ant_price * sell_efficiency
        unsold = 1.0 - sell_efficiency
        roi.risk_wrt = inventory * ant_price * unsold
        return roi

    def _provider_sell_efficiency(self, state: "StateManager") -> float:
        """Tighter book (bids vs asks) → easier to convert ANT to WRT before wipe."""
        bids = 0.0
        asks = 0.0
        for o in state.orders.values():
            remain = max(0.0, float(o.amount) - float(o.filled))
            if o.order_type == OrderType.BUY:
                bids += remain
            elif o.order_type == OrderType.SELL:
                asks += remain
        if bids + asks <= 1e-12:
            return 0.70
        tightness = bids / (bids + asks)
        return max(0.35, min(0.92, 0.45 + 0.50 * tightness))


def top_switch_candidates(
    strategy: ProfitStrategy,
    accounts: List["Account"],
    state: "StateManager",
    limit: int,
) -> List[Tuple["Account", Role, float]]:
    """Accounts with the largest profitable role delta, capped at ``limit``."""
    ranked: List[Tuple["Account", Role, float]] = []
    for acc in accounts:
        new_role, delta = strategy.rank_switch_delta(acc.address, acc, state)
        if new_role is None or delta <= 0:
            continue
        ranked.append((acc, new_role, delta))
    ranked.sort(key=lambda item: item[2], reverse=True)
    return ranked[: max(0, limit)]
