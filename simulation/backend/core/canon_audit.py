"""Пост-аудит записей ленты блока на соответствие канону (краткие выводы для панели логов)."""
from __future__ import annotations

from typing import TYPE_CHECKING, Any, Dict, List

if TYPE_CHECKING:
    from core.state import StateManager


SKIP_AUDIT = frozenset(
    {
        "begin_block",
        "end_block",
        "genesis_message",
        "genesis_validator_lzn",
        "genesis_lzn_activate",
        "genesis_validator_ant",
        "genesis_provider_ant",
        "genesis_market_seed",
    }
)


def audit_block_ledger(sm: "StateManager", txs: List[dict], block_height: int) -> None:
    log = sm.canon_log
    for item in txs:
        tx_type = (item.get("tx_type") or "").lower()
        if not tx_type or tx_type in SKIP_AUDIT:
            continue
        h = item.get("tx_hash") or ""
        if tx_type == "transfer":
            asset = (item.get("asset_type") or "wrt").lower()
            snd = (item.get("sender") or "")[:16]
            rcv = (item.get("receiver") or "")[:16]
            amt = item.get("amount")
            if asset in ("wrt", "lzn"):
                log.push(
                    source="engine",
                    status="ok",
                    category="transfer",
                    canon="§4.1",
                    title="Перевод ликвидного актива соответствует канону",
                    detail=f"{asset.upper()} {amt}: {snd}… → {rcv}… (прямой перевод ANT запрещён — только внутренний рынок).",
                    tx_hash=h,
                    block_height=block_height,
                )
            elif asset == "ant":
                log.push(
                    source="engine",
                    status="warn",
                    category="transfer",
                    canon="§4.1",
                    title="В ленту попал перевод ANT — по канону недопустим как MsgSend",
                    detail="Ожидалось: ANT только рынок / протокол. Проверьте источник tx.",
                    tx_hash=h,
                    block_height=block_height,
                )
        elif tx_type == "mint":
            asset = (item.get("asset_type") or "wrt").lower()
            rcv = item.get("receiver") or ""
            acc = sm.accounts.get(rcv)
            role = acc.role.value if acc else "?"
            if asset == "ant":
                ok_roles = role in ("provider", "validator")
                log.push(
                    source="engine",
                    status="ok" if ok_roles else "warn",
                    category="mint",
                    canon="§4.1–4.2",
                    title="Mint ANT: получатель Поставщик или Валидатор" if ok_roles else "Mint ANT: нетипичная роль получателя",
                    detail=f"Получатель роль={role}, amount={item.get('amount')}",
                    tx_hash=h,
                    block_height=block_height,
                )
            else:
                log.push(
                    source="engine",
                    status="ok",
                    category="mint",
                    canon="§4.1",
                    title=f"Mint {asset.upper()} из казны симуляции",
                    detail=f"→ {rcv[:20]}…",
                    tx_hash=h,
                    block_height=block_height,
                )
        elif tx_type == "trade":
            log.push(
                source="engine",
                status="ok",
                category="market",
                canon="§5.2",
                title="Сделка на внутреннем рынке ANT",
                detail=(
                    f"Покупатель (валидатор) {str(item.get('buyer',''))[:18]}… ↔ "
                    f"Продавец (поставщик) {str(item.get('seller',''))[:18]}… "
                    f"цена={item.get('price')} кол-во={item.get('amount')}"
                ),
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "create_order":
            ot = item.get("order_type") or item.get("orderType")
            market = bool(item.get("market"))
            log.push(
                source="engine",
                status="ok",
                category="order",
                canon="§5.2",
                title="Ордер / рыночная заявка приняты в блок",
                detail=f"side={ot}, market={market}, price={item.get('price')}, amount={item.get('amount')}",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "cancel_order":
            log.push(
                source="engine",
                status="ok",
                category="order",
                canon="§5.2",
                title="Отмена ордера (возврат эскроу)",
                detail=f"order_id={str(item.get('order_id',''))[:12]}…",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "set_role":
            log.push(
                source="engine",
                status="ok",
                category="role",
                canon="§4.2",
                title="Смена роли кошелька",
                detail=f"role={item.get('role')}",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "zkp_verify":
            log.push(
                source="engine",
                status="ok",
                category="identity",
                canon="§3.1",
                title="ZKP подтверждён (симуляция)",
                detail=f"addr={str(item.get('sender',''))[:20]}…",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "activate_lzn":
            log.push(
                source="engine",
                status="ok",
                category="lzn",
                canon="§4.2",
                title="Активация LZN только для Валидатора",
                detail=f"amount={item.get('amount')}",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "declare_participation":
            log.push(
                source="engine",
                status="ok",
                category="consensus",
                canon="§5.4",
                title="Declare участия: b_i (сжигание) + s_i (ставка)",
                detail=f"b={item.get('amount')} s={item.get('stake_amount')} (оба сжигаются при успешном блоке)",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "mempool_tx_dropped":
            log.push(
                source="engine",
                status="reject",
                category="delivertx",
                canon="—",
                title="Tx не обработана в DeliverTx симулятора",
                detail=str(item.get("details") or ""),
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "protocol_ant_burn":
            log.push(
                source="engine",
                status="ok",
                category="sanitize",
                canon="§4.1–4.2",
                title="Санитизация: ANT у роли «Гражданин» сжёжен",
                detail=item.get("details") or "",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "protocol_order_cancel":
            log.push(
                source="engine",
                status="ok",
                category="sanitize",
                canon="§4.2",
                title="Снятие ордеров с кошелька без права на рынок ANT",
                detail=item.get("details") or "",
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type in ("epoch_ant_wipe", "epoch_ant_credit", "epoch_emission"):
            log.push(
                source="engine",
                status="ok",
                category="epoch",
                canon="§5.5",
                title="Эпоха эмиссии ANT / сброс у Поставщиков",
                detail=(item.get("details") or "")[:240],
                tx_hash=h,
                block_height=block_height,
            )
        elif tx_type == "block_reward_skipped":
            log.push(
                source="engine",
                status="warn",
                category="reward",
                canon="§5.1–5.4",
                title="Базовая награда WRT пропущена — Σb_i вне цели λ·L_total",
                detail=item.get("details") or "",
                tx_hash="",
                block_height=block_height,
            )
        elif tx_type in ("block_reward",):
            log.push(
                source="engine",
                status="ok",
                category="reward",
                canon="§5.1",
                title="Базовая WRT и условные комиссии блока",
                detail=(item.get("details") or "")[:200],
                tx_hash="",
                block_height=block_height,
            )
        elif tx_type == "fee_distribution":
            log.push(
                source="engine",
                status="ok",
                category="fees",
                canon="§5.4",
                title="Делёж комиссий блока ∝ b_i / B",
                detail=(item.get("details") or "")[:200],
                tx_hash="",
                block_height=block_height,
            )


def log_wallet_rejection(sm: "StateManager", op: str, message: str, address: str = "") -> None:
    sm.canon_log.push(
        source="wallet",
        status="reject",
        category=op or "wallet",
        canon="канон / правила симуляции",
        title="Tx отклонена до мемпула",
        detail=f"{message}" + (f" (addr {address[:24]}…)" if address else ""),
        block_height=sm.current_height,
    )


def log_bot_queue(sm: "StateManager", action: str, detail: str, tx_hash: str = "") -> None:
    sm.canon_log.push(
        source="bot",
        status="info",
        category="bot",
        canon="тест нагрузки",
        title=f"Бот поставил в мемпул: {action}",
        detail=detail,
        tx_hash=tx_hash,
        block_height=sm.current_height,
    )


def log_engine_skip(
    sm: "StateManager",
    *,
    block_height: int,
    tx_hash: str,
    tx_type: str,
    canon: str,
    title: str,
    detail: str,
) -> None:
    sm.canon_log.push(
        source="engine",
        status="reject",
        category="delivertx",
        canon=canon,
        title=title,
        detail=detail,
        tx_hash=tx_hash,
        block_height=block_height,
    )
