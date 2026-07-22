"""AutoDeclareDaemon — фоновая корутина, подающая канонические declare §5.4.

Ruleset v2: блоки формируются при любом Σb_i ≤ λ·L_total (минимального порога
нет), но без declare валидатор не получает ни базовой WRT, ни доли комиссий
(b_i = 0 → доход = 0). Демон реализует эталонную стратегию клиента:
каждый валидатор объявляет b_i = λ·L_i — тогда Σb_i = λ·L_total, т.е. сеть
работает ровно на верхнем пределе дохода без внепротокольной координации.
В реальной цепи каждый валидатор сам подаёт `MsgDeclareParticipation`;
здесь симулируем этот процесс одной корутиной, которая итерируется по
`state.consensus_validator_set`.

Запускается из FastAPI lifespan; отключается флагом
`VOLNIX_SIM_AUTO_DECLARE=false` для тестов / ручной подачи.
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
    # Сначала пробуем через NetworkSim (если attached), иначе в общий мемпул.
    net = getattr(sm, "network", None)
    if net is not None:
        try:
            net.submit_from_addr(address, tx)
            return True
        except Exception:
            pass
    sm.mempool.append(tx)
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
    """
    vs = list(getattr(sm, "consensus_validator_set", []) or [])
    if not vs:
        return 0
    pending = _pending_declare_addrs(sm)
    submitted = 0
    for entry in vs:
        addr = entry.get("address") if isinstance(entry, dict) else None
        if not addr or addr in pending:
            continue
        acc = sm.accounts.get(addr)
        if not acc or acc.role != Role.VALIDATOR:
            continue
        L_i = float(acc.lzn_frozen_mining)
        if L_i <= 0:
            continue
        # b_i = λ·L_i; s_i=0 (демон не делает stake, чтобы не блокировать ANT)
        b_i = round(BURN_CAP_LAMBDA * L_i, 6)
        # Канон требует also ant >= b+s; используем существующий validate
        if acc.ant_balance + 1e-9 < b_i:
            # Не хватает ANT — пропускаем (бот / scenario должен пополнить).
            continue
        if _submit_declare(sm, addr, b_i, 0.0):
            submitted += 1
    return submitted


async def run(
    sm: StateManager,
    engine: SimulationEngine,
    *,
    stop_event: Optional[asyncio.Event] = None,
) -> None:
    """Бесконечный цикл: каждые `block_time/2` (но не реже MIN_TICK_SEC) — step_once.

    Прекращается по `stop_event.set()` или CancelledError.
    """
    while True:
        if stop_event is not None and stop_event.is_set():
            return
        try:
            step_once(sm, engine)
        except Exception as e:
            # Никогда не падаем: лог и продолжаем.
            print(f"AutoDeclareDaemon tick error: {e}")
        delay = max(MIN_TICK_SEC, float(engine.block_time) / 2.0)
        try:
            await asyncio.sleep(delay)
        except asyncio.CancelledError:
            return
