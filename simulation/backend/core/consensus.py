"""Многовалидаторный консенсус (Tendermint-style, упрощённый) для симуляции.

Раунд блока: Propose → PreVote → PreCommit с весами `power = max(s, w·L_i)`
из `consensus_validator_set` (см. `state.consensus_validator_set_from_participation`).

Этот модуль детерминирован и не делает I/O: на каждый раунд возвращается отчёт
с голосами, кворумом ≥ +2/3, и решением (`commit | nil | timeout`). Engine после
этого либо коммитит блок, либо переходит к следующему пропозеру и/или раунду.

Slashing-двух-голосов (double-sign) фиксируется отдельно через `EvidenceLog`;
санкции применяет engine в EndBlock.
"""
from __future__ import annotations

import random
from dataclasses import dataclass, field
from enum import Enum
from typing import Dict, List, Optional


class Vote(str, Enum):
    NIL = "nil"
    BLOCK = "block"
    ABSENT = "absent"  # валидатор не успел проголосовать


class Decision(str, Enum):
    COMMIT = "commit"
    NIL = "nil"
    TIMEOUT = "timeout"


# Канонический порог Tendermint: >2/3 power для commit.
QUORUM_FRACTION = 2.0 / 3.0


@dataclass
class ValidatorEntry:
    address: str
    power: float


@dataclass
class RoundReport:
    height: int
    round: int
    proposer: str
    pre_votes: Dict[str, Vote]
    pre_commits: Dict[str, Vote]
    power_total: float
    power_block_pv: float
    power_nil_pv: float
    power_block_pc: float
    power_nil_pc: float
    decision: Decision
    notes: List[str] = field(default_factory=list)

    def as_block_log_entry(self) -> dict:
        return {
            "height": self.height,
            "round": self.round,
            "proposer": self.proposer,
            "decision": self.decision.value,
            "power_total": round(self.power_total, 6),
            "power_block_pre_vote": round(self.power_block_pv, 6),
            "power_nil_pre_vote": round(self.power_nil_pv, 6),
            "power_block_pre_commit": round(self.power_block_pc, 6),
            "power_nil_pre_commit": round(self.power_nil_pc, 6),
            "votes": {
                "pre_vote": {a: v.value for a, v in self.pre_votes.items()},
                "pre_commit": {a: v.value for a, v in self.pre_commits.items()},
            },
            "notes": list(self.notes),
        }


@dataclass
class Evidence:
    """Двух-голосовое поведение: одного и того же высоты валидатор голосует
    различные блоки → нарушение. В упрощённой модели генерируем при PreVote vs
    PreCommit, дающих разные значения (BLOCK ↔ NIL) при стресс-тесте."""

    height: int
    address: str
    kind: str  # "double_sign" | "absent" | "conflicting"
    detail: str = ""


def normalize_validator_set(raw: List[dict]) -> List[ValidatorEntry]:
    """Преобразовать `state.consensus_validator_set` в типизированный список,
    нормализуя дубли и неположительные power."""
    seen: Dict[str, float] = {}
    for entry in raw or []:
        addr = entry.get("address") if isinstance(entry, dict) else None
        if not addr:
            continue
        power = float(entry.get("power", 0) or 0)
        if power <= 0:
            continue
        seen[addr] = seen.get(addr, 0.0) + power
    return [ValidatorEntry(address=a, power=p) for a, p in sorted(seen.items())]


def quorum_threshold(power_total: float) -> float:
    return QUORUM_FRACTION * power_total


@dataclass
class FaultModel:
    """Параметры стресс-симуляции консенсуса (можно установить из bot-probe).

    p_absent — вероятность не подписать раунд (валидатор «офлайн»);
    p_nil — вероятность подписать NIL вместо BLOCK;
    p_double_sign — вероятность подписать BLOCK на pre_vote и NIL на pre_commit
                    (или наоборот) — порождает Evidence.
    """

    p_absent: float = 0.0
    p_nil: float = 0.0
    p_double_sign: float = 0.0
    seed: Optional[int] = None

    def rng(self) -> random.Random:
        return random.Random(self.seed)


