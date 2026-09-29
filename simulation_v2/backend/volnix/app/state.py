"""Application state: accounts, books, params, supplies. Deterministically hashable."""

from __future__ import annotations

import copy
from dataclasses import asdict, dataclass, field
from fractions import Fraction
from typing import Any

from volnix.crypto.hashing import hash_obj
from volnix.types.role import Role

SCALE = 1_000_000  # micro-units per displayed token
LZN_HALVING_ERAS = 33
# Constitutional floor for new Supplier admission (§5.6); not DAO-tunable.
SUPPLIER_MIN_EPOCH_INCOME = 1_008_000_000  # micro-WRT (= 1008 WRT)


@dataclass
class Params:
    epoch_blocks: int = 10_080
    halving_interval: int = 2_100_000
    lambda_num: int = 1
    lambda_den: int = 3
    alpha_num: int = 1
    alpha_den: int = 50
    max_active_validators: int = 150
    base_block_time: int = 60
    lzn_total_supply: int = 1_000_000_000  # whole tokens
    base_block_reward: int = 50_000_000  # micro-WRT
    lzn_freeze_period: int = 10_080
    moa_supplier_window: int = 525_600
    moa_validator_window: int = 262_080
    max_active_suppliers: int = 108
    block_gas_limit: int = 40_000_000
    max_block_bytes: int = 22_020_096
    gov_voting_period: int = 10_080
    gov_timelock: int = 20_160
    gov_min_deposit: int = 1_000_000
    gov_quorum_num: int = 2
    gov_quorum_den: int = 5
    gov_threshold_num: int = 1
    gov_threshold_den: int = 2

    @property
    def lambda_(self) -> Fraction:
        return Fraction(self.lambda_num, self.lambda_den)

    @property
    def alpha(self) -> Fraction:
        return Fraction(self.alpha_num, self.alpha_den)

    @property
    def lzn_max_frozen_per_address(self) -> int:
        return (self.lzn_total_supply // 3) * SCALE

    @property
    def lzn_total_supply_micro(self) -> int:
        return self.lzn_total_supply * SCALE

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Params:
        known = {f.name for f in cls.__dataclass_fields__.values()}  # type: ignore[attr-defined]
        return cls(**{k: v for k, v in d.items() if k in known})


# DAO param bounds (canon §7.2)
PARAM_BOUNDS: dict[str, tuple[int, int]] = {
    "lzn_freeze_period": (1, 5_256_000),
    "moa_supplier_window": (1, 10_512_000),
    "moa_validator_window": (1, 5_256_000),
    "block_gas_limit": (1_000, 200_000_000),
    "max_block_bytes": (1024, 100_000_000),
    "max_active_suppliers": (1, 1_000_000),
    "max_active_validators": (1, 500),
    "epoch_blocks": (1, 1_000_000),
    "halving_interval": (100, 10_000_000),
}

# alpha: 1/100 <= α <= 1/10; lambda: λ <= 5/12
LAMBDA_MAX = Fraction(5, 12)
ALPHA_MIN = Fraction(1, 100)
ALPHA_MAX = Fraction(1, 10)


@dataclass
class Account:
    address: str
    pub_hex: str = ""
    role: Role = Role.CITIZEN
    wrt: int = 0
    lzn: int = 0  # free (unactivated) micro-LZN
    lzn_activated: int = 0
    ant: int = 0
    sequence: int = 0
    last_tx_height: int = 0
    zkp_id: str = ""
    genesis_no_zkp: bool = False
    lzn_freeze_until: int = 0  # height when current activation freeze ends
    created_height: int = 0

    def to_dict(self) -> dict[str, Any]:
        d = asdict(self)
        d["role"] = self.role.value if hasattr(self.role, "value") else str(self.role)
        return d

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Account:
        data = dict(d)
        data["role"] = Role(data.get("role", "citizen"))
        known = {f.name for f in cls.__dataclass_fields__.values()}  # type: ignore[attr-defined]
        return cls(**{k: v for k, v in data.items() if k in known})


@dataclass
class Order:
    order_id: str
    owner: str
    market: str
    side: str
    order_type: str
    amount: int
    price: int
    filled: int = 0
    escrow_base: int = 0
    escrow_quote: int = 0
    created_height: int = 0
    created_index: int = 0
    status: str = "open"

    @property
    def remaining(self) -> int:
        return self.amount - self.filled

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Order:
        known = {f.name for f in cls.__dataclass_fields__.values()}  # type: ignore[attr-defined]
        return cls(**{k: v for k, v in d.items() if k in known})


@dataclass
class Proposal:
    proposal_id: int
    proposer: str
    title: str
    description: str
    parameter_changes: dict[str, Any]
    deposit: int
    submit_height: int
    voting_end: int
    execute_height: int
    status: str = "voting"  # voting | passed | rejected | executed
    votes: dict[str, str] = field(default_factory=dict)  # addr -> option

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Proposal:
        known = {f.name for f in cls.__dataclass_fields__.values()}  # type: ignore[attr-defined]
        return cls(**{k: v for k, v in d.items() if k in known})


@dataclass
class EpochRecord:
    epoch: int
    height: int
    l_total: int
    ant_wiped: int
    ant_emit: int
    lzn_emit: int
    suppliers: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> EpochRecord:
        known = {f.name for f in cls.__dataclass_fields__.values()}  # type: ignore[attr-defined]
        return cls(**{k: v for k, v in d.items() if k in known})


@dataclass
class DeclareRecord:
    validator: str
    b_i: int
    s_i: int
    f_i: int = 0
    l_i: int = 0
    w_i: float = 0.0
    valid: bool = True
    reason: str = ""
    passed: bool = False
    excluded: str = ""  # lambda | fill | topk | ""


class AppState:
    def __init__(self) -> None:
        self.params = Params()
        self.accounts: dict[str, Account] = {}
        self.orders: dict[str, Order] = {}
        self.nullifiers: set[str] = set()
        self.declares: dict[str, DeclareRecord] = {}
        self.height: int = 0
        self.chain_id: str = "volnix-sim-2"
        self.genesis_time: str = "2026-01-01T00:00:00Z"
        self.next_order_id: int = 1
        self.next_proposal_id: int = 1
        self.proposals: dict[int, Proposal] = {}
        self.wrt_supply: int = 0
        self.lzn_minted_tokens: int = 0  # whole tokens already issued (incl. genesis 1)
        self.lzn_pool_remaining: int = 999_999_999
        self.epoch: int = 0
        self.epochs: list[EpochRecord] = []
        self.fee_pool: int = 0  # accumulated this block
        self.last_povb: dict[str, Any] = {}
        self.passed_validators: list[str] = []
        self.set_updated: bool = False
        self.genesis_validator: str = ""
        # Last executed ANT/WRT trade price; None = unset (§5.6).
        self.last_ant_wrt_price: int | None = None

    def get_account(self, address: str) -> Account:
        acc = self.accounts.get(address)
        if acc is None:
            acc = Account(address=address, created_height=self.height)
            self.accounts[address] = acc
        return acc

    def ensure_account(self, address: str, pub_hex: str = "") -> Account:
        acc = self.get_account(address)
        if pub_hex and not acc.pub_hex:
            acc.pub_hex = pub_hex
        return acc

    def l_total(self) -> int:
        return sum(a.lzn_activated for a in self.accounts.values() if a.role == Role.VALIDATOR)

    def active_suppliers(self) -> list[Account]:
        return sorted(
            [a for a in self.accounts.values() if a.role == Role.SUPPLIER],
            key=lambda a: a.address,
        )

    def validators(self) -> list[Account]:
        return [a for a in self.accounts.values() if a.role == Role.VALIDATOR]

    def snapshot(self) -> AppState:
        """Isolated copy for rollback. Flat records are shallow-copied; nested proposals are deep."""
        other = AppState.__new__(AppState)
        other.params = copy.copy(self.params)
        other.accounts = {k: copy.copy(v) for k, v in self.accounts.items()}
        other.orders = {k: copy.copy(v) for k, v in self.orders.items()}
        other.nullifiers = set(self.nullifiers)
        other.declares = {k: copy.copy(v) for k, v in self.declares.items()}
        other.height = self.height
        other.chain_id = self.chain_id
        other.genesis_time = self.genesis_time
        other.next_order_id = self.next_order_id
        other.next_proposal_id = self.next_proposal_id
        other.proposals = {k: copy.deepcopy(v) for k, v in self.proposals.items()}
        other.wrt_supply = self.wrt_supply
        other.lzn_minted_tokens = self.lzn_minted_tokens
        other.lzn_pool_remaining = self.lzn_pool_remaining
        other.epoch = self.epoch
        epochs: list[EpochRecord] = []
        for rec in self.epochs:
            copied = copy.copy(rec)
            copied.suppliers = list(rec.suppliers)
            epochs.append(copied)
        other.epochs = epochs
        other.fee_pool = self.fee_pool
        other.last_povb = copy.deepcopy(self.last_povb)
        other.passed_validators = list(self.passed_validators)
        other.set_updated = self.set_updated
        other.genesis_validator = self.genesis_validator
        other.last_ant_wrt_price = self.last_ant_wrt_price
        return other

    def restore(self, other: AppState) -> None:
        """Install a snapshot. The snapshot object must not be used afterwards."""
        self.__dict__.clear()
        self.__dict__.update(other.__dict__)

    def canonical(self) -> dict[str, Any]:
        return {
            "height": self.height,
            "chain_id": self.chain_id,
            "params": self.params.to_dict(),
            "accounts": {k: self.accounts[k].to_dict() for k in sorted(self.accounts)},
            "orders": {k: self.orders[k].to_dict() for k in sorted(self.orders)},
            "nullifiers": sorted(self.nullifiers),
            "wrt_supply": self.wrt_supply,
            "lzn_minted_tokens": self.lzn_minted_tokens,
            "lzn_pool_remaining": self.lzn_pool_remaining,
            "epoch": self.epoch,
            "next_order_id": self.next_order_id,
            "next_proposal_id": self.next_proposal_id,
            "proposals": {str(k): self.proposals[k].to_dict() for k in sorted(self.proposals)},
            "genesis_validator": self.genesis_validator,
            "last_ant_wrt_price": self.last_ant_wrt_price,
        }

    def app_hash(self) -> str:
        return hash_obj(self.canonical())

    def to_snapshot(self) -> dict[str, Any]:
        snap = self.canonical()
        snap["epochs"] = [e.to_dict() for e in self.epochs]
        snap["last_povb"] = dict(self.last_povb)
        snap["genesis_time"] = self.genesis_time
        return snap

    @classmethod
    def from_snapshot(cls, snap: dict[str, Any]) -> AppState:
        st = cls()
        st.height = int(snap.get("height", 0))
        st.chain_id = snap.get("chain_id", "volnix-sim-2")
        st.genesis_time = snap.get("genesis_time", st.genesis_time)
        st.params = Params.from_dict(snap.get("params", {}))
        st.accounts = {k: Account.from_dict(v) for k, v in snap.get("accounts", {}).items()}
        st.orders = {k: Order.from_dict(v) for k, v in snap.get("orders", {}).items()}
        st.nullifiers = set(snap.get("nullifiers", []))
        st.wrt_supply = int(snap.get("wrt_supply", 0))
        st.lzn_minted_tokens = int(snap.get("lzn_minted_tokens", 0))
        st.lzn_pool_remaining = int(snap.get("lzn_pool_remaining", 999_999_999))
        st.epoch = int(snap.get("epoch", 0))
        st.next_order_id = int(snap.get("next_order_id", 1))
        st.next_proposal_id = int(snap.get("next_proposal_id", 1))
        st.proposals = {int(k): Proposal.from_dict(v) for k, v in snap.get("proposals", {}).items()}
        st.genesis_validator = snap.get("genesis_validator", "")
        st.epochs = [EpochRecord.from_dict(e) for e in snap.get("epochs", [])]
        st.last_povb = dict(snap.get("last_povb", {}))
        price = snap.get("last_ant_wrt_price")
        st.last_ant_wrt_price = int(price) if price is not None else None
        return st
