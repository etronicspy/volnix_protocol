from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from volnix.crypto.hashing import hash_obj
from volnix.crypto.merkle import merkle_root
from volnix.types.events import Event
from volnix.types.tx import Tx, TxResult


@dataclass
class PartSetHeader:
    total: int = 1
    hash: str = ""

    def to_dict(self) -> dict[str, Any]:
        return {"total": self.total, "hash": self.hash}

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> PartSetHeader:
        return cls(total=int(d.get("total", 1)), hash=d.get("hash", ""))


@dataclass
class BlockID:
    hash: str
    parts: PartSetHeader

    def to_dict(self) -> dict[str, Any]:
        return {"hash": self.hash, "parts": self.parts.to_dict()}

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> BlockID:
        return cls(hash=d.get("hash", ""), parts=PartSetHeader.from_dict(d.get("parts", {})))


def empty_block_id() -> BlockID:
    return BlockID(hash="", parts=PartSetHeader())


@dataclass
class Header:
    version: dict[str, str]
    chain_id: str
    height: int
    time: str
    last_block_id: BlockID
    last_commit_hash: str
    data_hash: str
    validators_hash: str
    next_validators_hash: str
    consensus_hash: str
    app_hash: str
    last_results_hash: str
    evidence_hash: str
    proposer_address: str

    def to_dict(self) -> dict[str, Any]:
        return {
            "version": dict(self.version),
            "chain_id": self.chain_id,
            "height": self.height,
            "time": self.time,
            "last_block_id": self.last_block_id.to_dict(),
            "last_commit_hash": self.last_commit_hash,
            "data_hash": self.data_hash,
            "validators_hash": self.validators_hash,
            "next_validators_hash": self.next_validators_hash,
            "consensus_hash": self.consensus_hash,
            "app_hash": self.app_hash,
            "last_results_hash": self.last_results_hash,
            "evidence_hash": self.evidence_hash,
            "proposer_address": self.proposer_address,
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Header:
        return cls(
            version=dict(d.get("version", {})),
            chain_id=d["chain_id"],
            height=int(d["height"]),
            time=d["time"],
            last_block_id=BlockID.from_dict(d.get("last_block_id", {})),
            last_commit_hash=d.get("last_commit_hash", ""),
            data_hash=d.get("data_hash", ""),
            validators_hash=d.get("validators_hash", ""),
            next_validators_hash=d.get("next_validators_hash", ""),
            consensus_hash=d.get("consensus_hash", ""),
            app_hash=d.get("app_hash", ""),
            last_results_hash=d.get("last_results_hash", ""),
            evidence_hash=d.get("evidence_hash", ""),
            proposer_address=d.get("proposer_address", ""),
        )

    def hash(self) -> str:
        return hash_obj(self.to_dict())


@dataclass
class CommitSig:
    block_id_flag: str  # commit | nil | absent
    validator_address: str
    timestamp: str
    signature: str

    def to_dict(self) -> dict[str, Any]:
        return {
            "block_id_flag": self.block_id_flag,
            "validator_address": self.validator_address,
            "timestamp": self.timestamp,
            "signature": self.signature,
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> CommitSig:
        return cls(
            block_id_flag=d.get("block_id_flag", "absent"),
            validator_address=d.get("validator_address", ""),
            timestamp=d.get("timestamp", ""),
            signature=d.get("signature", ""),
        )


@dataclass
class Commit:
    height: int
    round: int
    block_id: BlockID
    signatures: list[CommitSig] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return {
            "height": self.height,
            "round": self.round,
            "block_id": self.block_id.to_dict(),
            "signatures": [s.to_dict() for s in self.signatures],
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Commit:
        return cls(
            height=int(d.get("height", 0)),
            round=int(d.get("round", 0)),
            block_id=BlockID.from_dict(d.get("block_id", {})),
            signatures=[CommitSig.from_dict(s) for s in d.get("signatures", [])],
        )

    def hash(self) -> str:
        return hash_obj(self.to_dict())


@dataclass
class Evidence:
    type: str
    height: int
    validator_address: str
    details: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "type": self.type,
            "height": self.height,
            "validator_address": self.validator_address,
            "details": dict(self.details),
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Evidence:
        return cls(
            type=d["type"],
            height=int(d["height"]),
            validator_address=d.get("validator_address", ""),
            details=dict(d.get("details", {})),
        )


@dataclass
class Block:
    header: Header
    data: dict[str, Any]
    evidence: list[Evidence] = field(default_factory=list)
    last_commit: Commit | None = None

    def txs(self) -> list[Tx]:
        return [Tx.from_dict(t) for t in self.data.get("txs", [])]

    def hash(self) -> str:
        return self.header.hash()

    def to_dict(self) -> dict[str, Any]:
        return {
            "header": self.header.to_dict(),
            "data": self.data,
            "evidence": [e.to_dict() for e in self.evidence],
            "last_commit": self.last_commit.to_dict() if self.last_commit else None,
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> Block:
        lc = d.get("last_commit")
        return cls(
            header=Header.from_dict(d["header"]),
            data=d.get("data", {"txs": []}),
            evidence=[Evidence.from_dict(e) for e in d.get("evidence", [])],
            last_commit=Commit.from_dict(lc) if lc else None,
        )


@dataclass
class BlockMeta:
    block_id: BlockID
    header: Header

    def to_dict(self) -> dict[str, Any]:
        return {"block_id": self.block_id.to_dict(), "header": self.header.to_dict()}


def txs_data_hash(tx_hashes: list[str]) -> str:
    return merkle_root([bytes.fromhex(h) for h in tx_hashes])


def results_hash(results: list[TxResult]) -> str:
    return merkle_root([hash_obj(r.to_dict()).encode() for r in results])


def evidence_hash(items: list[Evidence]) -> str:
    return merkle_root([hash_obj(e.to_dict()).encode() for e in items])


@dataclass
class BlockResults:
    height: int
    txs_results: list[TxResult] = field(default_factory=list)
    begin_block_events: list[Event] = field(default_factory=list)
    end_block_events: list[Event] = field(default_factory=list)
    validator_updates: list[dict[str, Any]] = field(default_factory=list)
    consensus_param_updates: dict[str, Any] = field(default_factory=dict)
    povb: dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> dict[str, Any]:
        return {
            "height": self.height,
            "txs_results": [r.to_dict() for r in self.txs_results],
            "begin_block_events": [e.to_dict() for e in self.begin_block_events],
            "end_block_events": [e.to_dict() for e in self.end_block_events],
            "validator_updates": list(self.validator_updates),
            "consensus_param_updates": dict(self.consensus_param_updates),
            "povb": dict(self.povb),
        }

    @classmethod
    def from_dict(cls, d: dict[str, Any]) -> BlockResults:
        return cls(
            height=int(d["height"]),
            txs_results=[TxResult.from_dict(r) for r in d.get("txs_results", [])],
            begin_block_events=[Event.from_dict(e) for e in d.get("begin_block_events", [])],
            end_block_events=[Event.from_dict(e) for e in d.get("end_block_events", [])],
            validator_updates=list(d.get("validator_updates", [])),
            consensus_param_updates=dict(d.get("consensus_param_updates", {})),
            povb=dict(d.get("povb", {})),
        )
