"""
Проверки для кошелька до попадания tx в мемпул (должны совпадать с правилами engine).
"""
from __future__ import annotations

import time
import uuid
from typing import Optional, Tuple

from core.models import OrderType, Role, Transaction, TransactionType
from core.state import (
    StateManager,
    LZN_MAX_FROZEN_PER_ADDRESS,
    SIM_TREASURY_ADDR,
    GENESIS_PROVIDER_ADDR,
    GENESIS_VALIDATOR_ADDR,
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
        return False, "ANT только у ролей Поставщик и Валидатор (§4.1–4.2); Citizen/ Guest — нет", None
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
    acc = _acc(sm, address)
    if not acc:
        return False, "Unknown address: create an account first (God Mode) or receive a transfer", None

    op = op.strip().lower()

    if op == "set_role":
        if role is None:
            return False, "role is required", None
        new_role = role
        if new_role == acc.role:
            return False, "Already has this role", None
        if new_role == Role.VALIDATOR:
            if not eligible_for_validator_role(address, acc):
                if address != GENESIS_VALIDATOR_ADDR and not acc.zkp_verified:
                    return (
                        False,
                        "Валидатор: сначала ZKP в панели кошелька (tx verify_zkp), затем нужен LZN",
                        None,
                    )
                return False, "Валидатор: нужен хотя бы 1 LZN (ликвидный или активированный)", None
        if new_role == Role.PROVIDER and acc.role == Role.GUEST:
            return False, "Guest cannot become Provider: become Citizen first (on-chain set_role)", None
        if new_role == Role.PROVIDER:
            if not eligible_for_provider_role(address, acc):
                if address != GENESIS_PROVIDER_ADDR and not acc.zkp_verified:
                    return (
                        False,
                        "Поставщик: сначала ZKP в панели кошелька (tx verify_zkp), затем нужен LZN",
                        None,
                    )
                return False, "Поставщик: нужен хотя бы 1 LZN (ликвидный или активированный)", None
        if new_role == Role.GUEST and acc.role == Role.GUEST:
            return False, "Already a Guest", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.SET_ROLE,
            sender=address,
            receiver=address,
            role=new_role,
            timestamp=ts,
        )
        return True, "accepted", tx

    if op == "verify_zkp":
        if acc.zkp_verified:
            return False, "ZKP уже подтверждён для этого адреса", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.ZKP_VERIFY,
            sender=address,
            receiver=address,
            timestamp=ts,
        )
        return True, "accepted", tx

    if op == "transfer":
        if not to_address or not to_address.strip():
            return False, "to_address is required", None
        if to_address == address:
            return False, "Cannot transfer to self", None
        if amount is None or amount <= 0:
            return False, "amount must be positive", None
        a = (asset or "wrt").lower()
        if a not in ("wrt", "lzn"):
            return False, "Only wrt and lzn transfers are allowed (ANT only via internal market)", None
        if a == "lzn" and acc.lzn_balance + 1e-12 < amount:
            return False, "Insufficient liquid LZN", None
        if a == "wrt" and acc.wrt_balance + 1e-12 < amount:
            return False, "Insufficient WRT", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.TRANSFER,
            sender=address,
            receiver=to_address.strip(),
            amount=float(amount),
            asset_type=a,
            timestamp=ts,
        )
        return True, "accepted", tx

    if op == "activate_lzn":
        if acc.role != Role.VALIDATOR:
            return False, "Only Validators can activate LZN for mining", None
        if amount is None or amount <= 0:
            return False, "amount must be positive", None
        if acc.lzn_balance + 1e-12 < amount:
            return False, "Insufficient liquid LZN", None
        if acc.lzn_frozen_mining + float(amount) > float(LZN_MAX_FROZEN_PER_ADDRESS) + 1e-9:
            return (
                False,
                f"Activated LZN per address cap exceeded (max {LZN_MAX_FROZEN_PER_ADDRESS:.0f} per §4.2)",
                None,
            )
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.ACTIVATE_LZN,
            sender=address,
            amount=float(amount),
            asset_type="lzn",
            timestamp=ts,
        )
        return True, "accepted", tx

    if op == "create_order":
        if acc.role == Role.GUEST:
            return False, "Guests cannot trade on the internal market", None
        if acc.role == Role.CITIZEN:
            return False, "Citizen не торгует ANT на внутреннем рынке (§4.2: только Поставщик/Валидатор)", None
        if side not in ("buy", "sell"):
            return False, "side must be buy or sell", None
        ot = OrderType.BUY if side == "buy" else OrderType.SELL

        if market:
            if amount is None or amount <= 0:
                return False, "amount must be positive", None
            if ot == OrderType.BUY:
                if acc.role != Role.VALIDATOR:
                    return False, "Only Validators may buy ANT (§5.2)", None
                asks = [o for o in sm.orders.values() if o.order_type == OrderType.SELL]
                if not asks:
                    return False, "Рыночная покупка: в книге нет заявок на продажу", None
                cap = float(max_wrt) if max_wrt is not None else acc.wrt_balance
                if cap <= 0:
                    return False, "Укажите max_wrt > 0 или пополните WRT", None
                if acc.wrt_balance + 1e-12 < cap:
                    return False, "Insufficient WRT", None
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
                return True, "accepted", tx
            if acc.role != Role.PROVIDER:
                return False, "Only Providers may sell ANT (§5.2)", None
            if acc.ant_balance + 1e-12 < amount:
                return False, "Insufficient ANT", None
            bids = [o for o in sm.orders.values() if o.order_type == OrderType.BUY]
            if not bids:
                return False, "Рыночная продажа: в книге нет заявок на покупку", None
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.CREATE_ORDER,
                sender=address,
                order_type=ot,
                price=0.0,
                amount=float(amount),
                market=True,
                timestamp=ts,
            )
            return True, "accepted", tx

        if price is None or price <= 0 or amount is None or amount <= 0:
            return False, "price and amount must be positive", None
        if ot == OrderType.SELL:
            if acc.role != Role.PROVIDER:
                return False, "Only Providers may place SELL orders (§5.2)", None
            if acc.ant_balance + 1e-12 < amount:
                return False, "Insufficient ANT (not in escrow)", None
        else:
            if acc.role != Role.VALIDATOR:
                return False, "Only Validators may place BUY orders — спрос на ANT (§5.2)", None
            cost = float(price) * float(amount)
            if acc.wrt_balance + 1e-12 < cost:
                return False, "Insufficient WRT for buy order escrow", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.CREATE_ORDER,
            sender=address,
            order_type=ot,
            price=float(price),
            amount=float(amount),
            timestamp=ts,
        )
        return True, "accepted", tx

    if op == "cancel_order":
        if not order_id:
            return False, "order_id is required", None
        order = sm.orders.get(order_id)
        if not order or order.owner != address:
            return False, "Order not found or not owned by this address", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.CANCEL_ORDER,
            sender=address,
            order_id=order_id,
            timestamp=ts,
        )
        return True, "accepted", tx

    if op == "declare":
        if acc.role != Role.VALIDATOR:
            return False, "Only Validators can declare participation (§5.4)", None
        b = float(burn_b or 0)
        s = float(stake_s or 0)
        if b < 0 or s < 0:
            return False, "burn_b and stake_s must be non-negative", None
        L_i = float(acc.lzn_frozen_mining)
        if L_i <= 0:
            return False, "No activated LZN (activate LZN first)", None
        if b + s > L_i + 1e-9:
            return False, f"b + s must not exceed activated LZN ({L_i})", None
        if b + s <= 0:
            return False, "b + s must be positive", None
        if acc.ant_balance + 1e-12 < b + s:
            return False, "Insufficient ANT for burn + stake", None
        tx = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.DECLARE_PARTICIPATION,
            sender=address,
            amount=b,
            stake_amount=s,
            asset_type="ant",
            timestamp=ts,
        )
        return True, "accepted", tx

    return False, f"Unknown op: {op}", None
