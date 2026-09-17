"""
Security subsystems for the simulation engine — closing spec coverage gaps.

Implements:
  1. MOA (Mechanism of Accountability) — §3.3, §5.3
  2. Min-burn liveness mode (canon v4.20 strict) — §5.4
  3. ValidatorSet recalculation via w_i = s_i/L_i — §6.1
  4. ZKP nullifier uniqueness — §3.1
  5. Supplier (Provider) cap — §7.2 item 8
  6. Proposer MEV reorder model — §5.2

These are opt-in features controlled by SecurityConfig; existing behavior
is unchanged when the config is disabled (backward compatible).
"""
from __future__ import annotations

import time
from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Dict, List, Optional, Set

from core.models import Role, Transaction, TransactionType

if TYPE_CHECKING:
    from core.state import StateManager

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------


@dataclass
class SecurityConfig:
    """Toggle security features independently for gradual rollout."""

    moa_enabled: bool = False
    moa_window_supplier_blocks: int = 1000
    moa_window_validator_blocks: int = 500

    min_burn_enabled: bool = True
    # "canon"    — коридор §5.4: Σb_i ≥ λ·L_total (низ); верх (1−λ)·L_total — в EndBlocker.
    # "positive" — отладка: только Σb_i > 0 (без нижней границы λ).
    min_burn_mode: str = "canon"

    validator_set_update_enabled: bool = False
    max_active_validators_k: int = 150

    zkp_nullifier_enabled: bool = False

    supplier_cap_enabled: bool = False
    max_active_suppliers: int = 50

    mev_reorder_enabled: bool = False
    mev_reorder_mode: str = "fee_priority"


# ---------------------------------------------------------------------------
# 1. MOA — Mechanism of Accountability (§3.3, §5.3)
# ---------------------------------------------------------------------------


@dataclass
class MOATracker:
    """Tracks last_active block height per address. Strip role on inactivity."""

    last_active: Dict[str, int] = field(default_factory=dict)
    config: SecurityConfig = field(default_factory=SecurityConfig)

    def record_activity(self, address: str, height: int) -> None:
        self.last_active[address] = height

    def check_inactivity(
        self,
        state: "StateManager",
        current_height: int,
    ) -> List[dict]:
        """Return list of MOA sanction actions applied this block."""
        if not self.config.moa_enabled:
            return []

        sanctions: List[dict] = []
        for addr, acc in list(state.accounts.items()):
            if acc.role == Role.CITIZEN:
                continue

            window = (
                self.config.moa_window_validator_blocks
                if acc.role == Role.VALIDATOR
                else self.config.moa_window_supplier_blocks
            )
            last = self.last_active.get(addr, 0)
            if current_height - last < window:
                continue

            # Sanction: burn ANT, strip role to CITIZEN, keep LZN
            burned_ant = acc.ant_balance
            acc.ant_balance = 0.0
            acc.role = Role.CITIZEN

            sanctions.append({
                "address": addr,
                "burned_ant": burned_ant,
                "last_active": last,
                "window": window,
                "height": current_height,
            })

        return sanctions


# ---------------------------------------------------------------------------
# 2. Min-burn liveness (canon v4.20 strict mode)
# ---------------------------------------------------------------------------


def check_min_burn_threshold(
    B_applied: float,
    L_total: float,
    burn_cap_lambda: float,
    config: SecurityConfig,
) -> bool:
    """Return True if block passes min-burn check. False = block should stall.

    Canon («canon»): Σb_i ≥ λ·L_total (нижняя граница коридора).
    Верх (1−λ)·L_total применяется отдельно в EndBlocker (λ-отсев).
    Режим «positive» — только отладка: Σb_i > 0.
    """
    if not config.min_burn_enabled:
        return True
    if L_total <= 0:
        return True
    if config.min_burn_mode == "positive":
        return B_applied > 1e-9
    threshold = burn_cap_lambda * L_total
    return B_applied >= threshold - 1e-6


# ---------------------------------------------------------------------------
# 3. ValidatorSet recalculation by w_i = s_i / L_i (§6.1)
# ---------------------------------------------------------------------------


def recalculate_validator_set(
    participation: Dict[str, dict],
    state: "StateManager",
    config: SecurityConfig,
) -> List[dict]:
    """Rebuild ValidatorSet from declare batch using w_i = s_i / L_i.

    Returns sorted list of {address, power, w_i} entries, capped at K.
    Power = w_i (fractional weight) for CometBFT-style proposer selection.
    """
    if not config.validator_set_update_enabled:
        return []

    entries: List[dict] = []
    for addr, data in participation.items():
        s_i = float(data.get("s", 0) or 0)
        L_i = float(data.get("L_i", 0) or 0)
        b_i = float(data.get("b", 0) or 0)
        if b_i <= 0 or L_i <= 0:
            continue
        w_i = s_i / L_i
        entries.append({"address": addr, "power": max(1e-12, w_i), "w_i": w_i})

    # Sort by w_i descending, tie-break by address (deterministic)
    entries.sort(key=lambda x: (-x["w_i"], x["address"]))

    K = config.max_active_validators_k
    return entries[:K]


# ---------------------------------------------------------------------------
# 4. ZKP Nullifier — unique identity hash (§3.1)
# ---------------------------------------------------------------------------


@dataclass
class ZKPRegistry:
    """Maintains a set of identity nullifiers to prevent double-verification."""

    nullifiers: Set[str] = field(default_factory=set)
    config: SecurityConfig = field(default_factory=SecurityConfig)

    def can_verify(self, identity_hash: Optional[str]) -> bool:
        """Check if this identity hash has already been used."""
        if not self.config.zkp_nullifier_enabled:
            return True
        if not identity_hash:
            return True
        return identity_hash not in self.nullifiers

    def register(self, identity_hash: Optional[str]) -> bool:
        """Register nullifier. Returns False if already registered."""
        if not self.config.zkp_nullifier_enabled:
            return True
        if not identity_hash:
            return True
        if identity_hash in self.nullifiers:
            return False
        self.nullifiers.add(identity_hash)
        return True


# ---------------------------------------------------------------------------
# 5. Supplier (Provider) cap — §7.2 item 8
# ---------------------------------------------------------------------------


def check_supplier_cap(state: "StateManager", config: SecurityConfig) -> bool:
    """Return True if a new supplier can be added (under cap)."""
    if not config.supplier_cap_enabled:
        return True
    active_suppliers = sum(
        1 for acc in state.accounts.values() if acc.role == Role.PROVIDER
    )
    return active_suppliers < config.max_active_suppliers


# ---------------------------------------------------------------------------
# 6. Proposer MEV — tx reordering model (§5.2)
# ---------------------------------------------------------------------------


def reorder_mempool_mev(
    mempool: List[Transaction],
    proposer: str,
    config: SecurityConfig,
) -> List[Transaction]:
    """Reorder mempool to model MEV extraction by proposer.

    Modes:
      - fee_priority: sort by amount descending (higher-value txs first)
      - proposer_front: proposer's own txs go first

    Returns reordered mempool (new list).
    """
    if not config.mev_reorder_enabled:
        return mempool

    if config.mev_reorder_mode == "proposer_front":
        proposer_txs = [tx for tx in mempool if tx.sender == proposer]
        other_txs = [tx for tx in mempool if tx.sender != proposer]
        return proposer_txs + other_txs

    # Default: fee_priority (higher amount = higher implicit fee)
    return sorted(mempool, key=lambda tx: -(tx.amount or 0))
