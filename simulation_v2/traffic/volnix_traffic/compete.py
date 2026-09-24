"""Competitive (b_i, s_i) picker against rival declares (canon §5.4 steps 2–6)."""

from __future__ import annotations

import math
from dataclasses import dataclass
from fractions import Fraction
from typing import List, Optional, Sequence, Tuple


@dataclass(frozen=True)
class Rival:
    address: str
    l_i: int
    b_i: int
    s_i: int


@dataclass(frozen=True)
class DeclarePick:
    b_i: int
    s_i: int
    enter: bool
    expected_wrt: int
    f_i: int


def entry_burn(l_i: int, alpha: Fraction) -> int:
    if l_i <= 0 or alpha.denominator <= 0:
        return 0
    return (alpha.numerator * l_i) // alpha.denominator


def corridor_bounds(l_decl: int, lam: Fraction) -> Tuple[int, int]:
    p, q = lam.numerator, lam.denominator
    if l_decl <= 0 or q <= 0:
        return 0, 0
    b_min = math.ceil(p * l_decl / q)
    b_max = (q - p) * l_decl // q
    return b_min, b_max


def _priority_key(rec: Tuple[str, int, int, int, int, float]) -> Tuple[float, str]:
    # rec: address, l_i, b_i, s_i, f_i, w_i
    return (-rec[5], rec[0])


def simulate_enter(
    address: str,
    b_i: int,
    s_i: int,
    l_i: int,
    rivals: Sequence[Rival],
    alpha: Fraction,
    lam: Fraction,
    k: int,
) -> Tuple[bool, int]:
    """Return (self entered applied set, B = Σ b of applied set)."""
    if l_i <= 0 or s_i <= 0:
        return False, 0
    f_self = entry_burn(l_i, alpha)
    if f_self + b_i + s_i > l_i:
        return False, 0

    recs: List[Tuple[str, int, int, int, int, float]] = [
        (address, l_i, b_i, s_i, f_self, s_i / l_i)
    ]
    for r in rivals:
        if r.address == address or r.l_i <= 0 or r.s_i <= 0:
            continue
        f_j = entry_burn(r.l_i, alpha)
        if f_j + r.b_i + r.s_i > r.l_i:
            continue
        recs.append((r.address, r.l_i, r.b_i, r.s_i, f_j, r.s_i / r.l_i))

    l_decl = sum(x[1] for x in recs)
    b_min, b_max = corridor_bounds(l_decl, lam)
    remaining = sorted(recs, key=_priority_key)

    while remaining and sum(x[2] for x in remaining) > b_max:
        remaining.pop()

    f_all = sum(x[4] for x in recs)

    def fill(group: Sequence[Tuple[str, int, int, int, int, float]]) -> int:
        return f_all + sum(x[2] + x[3] for x in group)

    while remaining and fill(remaining) > l_decl:
        remaining.pop()

    if k > 0 and len(remaining) > k:
        remaining = remaining[:k]

    sum_b = sum(x[2] for x in remaining)
    set_updated = bool(remaining) and sum_b >= b_min
    if not set_updated:
        return False, 0
    if address not in {x[0] for x in remaining}:
        return False, sum_b
    return True, sum_b


def _candidate_pairs(
    l_i: int,
    ant: int,
    alpha: Fraction,
    b_frac: float,
    s_frac: float,
    rival_s: Optional[Sequence[int]] = None,
) -> List[Tuple[int, int]]:
    f_i = entry_burn(l_i, alpha)
    residual = min(l_i - f_i, ant - f_i)
    if residual < 1:
        return []
    pairs: List[Tuple[int, int]] = []
    want_s = max(1, int(l_i * s_frac))
    want_b = max(0, int(l_i * b_frac))
    if want_s + want_b > residual:
        want_s = max(1, min(want_s, residual))
        want_b = residual - want_s
    pairs.append((want_b, want_s))

    extra_s = [1, residual]
    for s_j in rival_s or ():
        if 1 <= s_j < residual:
            extra_s.append(s_j)
            extra_s.append(min(residual, s_j + 1))
    step = max(1, residual // 80)
    for s_i in list(range(1, residual + 1, step)) + extra_s:
        pairs.append((residual - s_i, s_i))
    pairs.append((max(0, residual - 1), 1))
    if residual >= 1:
        pairs.append((0, residual))

    seen = set()
    out: List[Tuple[int, int]] = []
    for b_i, s_i in pairs:
        if s_i <= 0 or b_i < 0:
            continue
        if f_i + b_i + s_i > l_i:
            continue
        key = (b_i, s_i)
        if key in seen:
            continue
        seen.add(key)
        out.append(key)
    return out


def pick_declare(
    *,
    address: str,
    l_i: int,
    ant: int,
    rivals: Sequence[Rival],
    alpha: Fraction,
    lam: Fraction,
    k: int,
    subsidy: int,
    fees: int,
    ant_price: int,
    b_frac: float = 0.50,
    s_frac: float = 0.40,
) -> Optional[DeclarePick]:
    """Choose (b_i, s_i): enter the set first, then maximize net WRT share."""
    f_i = entry_burn(l_i, alpha)
    if l_i <= 0 or ant < f_i + 1:
        return None
    pairs = _candidate_pairs(
        l_i,
        ant,
        alpha,
        b_frac,
        s_frac,
        rival_s=[r.s_i for r in rivals],
    )
    if not pairs:
        return None

    entered: List[DeclarePick] = []
    fallback: Optional[DeclarePick] = None
    best_s = -1
    for b_i, s_i in pairs:
        enter, b_sum = simulate_enter(address, b_i, s_i, l_i, rivals, alpha, lam, k)
        reward = 0
        if enter and b_sum > 0:
            reward = (int(subsidy) + int(fees)) * b_i // b_sum
        burned = f_i + (b_i + s_i if enter else 0)
        net = reward - int(ant_price) * burned
        pick = DeclarePick(b_i=b_i, s_i=s_i, enter=enter, expected_wrt=net, f_i=f_i)
        if enter:
            entered.append(pick)
        elif s_i > best_s:
            best_s = s_i
            fallback = pick

    if entered:
        return max(entered, key=lambda p: (p.expected_wrt, p.b_i, p.s_i))
    return fallback
