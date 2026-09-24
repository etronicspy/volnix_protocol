"""Reference AutoDeclare for traffic bot validators (§5.4)."""

from __future__ import annotations

import logging
from fractions import Fraction
from typing import Any, Dict, Optional, Set, Tuple

from volnix_traffic import actions
from volnix_traffic.client import NodeClient
from volnix_traffic.registry import BotRegistry

log = logging.getLogger("volnix_traffic.declare")


def entry_burn(l_i: int, alpha: Fraction) -> int:
    if l_i <= 0 or alpha.denominator <= 0:
        return 0
    return (alpha.numerator * l_i) // alpha.denominator


def suggest_declare_bs(l_i: int, ant: int, alpha: Fraction) -> Optional[Tuple[int, int]]:
    """Mirror node povb.suggest_declare_bs (§6.3(5) proportions)."""
    f_i = entry_burn(l_i, alpha)
    if l_i <= 0 or ant < f_i + 1:
        return None
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


def suggest_lambda_declare(
    l_i: int, ant: int, lam: Fraction, alpha: Fraction
) -> Optional[Tuple[int, int]]:
    """b_i ≈ λ·L_i with residual s_i so f+b+s fits under L_i and ant."""
    f_i = entry_burn(l_i, alpha)
    if l_i <= 0 or ant <= f_i:
        return None
    b_i = (lam.numerator * l_i) // lam.denominator
    if b_i <= 0:
        b_i = 1
    budget = ant - f_i
    b_i = min(b_i, budget - 1) if budget > 1 else 0
    s_i = min(budget - b_i, max(1, l_i - f_i - b_i))
    if s_i <= 0 or b_i < 0:
        return suggest_declare_bs(l_i, ant, alpha)
    if f_i + b_i + s_i > l_i:
        return suggest_declare_bs(l_i, ant, alpha)
    return b_i, s_i


class AutoDeclare:
    async def step(
        self,
        client: NodeClient,
        registry: BotRegistry,
        *,
        accounts_by_addr: Dict[str, Dict[str, Any]],
        lambda_num: int = 1,
        lambda_den: int = 3,
        alpha_num: int = 1,
        alpha_den: int = 50,
        pending_signers: Optional[Set[str]] = None,
    ) -> int:
        lam = Fraction(lambda_num, lambda_den)
        alpha = Fraction(alpha_num, alpha_den)
        pending = pending_signers or set()
        submitted = 0
        for bot in registry.validators():
            if bot.address in pending:
                continue
            row = accounts_by_addr.get(bot.address) or {}
            l_i = int(row.get("lzn_activated") or 0)
            ant = int(row.get("ant") or 0)
            if l_i <= 0:
                continue
            pair = suggest_lambda_declare(l_i, ant, lam, alpha)
            if pair is None:
                continue
            b_i, s_i = pair
            if await actions.declare(client, bot, b_i, s_i):
                submitted += 1
                pending.add(bot.address)
        return submitted
