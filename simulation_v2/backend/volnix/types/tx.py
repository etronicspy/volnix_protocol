from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from volnix.crypto.hashing import canonical_bytes, sha256_hex
from volnix.types.events import Event
from volnix.types.msgs import Msg, msg_from_dict


@dataclass
class Fee:
    amount: int  # WRT micro paid as fee
    gas_limit: int
    denom: str = "uwrt"

    def to_dict(self) -> dict[str, Any]:
        return {"amount": self.amount, "gas_limit": self.gas_limit, "denom": self.denom}

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Fee:
        return cls(amount=int(d["amount"]), gas_limit=int(d["gas_limit"]), denom=d.get("denom", "uwrt"))


@dataclass
class SignerInfo:
    address: str
    pub_hex: str
    sequence: int

    def to_dict(self) -> dict[str, Any]:
        return {"address": self.address, "pub_hex": self.pub_hex, "sequence": self.sequence}

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> SignerInfo:
        return cls(address=d["address"], pub_hex=d["pub_hex"], sequence=int(d["sequence"]))


@dataclass
class TxBody:
    messages: list[Msg]
    memo: str = ""
    timeout_height: int = 0

    def to_dict(self) -> dict[str, Any]:
        return {
            "messages": [m.to_dict() for m in self.messages],
            "memo": self.memo,
            "timeout_height": self.timeout_height,
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> TxBody:
        return cls(
            messages=[msg_from_dict(m) for m in d.get("messages", [])],
            memo=d.get("memo", ""),
            timeout_height=int(d.get("timeout_height", 0)),
        )


@dataclass
class AuthInfo:
    signer_infos: list[SignerInfo]
    fee: Fee

    def to_dict(self) -> dict[str, Any]:
        return {
            "signer_infos": [s.to_dict() for s in self.signer_infos],
            "fee": self.fee.to_dict(),
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> AuthInfo:
        return cls(
            signer_infos=[SignerInfo.from_dict(s) for s in d.get("signer_infos", [])],
            fee=Fee.from_dict(d["fee"]),
        )


@dataclass
class Tx:
    body: TxBody
    auth_info: AuthInfo
    signatures: list[str] = field(default_factory=list)

    def sign_bytes(self) -> bytes:
        return canonical_bytes({"body": self.body.to_dict(), "auth_info": self.auth_info.to_dict()})

    def tx_hash(self) -> str:
        payload = {
            "body": self.body.to_dict(),
            "auth_info": self.auth_info.to_dict(),
            "signatures": self.signatures,
        }
        return sha256_hex(canonical_bytes(payload))

    def signer(self) -> str:
        if not self.auth_info.signer_infos:
            raise ValueError("tx has no signer")
        return self.auth_info.signer_infos[0].address

    def to_dict(self) -> dict[str, Any]:
        return {
            "body": self.body.to_dict(),
            "auth_info": self.auth_info.to_dict(),
            "signatures": list(self.signatures),
            "hash": self.tx_hash(),
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Tx:
        return cls(
            body=TxBody.from_dict(d["body"]),
            auth_info=AuthInfo.from_dict(d["auth_info"]),
            signatures=list(d.get("signatures", [])),
        )


@dataclass
class TxResult:
    code: int
    log: str
    gas_wanted: int
    gas_used: int
    events: list[Event] = field(default_factory=list)
    tx_hash: str = ""

    def to_dict(self) -> dict[str, Any]:
        return {
            "code": self.code,
            "log": self.log,
            "gas_wanted": self.gas_wanted,
            "gas_used": self.gas_used,
            "events": [e.to_dict() for e in self.events],
            "tx_hash": self.tx_hash,
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> TxResult:
        return cls(
            code=int(d["code"]),
            log=d.get("log", ""),
            gas_wanted=int(d.get("gas_wanted", 0)),
            gas_used=int(d.get("gas_used", 0)),
            events=[Event.from_dict(e) for e in d.get("events", [])],
            tx_hash=d.get("tx_hash", ""),
        )
