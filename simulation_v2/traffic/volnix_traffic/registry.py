"""Local registry of traffic-managed bot wallets."""

from __future__ import annotations

import json
import secrets
from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional


ROLE_CITIZEN = "citizen"
ROLE_SUPPLIER = "supplier"
ROLE_VALIDATOR = "validator"

KIND_CITIZEN = "citizen"
KIND_ENRICHMENT = "enrichment"
KIND_RETIRED = "retired"


@dataclass
class BotWallet:
    seed: str
    address: str = ""
    pub_hex: str = ""
    role: str = ROLE_CITIZEN
    desired_role: str = ROLE_CITIZEN
    zkp_id: str = ""
    verified: bool = False
    funded: bool = False
    kind: str = KIND_CITIZEN
    genesis: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)

    @classmethod
    def from_dict(cls, d: Dict[str, Any]) -> "BotWallet":
        kind = str(d.get("kind") or "")
        genesis = bool(d.get("genesis"))
        role = str(d.get("role") or ROLE_CITIZEN)
        if not kind:
            if genesis or role in (ROLE_SUPPLIER, ROLE_VALIDATOR):
                kind = KIND_ENRICHMENT
            else:
                kind = KIND_CITIZEN
        return cls(
            seed=str(d.get("seed") or ""),
            address=str(d.get("address") or ""),
            pub_hex=str(d.get("pub_hex") or ""),
            role=role,
            desired_role=str(d.get("desired_role") or ROLE_CITIZEN),
            zkp_id=str(d.get("zkp_id") or ""),
            verified=bool(d.get("verified")),
            funded=bool(d.get("funded")),
            kind=kind,
            genesis=genesis,
        )


class BotRegistry:
    """Seed → wallet map. Not a dataclass (avoids field name clashes with callers)."""

    def __init__(self) -> None:
        self._by_seed: Dict[str, BotWallet] = {}
        self._by_address: Dict[str, str] = {}

    def __len__(self) -> int:
        return len(self._by_seed)

    def all(self) -> List[BotWallet]:
        return list(self._by_seed.values())

    def by_role(self, role: str) -> List[BotWallet]:
        return [b for b in self._by_seed.values() if b.role == role]

    def by_kind(self, kind: str) -> List[BotWallet]:
        return [b for b in self._by_seed.values() if b.kind == kind]

    def citizens(self) -> List[BotWallet]:
        return self.by_kind(KIND_CITIZEN)

    def enrichment(self) -> List[BotWallet]:
        return self.by_kind(KIND_ENRICHMENT)

    def retired(self) -> List[BotWallet]:
        return self.by_kind(KIND_RETIRED)

    def suppliers(self) -> List[BotWallet]:
        return [b for b in self.enrichment() if b.role == ROLE_SUPPLIER]

    def validators(self) -> List[BotWallet]:
        return [b for b in self.enrichment() if b.role == ROLE_VALIDATOR]

    def get_by_address(self, address: str) -> Optional[BotWallet]:
        seed = self._by_address.get(address)
        return self._by_seed.get(seed) if seed else None

    def get_by_seed(self, seed: str) -> Optional[BotWallet]:
        return self._by_seed.get(seed)

    def clear(self) -> None:
        self._by_seed.clear()
        self._by_address.clear()

    def add(self, bot: BotWallet) -> BotWallet:
        self._by_seed[bot.seed] = bot
        if bot.address:
            self._by_address[bot.address] = bot.seed
        return bot

    def sync_from_chain(self, accounts: Iterable[Dict[str, Any]]) -> None:
        """Refresh role from explorer rows for known addresses."""
        by_addr = {a["address"]: a for a in accounts if "address" in a}
        for bot in self._by_seed.values():
            row = by_addr.get(bot.address)
            if not row:
                continue
            bot.role = str(row.get("role") or bot.role)
            if bot.role in (ROLE_SUPPLIER, ROLE_VALIDATOR):
                bot.verified = True
                if bot.kind == KIND_CITIZEN and not bot.genesis:
                    # verified wallets that are still tagged citizen kind stay
                    # citizen-kind only if they were never enrichment
                    pass

    def save(self, path: Path) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        payload = {
            "wallets": [b.to_dict() for b in self.all()],
            "next_index_hint": len(self._by_seed),
        }
        path.write_text(json.dumps(payload, indent=2), encoding="utf-8")

    def load(self, path: Path) -> int:
        if not path.is_file():
            return 0
        try:
            data = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            return 0
        n = 0
        for row in data.get("wallets") or []:
            if not isinstance(row, dict) or not row.get("seed"):
                continue
            self.add(BotWallet.from_dict(row))
            n += 1
        return n

    @staticmethod
    def new_seed(index: int) -> str:
        return f"bot-{index:04d}-{secrets.token_hex(4)}"
