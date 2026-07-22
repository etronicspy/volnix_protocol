"""
Проверки для кошелька до попадания tx в мемпул (должны совпадать с правилами engine).
"""
from __future__ import annotations

import time
import uuid
from typing import Optional, Tuple

from core.models import OrderType, Role, Transaction, TransactionType
from core.state import (
    GENESIS_PROVIDER_ADDR,
    GENESIS_VALIDATOR_ADDR,
    LZN_MAX_FROZEN_PER_ADDRESS,
    SIM_TREASURY_ADDR,
    StateManager,
    eligible_for_provider_role,
    eligible_for_validator_role,
)


def validate_treasury_mint(
    sm: StateManager,
    receiver_address: str,
    amount: float,
    asset: str,
) -> Tuple[bool, str, Optional[Transaction]]:
    """Эмиссия из sim-казначейства только через tx mint в мемпуле (списание в блоке)."""
    if amount is None or amount <= 0:
        return False, "amount must be positive", None
    tr = sm.accounts.get(SIM_TREASURY_ADDR)
    if not tr:
        return False, "Simulation treasury not initialized", None
    addr = receiver_address.strip()
    if not addr:
        return False, "receiver address is required", None
    recv = sm.accounts.get(addr)
    if not recv:
        recv = sm.create_account(addr)
    a = (asset or "wrt").lower()
    if a not in ("wrt", "lzn", "ant"):
        return False, "asset must be wrt, lzn, or ant", None
    if a == "ant" and recv.role not in (Role.PROVIDER, Role.VALIDATOR):
        return False, "ANT только у Поставщика и Валидатора (§4.1–4.2); у Гражданина (тип 1) — нет", None
    if a == "wrt" and tr.wrt_balance + 1e-12 < amount:
        return False, "Simulation treasury has insufficient WRT", None
    if a == "lzn" and tr.lzn_balance + 1e-12 < amount:
        return False, "Simulation treasury has insufficient LZN", None
    if a == "ant" and tr.ant_balance + 1e-12 < amount:
        return False, "Simulation treasury has insufficient ANT", None
    tx = Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.MINT,
        sender=SIM_TREASURY_ADDR,
        receiver=addr,
        amount=float(amount),
        asset_type=a,
        timestamp=time.time(),
    )
    return True, "queued for next block", tx


def _acc(sm: StateManager, addr: str):
    return sm.accounts.get(addr)


def validate_and_build_tx(
    sm: StateManager,
    op: str,
    address: str,
    *,
    to_address: Optional[str] = None,
    amount: Optional[float] = None,
    asset: str = "wrt",
    role: Optional[Role] = None,
    price: Optional[float] = None,
    order_id: Optional[str] = None,
    side: Optional[str] = None,
    burn_b: Optional[float] = None,
    stake_s: Optional[float] = None,
    market: bool = False,
    max_wrt: Optional[float] = None,
) -> Tuple[bool, str, Optional[Transaction]]:
    if not address or not address.strip():
        return False, "Address is required", None
    if address == SIM_TREASURY_ADDR:
        return False, "Cannot submit transactions from simulation treasury address", None

    ts = time.time()
    # Важно: кошелёк — это "подача предложения в блок" (mempool admission).
    # Узел проверяет канон и валидность при сборке блока (engine DeliverTx/EndBlock) и пишет отклонения в канон-аудит.
    acc = _acc(sm, address)

    op = op.strip().lower()

    if op == "set_role":
        if role is None:
            return False, "role is required", None
        new_role = role
        if new_role == acc.role:
            return False, "Already has this role", None
        # Важно: set_role всегда принимаем в мемпул как "предложение в блок".
        # Канонические проверки (ZKP/LZN/genesis-исключения) выполняются в DeliverTx симулятора (engine),
        # и там же записываются отклонения в канон-аудит.
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.SET_ROLE,
            sender=address,
            receiver=address,
            role=new_role,
            timestamp=ts,
        )
        return True, "accepted (will be validated in next block)", tx

    if op == "verify_zkp":
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.ZKP_VERIFY,
            sender=address,
            receiver=address,
            timestamp=ts,
        )
        return True, "accepted (will be validated in next block)", tx

    if op == "transfer":
        if not to_address or not to_address.strip():
            return False, "to_address is required", None
        if to_address == address:
            return False, "Cannot transfer to self", None
        if amount is None or amount <= 0:
            return False, "amount must be positive", None
        a = (asset or "wrt").lower()
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.TRANSFER,
            sender=address,
            receiver=to_address.strip(),
            amount=float(amount),
            asset_type=a,
            timestamp=ts,
        )
        return True, "accepted (will be validated in next block)", tx

    if op == "activate_lzn":
        if amount is None or amount <= 0:
            return False, "amount must be positive", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.ACTIVATE_LZN,
            sender=address,
            amount=float(amount),
            asset_type="lzn",
            timestamp=ts,
        )
        return True, "accepted (will be validated in next block)", tx

    if op == "create_order":
        if side not in ("buy", "sell"):
            return False, "side must be buy or sell", None
        ot = OrderType.BUY if side == "buy" else OrderType.SELL

        if market:
            if amount is None or amount <= 0:
                return False, "amount must be positive", None
            cap = None
            if max_wrt is not None:
                try:
                    cap = float(max_wrt)
                except (TypeError, ValueError):
                    return False, "max_wrt must be a number", None
                if cap <= 0:
                    return False, "max_wrt must be positive", None
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.CREATE_ORDER,
                    sender=address,
                    order_type=ot,
                    price=0.0,
                    amount=float(amount),
                    market=True,
                max_wrt=cap,
                    timestamp=ts,
                )
            return True, "accepted (will be validated in next block)", tx

        if price is None or price <= 0 or amount is None or amount <= 0:
            return False, "price and amount must be positive", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.CREATE_ORDER,
            sender=address,
            order_type=ot,
            price=float(price),
            amount=float(amount),
            timestamp=ts,
        )
        return True, "accepted (will be validated in next block)", tx

    if op == "cancel_order":
        if not order_id:
            return False, "order_id is required", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.CANCEL_ORDER,
            sender=address,
            order_id=order_id,
            timestamp=ts,
        )
        return True, "accepted (will be validated in next block)", tx

    if op == "declare":
        b = float(burn_b or 0)
        s = float(stake_s or 0)
        if b < 0 or s < 0:
            return False, "burn_b and stake_s must be non-negative", None
        if b + s <= 0:
            return False, "b + s must be positive", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.DECLARE_PARTICIPATION,
            sender=address,
            amount=b,
            stake_amount=s,
            asset_type="ant",
            timestamp=ts,
        )
        return True, "accepted (will be validated in next block)", tx

    return False, f"Unknown op: {op}", None
