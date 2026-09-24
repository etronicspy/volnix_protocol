from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from volnix.crypto.hashing import hash_obj

POWER_SCALE = 1_000_000


def voting_power(s_i: int, l_i: int) -> int:
    """VotingPower_i = max(1, ⌊s_i · 10⁶ / L_i⌋) — canon §6.1."""
    if l_i <= 0 or s_i <= 0:
        return 1
    return max(1, (s_i * POWER_SCALE) // l_i)


def weight_to_power(w_i: float) -> int:
    """Legacy helper; prefer voting_power(s_i, l_i)."""
    return max(1, int(w_i * POWER_SCALE))


@dataclass
class Validator:
    address: str
    pub_hex: str
    l_i: int
    s_i: int
    w_i: float
    power: int
    proposer_priority: int = 0

    def to_dict(self) -> dict[str, Any]:
        return {
            "address": self.address,
            "pub_hex": self.pub_hex,
            "l_i": self.l_i,
            "s_i": self.s_i,
            "w_i": self.w_i,
            "power": self.power,
            "proposer_priority": self.proposer_priority,
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Validator:
        return cls(
            address=d["address"],
            pub_hex=d["pub_hex"],
            l_i=int(d["l_i"]),
            s_i=int(d["s_i"]),
            w_i=float(d["w_i"]),
            power=int(d["power"]),
            proposer_priority=int(d.get("proposer_priority", 0)),
        )


@dataclass
class ValidatorSet:
    validators: list[Validator] = field(default_factory=list)
    proposer: str = ""

    def total_power(self) -> int:
        return sum(v.power for v in self.validators)

    def by_address(self) -> dict[str, Validator]:
        return {v.address: v for v in self.validators}

    def hash(self) -> str:
        payload = [
            {"address": v.address, "pub_hex": v.pub_hex, "power": v.power, "w_i": v.w_i}
            for v in sorted(self.validators, key=lambda x: x.address)
        ]
        return hash_obj(payload)

    def to_dict(self) -> dict[str, Any]:
        return {
            "validators": [v.to_dict() for v in self.validators],
            "proposer": self.proposer,
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> ValidatorSet:
        return cls(
            validators=[Validator.from_dict(v) for v in d.get("validators", [])],
            proposer=d.get("proposer", ""),
        )

    def copy(self) -> ValidatorSet:
        return ValidatorSet.from_dict(self.to_dict())
