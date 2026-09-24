"""PoVB EndBlocker — canon §5.4 (5.2-sim) steps 1–7."""

from __future__ import annotations

import math
from fractions import Fraction
from typing import Any, Optional

from volnix.app.modules.bank import BankError, burn
from volnix.app.state import AppState, DeclareRecord
from volnix.types.events import Event, ev
from volnix.types.msgs import MsgDeclareParticipation
from volnix.types.role import Role
from volnix.types.validator import Validator, ValidatorSet, voting_power


class PovbError(ValueError):
    pass


def entry_burn(l_i: int, alpha: Fraction) -> int:
    """f_i = ⌊α · L_i⌋ = ⌊p · L_i / q⌋ (DeclareEntryAlpha, canon §5.4 / §7.2 п. 13)."""
    if l_i <= 0 or alpha.denominator <= 0:
        return 0
    return (alpha.numerator * l_i) // alpha.denominator


def suggest_declare_bs(l_i: int, ant: int, alpha: Fraction) -> Optional[tuple[int, int]]:
    """Pick (b_i, s_i) for a stand auto-declare using §6.3(5) proportions.

    Reference at L_i = 10^6: b_i = 500_000, s_i = 400_000 (f_i = 20_000).
    Returns None when fuel cannot cover f_i + at least s_i = 1.
    """
    f_i = entry_burn(l_i, alpha)
    if l_i <= 0 or ant < f_i + 1:
        return None
    # Scale §6.3(5): b = L/2, s = 2L/5
    b_i = l_i // 2
    s_i = (2 * l_i) // 5
    if s_i <= 0:
        s_i = 1
    if f_i + b_i + s_i > l_i:
        b_i = max(0, l_i - f_i - s_i)
    need = f_i + b_i + s_i
    if need > ant:
        budget = ant - f_i
        if budget < 1:
            return None
        # Keep ~5:4 b:s while spending all available fuel under the cap.
        b_i = min(b_i, (budget * 5) // 9)
        s_i = budget - b_i
        if s_i <= 0:
            s_i = 1
            b_i = budget - 1
        if b_i < 0:
            return None
    if f_i + b_i + s_i > l_i or s_i <= 0:
        return None
    return b_i, s_i


def corridor_bounds(l_decl: int, lam: Fraction) -> tuple[int, int]:
    """B_min = ⌈p · L_decl / q⌉, B_max = ⌊(q − p) · L_decl / q⌋ for λ = p/q."""
    p, q = lam.numerator, lam.denominator
    if l_decl <= 0 or q <= 0:
        return 0, 0
    b_min = math.ceil(p * l_decl / q)
    b_max = (q - p) * l_decl // q
    return b_min, b_max


def deliver_declare(state: AppState, msg: MsgDeclareParticipation) -> list[Event]:
    acc = state.accounts.get(msg.validator)
    if acc is None or acc.role != Role.VALIDATOR:
        raise PovbError("only a validator may declare")
    if msg.b_i < 0 or msg.s_i < 0:
        raise PovbError("b_i and s_i must be non-negative")
    if msg.s_i == 0:
        raise PovbError("s_i must be > 0")
    state.declares[msg.validator] = DeclareRecord(
        validator=msg.validator, b_i=msg.b_i, s_i=msg.s_i
    )
    return [
        ev(
            "consensus.declare",
            validator=msg.validator,
            b_i=msg.b_i,
            s_i=msg.s_i,
            block_height=state.height,
        )
    ]


def _priority_key(rec: DeclareRecord) -> tuple[float, str]:
    """Head = highest priority: w_i descending, then address ascending."""
    return (-rec.w_i, rec.validator)


def process_endblocker(state: AppState, current_set: ValidatorSet) -> tuple[ValidatorSet, list[Event], dict[str, Any]]:
    """Run §5.4 steps 1–7. Returns (next_set, events, povb_trace).

    Sim note: b_i+s_i burn on EndBlocker N when set_updated (applied for N+1),
    equivalent when every produced block commits.
    """
    events: list[Event] = []
    params = state.params
    alpha = params.alpha
    lam = params.lambda_
    l_total = state.l_total()
    k = params.max_active_validators

    # Step 1: validate declares
    candidates: list[DeclareRecord] = []
    rejected: list[DeclareRecord] = []
    for rec in state.declares.values():
        acc = state.accounts.get(rec.validator)
        if acc is None or acc.role != Role.VALIDATOR:
            rec.valid = False
            rec.reason = "not_validator"
            rejected.append(rec)
            continue
        rec.l_i = acc.lzn_activated
        rec.f_i = entry_burn(rec.l_i, alpha)
        if rec.l_i <= 0:
            rec.valid = False
            rec.reason = "no_activated_lzn"
            rejected.append(rec)
            continue
        if rec.s_i <= 0:
            rec.valid = False
            rec.reason = "s_i_zero"
            rejected.append(rec)
            continue
        if rec.f_i + rec.b_i + rec.s_i > rec.l_i:
            rec.valid = False
            rec.reason = "f+b+s_exceeds_L"
            rejected.append(rec)
            continue
        if acc.ant < rec.f_i + rec.b_i + rec.s_i:
            rec.valid = False
            rec.reason = "insufficient_ant"
            rejected.append(rec)
            continue
        rec.w_i = rec.s_i / rec.l_i
        rec.valid = True
        candidates.append(rec)

    # Step 2: fix L_decl once (not recomputed on cull)
    l_decl = sum(r.l_i for r in candidates)
    b_min, b_max = corridor_bounds(l_decl, lam)

    def fill_of(group: list[DeclareRecord]) -> int:
        f_sum = sum(r.f_i for r in candidates)
        rest = sum(r.b_i + r.s_i for r in group)
        return f_sum + rest

    # Priority order: head = highest w_i
    remaining = sorted(candidates, key=_priority_key)

    # Step 3: λ upper — exclude from tail
    while remaining and sum(r.b_i for r in remaining) > b_max:
        victim = remaining.pop()  # tail
        victim.excluded = "lambda"
        victim.passed = False

    # Step 4: Fill ≤ L_decl — exclude from tail
    while remaining and fill_of(remaining) > l_decl:
        victim = remaining.pop()
        victim.excluded = "fill"
        victim.passed = False

    # Step 5: top-K from head
    if len(remaining) > k:
        for extra in remaining[k:]:
            extra.excluded = "topk"
            extra.passed = False
        remaining = remaining[:k]

    # Step 6: floor on final composition
    sum_b = sum(r.b_i for r in remaining)
    set_updated = bool(remaining) and sum_b >= b_min

    for rec in remaining:
        rec.passed = set_updated

    # f_i burns for all valid declares regardless of set update
    for rec in candidates:
        try:
            burn(state, rec.validator, "uant", rec.f_i)
            events.append(
                ev("consensus.burn_executed", validator=rec.validator, kind="f_i", amount=rec.f_i)
            )
        except BankError:
            rec.valid = False
            rec.reason = "burn_f_failed"

    # b+s only when set applied (sim: EndBlocker N ≡ success path for N+1)
    if set_updated:
        for rec in remaining:
            try:
                burn(state, rec.validator, "uant", rec.b_i + rec.s_i)
                events.append(
                    ev(
                        "consensus.burn_executed",
                        validator=rec.validator,
                        kind="b_s",
                        amount=rec.b_i + rec.s_i,
                    )
                )
            except BankError:
                rec.passed = False
                rec.reason = "burn_bs_failed"

    # Step 7: fix ValidatorSet
    next_set = current_set.copy()
    if set_updated:
        vals: list[Validator] = []
        for rec in remaining:
            acc = state.accounts[rec.validator]
            vals.append(
                Validator(
                    address=rec.validator,
                    pub_hex=acc.pub_hex,
                    l_i=rec.l_i,
                    s_i=rec.s_i,
                    w_i=rec.w_i,
                    power=voting_power(rec.s_i, rec.l_i),
                    proposer_priority=0,
                )
            )
        prev = current_set.by_address()
        for v in vals:
            if v.address in prev:
                v.proposer_priority = prev[v.address].proposer_priority
        next_set = ValidatorSet(validators=vals, proposer=current_set.proposer)

    state.passed_validators = [r.validator for r in remaining if r.passed]
    state.set_updated = set_updated

    fill_final = fill_of(remaining) if remaining else sum(r.f_i for r in candidates)
    trace: dict[str, Any] = {
        "height": state.height,
        "l_total": l_total,
        "l_decl": l_decl,
        "lambda": f"{params.lambda_num}/{params.lambda_den}",
        "alpha": f"{params.alpha_num}/{params.alpha_den}",
        "b_min": b_min,
        "b_max": b_max,
        "upper": b_max,
        "lower": b_min,
        "sum_b": sum_b,
        "fill": fill_final,
        "k": k,
        "set_updated": set_updated,
        "declares": [
            {
                "validator": r.validator,
                "b_i": r.b_i,
                "s_i": r.s_i,
                "f_i": r.f_i,
                "l_i": r.l_i,
                "w_i": r.w_i,
                "valid": r.valid,
                "reason": r.reason,
                "passed": r.passed,
                "excluded": r.excluded,
            }
            for r in list(candidates) + rejected
        ],
        "passed": [r.validator for r in remaining if r.passed],
    }
    state.last_povb = trace
    events.append(
        ev(
            "consensus.per_height_burn",
            l_decl=l_decl,
            l_tot=l_total,
            lambda_=str(lam),
            fill=fill_final,
            set_updated=int(set_updated),
            passed=",".join(trace["passed"]),
        )
    )
    return next_set, events, trace
