from __future__ import annotations

from volnix.app.state import AppState
from volnix.types.events import Event, ev
from volnix.types.msgs import MsgActivateLZN, MsgDeactivateLZN
from volnix.types.role import Role


class LizenzError(ValueError):
    pass


def deliver_activate(state: AppState, msg: MsgActivateLZN) -> list[Event]:
    acc = state.accounts.get(msg.validator)
    if acc is None or acc.role != Role.VALIDATOR:
        raise LizenzError("only a validator can activate LZN")
    if msg.amount <= 0:
        raise LizenzError("amount must be positive")
    if acc.lzn < msg.amount:
        raise LizenzError("insufficient free LZN")
    if acc.lzn_activated + msg.amount > state.params.lzn_max_frozen_per_address:
        raise LizenzError("activation exceeds per-address ceiling")
    acc.lzn -= msg.amount
    acc.lzn_activated += msg.amount
    acc.lzn_freeze_until = state.height + state.params.lzn_freeze_period
    return [
        ev(
            "lizenz.lizenz_activated",
            validator=msg.validator,
            amount=msg.amount,
            freeze_until=acc.lzn_freeze_until,
            block_height=state.height,
        )
    ]


def deliver_deactivate(state: AppState, msg: MsgDeactivateLZN) -> list[Event]:
    acc = state.accounts.get(msg.validator)
    if acc is None or acc.role != Role.VALIDATOR:
        raise LizenzError("only a validator can deactivate LZN")
    if msg.amount <= 0:
        raise LizenzError("amount must be positive")
    if acc.lzn_activated < msg.amount:
        raise LizenzError("insufficient activated LZN")
    if state.height < acc.lzn_freeze_until:
        raise LizenzError("LZN still frozen")
    acc.lzn_activated -= msg.amount
    acc.lzn += msg.amount
    return [
        ev(
            "lizenz.lizenz_deactivated",
            validator=msg.validator,
            amount=msg.amount,
            reason=msg.reason,
            block_height=state.height,
        )
    ]
