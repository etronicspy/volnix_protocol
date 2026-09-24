"""Single-process CometBFT rounds: propose → prevote → precommit, +2/3 by power."""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum

from volnix.crypto.hashing import sha256_hex
from volnix.types.block import BlockID, Commit, CommitSig, PartSetHeader
from volnix.types.validator import ValidatorSet


class Decision(str, Enum):
    COMMIT = "commit"
    NIL = "nil"
    TIMEOUT = "timeout"


@dataclass
class FaultModel:
    """Per-address fault injection for the current height."""

    absent: set[str] = field(default_factory=set)
    nil_vote: set[str] = field(default_factory=set)


@dataclass
class RoundDecision:
    decision: Decision
    round: int
    commit: Commit
    voted_power: int
    total_power: int


def _vote_signature(address: str, height: int, round_: int, flag: str, block_hash: str) -> str:
    return sha256_hex(f"{address}|{height}|{round_}|{flag}|{block_hash}".encode())


class ConsensusEngine:
    def __init__(self) -> None:
        self.faults = FaultModel()
        self.round = 0

    def run_rounds(
        self,
        height: int,
        timestamp: str,
        block_hash: str,
        vset: ValidatorSet,
        max_rounds: int = 4,
    ) -> RoundDecision:
        total = vset.total_power()
        threshold = (total * 2) // 3 + 1
        block_id = BlockID(hash=block_hash, parts=PartSetHeader(total=1, hash=block_hash))
        start = self.round
        for r in range(start, start + max_rounds):
            self.round = r
            sigs: list[CommitSig] = []
            voted = 0
            for v in sorted(vset.validators, key=lambda x: x.address):
                if v.address in self.faults.absent:
                    flag = "absent"
                    sig = ""
                elif v.address in self.faults.nil_vote:
                    flag = "nil"
                    sig = _vote_signature(v.address, height, r, "nil", "")
                else:
                    flag = "commit"
                    sig = _vote_signature(v.address, height, r, "commit", block_hash)
                    voted += v.power
                sigs.append(
                    CommitSig(
                        block_id_flag=flag,
                        validator_address=v.address,
                        timestamp=timestamp,
                        signature=sig,
                    )
                )
            commit = Commit(height=height, round=r, block_id=block_id, signatures=sigs)
            if voted >= threshold:
                self.round = 0
                return RoundDecision(
                    decision=Decision.COMMIT,
                    round=r,
                    commit=commit,
                    voted_power=voted,
                    total_power=total,
                )
        self.round += 1
        return RoundDecision(
            decision=Decision.TIMEOUT,
            round=self.round,
            commit=Commit(height=height, round=self.round, block_id=block_id, signatures=[]),
            voted_power=0,
            total_power=total,
        )
