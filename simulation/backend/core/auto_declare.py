"""AutoDeclareDaemon — эталонный клиент §5.4: declare перед каждой высотой.

Коридор: λ·L_total ≤ Σb_i ≤ (1−λ)·L_total. Без declare валидатор не получает
ни базовой WRT, ни доли комиссий (b_i = 0 → доход = 0). Демон реализует
эталонную стратегию: каждый валидатор объявляет b_i = λ·L_i — тогда
Σb_i = λ·L_total (нижняя граница коридора).

В lifespan вызывается только из ``engine.pre_block_hook`` (один раз на попытку
блока). Подача идёт через ``StateManager.submit_tx`` (replace-by-sender).
Флаг ``VOLNIX_SIM_AUTO_DECLARE=false`` отключает подачу в pre_block.
"""
from __future__ import annotations

import asyncio
from typing import Optional

from core.canon_audit import log_wallet_rejection
from core.engine import BURN_CAP_LAMBDA, SimulationEngine
from core.models import Role
from core.state import StateManager
from core.wallet_validate import validate_and_build_tx

# Минимальный шаг подачи declare; чаще, чем block_time/2, не имеет смысла
# (всё равно block_time/2 — это нижний предел частоты relevant изменений).
MIN_TICK_SEC = 0.5


def _submit_declare(
    sm: StateManager,
    address: str,
    burn_b: float,
    stake_s: float,
) -> bool:
    ok, msg, tx = validate_and_build_tx(
        sm, "declare", address, burn_b=float(burn_b), stake_s=float(stake_s)
    )
    if not ok or tx is None:
        log_wallet_rejection(sm, "declare", f"auto_declare: {msg}", address)
        return False
    sm.submit_tx(tx)
    return True


def _pending_declare_addrs(sm: StateManager) -> set[str]:
    """Адреса, у которых уже есть declare в мемпуле — не дублируем."""
    out: set[str] = set()
    net = getattr(sm, "network", None)
    if net is not None:
        for tx in net.iter_pending_txs():
            if (
                tx.tx_type.value == "declare_participation"
                and tx.sender
            ):
                out.add(tx.sender)
    for tx in sm.mempool:
        if tx.tx_type.value == "declare_participation" and tx.sender:
            out.add(tx.sender)
    return out


def step_once(sm: StateManager, engine: SimulationEngine) -> int:
    """Один проход: вернуть число поданных declare за тик.

    §5.4 v2 (эталонная стратегия): каждому валидатору `b_i = λ·L_i`
    (распределение пропорционально собственному L_i), тогда
    Σb_i = λ·Σ L_i = λ·L_total — ровно верхний предел, отсев не срабатывает.

    Кандидаты — **все** валидаторы с активированным LZN, а не только текущий
    `consensus_validator_set`: набор пересобирается из participation прошлого
    блока, и обход по нему замыкал круг (кто не в наборе — тому не подаётся
    declare, значит он никогда в набор и не попадёт). ANT — «электричество»
    §5.4: его жжёт каждый валидатор с «оборудованием» (активированный LZN).
    При нехватке ANT на полный `λ·L_i` объявляется остаток баланса — участие
    в блоке важнее максимального b_i (при b_i = 0 дохода нет вовсе).

    Клиентская дисциплина: если declare уже в мемпуле — не слать повторно
    (mempool всё равно replace-by-sender; skip экономит работу).
    """
    pending = _pending_declare_addrs(sm)
    submitted = 0
    for addr, acc in sorted(sm.accounts.items()):
        if addr in pending or acc.role != Role.VALIDATOR:
            continue
        L_i = float(acc.lzn_frozen_mining)
        if L_i <= 0:
            continue
        # b_i = λ·L_i; s_i=0 (демон не делает stake, чтобы не блокировать ANT).
        # Без round(..., 6) вниз — иначе 0.333333 < λ·1 и низ коридора не набирается.
        b_i = min(BURN_CAP_LAMBDA * L_i, float(acc.ant_balance))
        if b_i <= 1e-12:
            # Нет ANT — «электричество» кончилось, валидатор простаивает (§5.1: доход 0).
            continue
        if _submit_declare(sm, addr, b_i, 0.0):
            submitted += 1
            pending.add(addr)
    return submitted


async def run(
    sm: StateManager,
    engine: SimulationEngine,
    *,
    stop_event: Optional[asyncio.Event] = None,
) -> None:
    """Устарело для lifespan: declare идёт из ``engine.pre_block_hook``.

    Оставлено для ручных/тестовых сценариев (тик по wall-clock).
    """
    while True:
        if stop_event is not None and stop_event.is_set():
            return
        try:
            step_once(sm, engine)
        except Exception as e:  # pragma: no cover
            print(f"AutoDeclareDaemon tick error: {e}")
        delay = max(MIN_TICK_SEC, float(engine.block_time) / 2.0)
        try:
            await asyncio.sleep(delay)
        except asyncio.CancelledError:
            return
