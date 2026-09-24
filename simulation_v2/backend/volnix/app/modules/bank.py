from __future__ import annotations

from volnix.app.state import AppState
from volnix.types.events import Event, ev
from volnix.types.msgs import MsgSend
from volnix.types.role import Role


class BankError(ValueError):
    pass


def deliver_send(state: AppState, msg: MsgSend) -> list[Event]:
    if msg.denom != "uwrt" and msg.denom != "wrt":
        raise BankError("MsgSend is only allowed for WRT")
    if msg.amount <= 0:
        raise BankError("amount must be positive")
    if msg.from_address == msg.to_address:
        raise BankError("cannot send to self")
    src = state.accounts.get(msg.from_address)
    if src is None:
        raise BankError("sender account not found")
    if src.wrt < msg.amount:
        raise BankError("insufficient WRT")
    src.wrt -= msg.amount
    dst = state.ensure_account(msg.to_address)
    dst.wrt += msg.amount
    return [
        ev(
            "bank.transfer",
            from_address=msg.from_address,
            to_address=msg.to_address,
            denom="uwrt",
            amount=msg.amount,
        )
    ]


def credit(state: AppState, address: str, denom: str, amount: int) -> None:
    if amount <= 0:
        return
    acc = state.ensure_account(address)
    if denom == "uwrt":
        acc.wrt += amount
        state.wrt_supply += amount
    elif denom == "ulzn_free":
        acc.lzn += amount
    elif denom == "ulzn_activated":
        acc.lzn_activated += amount
    elif denom == "uant":
        acc.ant += amount
    else:
        raise BankError(f"unknown denom {denom}")


def burn(state: AppState, address: str, denom: str, amount: int) -> None:
    if amount <= 0:
        return
    acc = state.get_account(address)
    if denom == "uant":
        if acc.ant < amount:
            raise BankError("insufficient ANT to burn")
        acc.ant -= amount
    elif denom == "uwrt":
        if acc.wrt < amount:
            raise BankError("insufficient WRT")
        acc.wrt -= amount
        state.wrt_supply -= amount
    else:
        raise BankError(f"cannot burn {denom}")


def can_hold_ant(role: Role) -> bool:
    return role in (Role.SUPPLIER, Role.VALIDATOR)