def run_round(
    *,
    height: int,
    round_idx: int,
    proposer: str,
    validators: List[ValidatorEntry],
    fault_model: Optional[FaultModel] = None,
    evidence_sink: Optional[List[Evidence]] = None,
) -> RoundReport:
    """Симулировать один раунд: пропозер уже выбран снаружи.

    Возвращает RoundReport с decision (commit/nil/timeout). Evidence (double_sign)
    добавляются в evidence_sink, если передан.
    """
    fm = fault_model or FaultModel()
    rng = fm.rng()

    if not validators:
        return RoundReport(
            height=height,
            round=round_idx,
            proposer=proposer,
            pre_votes={},
            pre_commits={},
            power_total=0.0,
            power_block_pv=0.0,
            power_nil_pv=0.0,
            power_block_pc=0.0,
            power_nil_pc=0.0,
            decision=Decision.NIL,
            notes=["empty validator set"],
        )

    power_total = sum(v.power for v in validators)
    threshold = quorum_threshold(power_total)

    pre_votes: Dict[str, Vote] = {}
    power_block_pv = 0.0
    power_nil_pv = 0.0
    # Пропозер всегда голосует BLOCK в своём раунде, если не «отсутствует»
    # (упрощение реального Tendermint).
    for v in validators:
        if rng.random() < fm.p_absent:
            pre_votes[v.address] = Vote.ABSENT
            continue
        # NIL чаще от не-пропозера
        prefers_nil = rng.random() < fm.p_nil and v.address != proposer
        vote = Vote.NIL if prefers_nil else Vote.BLOCK
        pre_votes[v.address] = vote
        if vote == Vote.BLOCK:
            power_block_pv += v.power
        elif vote == Vote.NIL:
            power_nil_pv += v.power

    # Если кворум NIL — пропозер «отвергнут», commit невозможен в этом раунде
    nil_quorum_pv = power_nil_pv > threshold
    block_quorum_pv = power_block_pv > threshold

    pre_commits: Dict[str, Vote] = {}
    power_block_pc = 0.0
    power_nil_pc = 0.0
    notes: List[str] = []

    if nil_quorum_pv:
        # Если pre_vote NIL >2/3 — pre_commit тоже NIL (квалифицированный nil)
        for v in validators:
            if pre_votes.get(v.address) == Vote.ABSENT:
                pre_commits[v.address] = Vote.ABSENT
                continue
            pre_commits[v.address] = Vote.NIL
            power_nil_pc += v.power
        notes.append("pre_vote NIL >2/3 — раунд отказан в пользу следующего пропозера")
    elif block_quorum_pv:
        for v in validators:
            pv = pre_votes.get(v.address, Vote.ABSENT)
            if pv == Vote.ABSENT:
                pre_commits[v.address] = Vote.ABSENT
                continue
            # double-sign: pre_vote BLOCK → pre_commit NIL
            if pv == Vote.BLOCK and rng.random() < fm.p_double_sign:
                pre_commits[v.address] = Vote.NIL
                power_nil_pc += v.power
                if evidence_sink is not None:
                    evidence_sink.append(
                        Evidence(
                            height=height,
                            address=v.address,
                            kind="double_sign",
                            detail="pre_vote=BLOCK; pre_commit=NIL",
                        )
                    )
                continue
            pre_commits[v.address] = pv  # держим тот же
            if pv == Vote.BLOCK:
                power_block_pc += v.power
            elif pv == Vote.NIL:
                power_nil_pc += v.power
    else:
        # Ни BLOCK, ни NIL не набрали >2/3 → timeout раунда
        for v in validators:
            pre_commits[v.address] = pre_votes.get(v.address, Vote.ABSENT)
            pv = pre_votes.get(v.address, Vote.ABSENT)
            if pv == Vote.BLOCK:
                power_block_pc += v.power
            elif pv == Vote.NIL:
                power_nil_pc += v.power
        notes.append("ни BLOCK, ни NIL не набрали +2/3 на pre_vote")

    if power_block_pc > threshold:
        decision = Decision.COMMIT
    elif power_nil_pc > threshold:
        decision = Decision.NIL
    else:
        decision = Decision.TIMEOUT

    return RoundReport(
        height=height,
        round=round_idx,
        proposer=proposer,
        pre_votes=pre_votes,
        pre_commits=pre_commits,
        power_total=power_total,
        power_block_pv=power_block_pv,
        power_nil_pv=power_nil_pv,
        power_block_pc=power_block_pc,
        power_nil_pc=power_nil_pc,
        decision=decision,
        notes=notes,
    )


def run_consensus(
    *,
    height: int,
    validators: List[dict],
    proposer_for_round_fn,
    fault_model: Optional[FaultModel] = None,
    max_rounds: int = 4,
) -> tuple[List[RoundReport], List[Evidence]]:
    """Запустить до `max_rounds` раундов. Возвращает (отчёты, evidence).

    `proposer_for_round_fn(round_idx)` должен вернуть address пропозера для раунда;
    обычно: `select_proposer_for_height(height + round_idx, validators, genesis)`.
    """
    norm_validators = normalize_validator_set(validators)
    evidence: List[Evidence] = []
    reports: List[RoundReport] = []
    for r in range(max_rounds):
        proposer = proposer_for_round_fn(r)
        if not proposer:
            break
        report = run_round(
            height=height,
            round_idx=r,
            proposer=proposer,
            validators=norm_validators,
            fault_model=fault_model,
            evidence_sink=evidence,
        )
        reports.append(report)
        if report.decision == Decision.COMMIT:
            break
    return reports, evidence


def slashing_amount(power: float, lzn_frozen_mining: float, fraction: float = 0.1) -> float:
    """Сколько LZN снять у нарушителя за double_sign.

    По §5.4: часть → burn, часть → казна; здесь возвращаем суммарную сумму
    штрафа (доля от `lzn_frozen_mining`).
    """
    if lzn_frozen_mining <= 0:
        return 0.0
    return max(0.01, min(lzn_frozen_mining, lzn_frozen_mining * fraction))
