"""High-level bot actions via NodeClient (stub ZKP, transfers, market, activate)."""

from __future__ import annotations

import logging
import uuid
from typing import Any, Optional

from volnix_traffic.client import NodeClient
from volnix_traffic.registry import ROLE_SUPPLIER, ROLE_VALIDATOR, BotWallet

log = logging.getLogger("volnix_traffic.actions")


def stub_zkp() -> str:
    return f"stub-zkp-{uuid.uuid4().hex}"


def ok_broadcast(resp: dict[str, Any]) -> bool:
    code = resp.get("code")
    if code is None and "broadcast" in resp:
        code = (resp.get("broadcast") or {}).get("code")
    return code == 0 or code == "0" or code is None and "hash" in resp


async def create_wallet(client: NodeClient, bot: BotWallet) -> BotWallet:
    out = await client.create_account(bot.seed)
    bot.address = str(out.get("address") or "")
    bot.pub_hex = str(out.get("pub_hex") or "")
    return bot


async def fund_wrt(client: NodeClient, bot: BotWallet, amount: int) -> bool:
    if amount <= 0 or not bot.address:
        return False
    try:
        out = await client.mint_wrt(bot.address, amount)
    except Exception as exc:
        log.debug("fund %s failed: %s", bot.address, exc)
        return False
    ok = ok_broadcast(out)
    if ok:
        bot.funded = True
    return ok


async def verify_identity(client: NodeClient, bot: BotWallet, desired_role: str) -> bool:
    if desired_role not in (ROLE_SUPPLIER, ROLE_VALIDATOR):
        return False
    proof = bot.zkp_id or stub_zkp()
    try:
        out = await client.verify(bot.seed, desired_role, proof)
    except Exception as exc:
        log.debug("verify %s failed: %s", bot.seed, exc)
        return False
    if ok_broadcast(out):
        bot.zkp_id = proof
        bot.verified = True
        bot.desired_role = desired_role
        bot.role = desired_role
        return True
    return False


async def send_wrt(client: NodeClient, bot: BotWallet, to_address: str, amount: int) -> bool:
    if amount <= 0 or not bot.address or not to_address:
        return False
    try:
        out = await client.signed_tx(
            bot.seed,
            [
                {
                    "type": "bank/MsgSend",
                    "from_address": bot.address,
                    "to_address": to_address,
                    "denom": "uwrt",
                    "amount": int(amount),
                }
            ],
        )
    except Exception as exc:
        log.debug("send %s failed: %s", bot.address, exc)
        return False
    return ok_broadcast(out)


async def activate_lzn(client: NodeClient, bot: BotWallet, amount: int) -> bool:
    if amount <= 0 or not bot.address:
        return False
    try:
        out = await client.signed_tx(
            bot.seed,
            [
                {
                    "type": "lizenz/MsgActivateLZN",
                    "validator": bot.address,
                    "amount": int(amount),
                }
            ],
        )
    except Exception as exc:
        log.debug("activate %s failed: %s", bot.address, exc)
        return False
    return ok_broadcast(out)


async def place_order(
    client: NodeClient,
    bot: BotWallet,
    *,
    market: str,
    side: str,
    amount: int,
    price: int,
    order_type: str = "LIMIT",
) -> bool:
    if amount <= 0:
        return False
    try:
        out = await client.place_order(
            bot.seed,
            market=market,
            side=side,
            amount=amount,
            price=price,
            order_type=order_type,
        )
    except Exception as exc:
        log.debug("order %s failed: %s", bot.address, exc)
        return False
    return ok_broadcast(out)


async def cancel_order(client: NodeClient, bot: BotWallet, order_id: str) -> bool:
    if not order_id:
        return False
    try:
        out = await client.signed_tx(
            bot.seed,
            [
                {
                    "type": "anteil/MsgCancelOrder",
                    "owner": bot.address,
                    "order_id": order_id,
                }
            ],
        )
    except Exception as exc:
        log.debug("cancel %s failed: %s", bot.address, exc)
        return False
    return ok_broadcast(out)


async def declare(client: NodeClient, bot: BotWallet, b_i: int, s_i: int) -> bool:
    try:
        out = await client.declare(bot.seed, b_i, s_i)
    except Exception as exc:
        log.debug("declare %s failed: %s", bot.address, exc)
        return False
    return ok_broadcast(out)
