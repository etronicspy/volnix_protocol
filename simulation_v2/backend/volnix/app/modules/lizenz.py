from __future__ import annotations

from volnix.app.state import AppState
from volnix.types.events import Event, ev
from volnix.types.msgs import MsgActivateLZN, MsgDeactivateLZN
from volnix.types.role import Role

# Sentinel: sole / first activator is not share-capped (Cosmos x/lizenz).
_UNLIMITED = 1 << 62


class LizenzError(ValueError):
    pass


def max_activated_per_validator(new_total: int) -> int:
    """⌊new_total / 3⌋ — absolute activated ceiling after the transition."""
    if new_total <= 0:
        return 0
    return new_total // 3


def _others_activated(state: AppState, address: str) -> int:
    """L_total excluding this address's activated balance."""
    acc = state.accounts.get(address)
    current = int(acc.lzn_activated) if acc is not None else 0
    return max(0, state.l_total() - current)


def share_cap_exempt(state: AppState, address: str) -> bool:
    """First network activation or sole activated holder — no ⌊L_total/3⌋ cap."""
    acc = state.accounts.get(address)
    current = int(acc.lzn_activated) if acc is not None else 0
    l_total = state.l_total()
    if l_total == 0 and current == 0:
        return True
    if current > 0 and l_total == current:
        return True
    return False


def max_allowed_activated(state: AppState, address: str) -> int:
    """
    Absolute activated LZN this address may hold after a transition.

    Rule (Cosmos x/lizenz / L_total): new_amount ≤ ⌊new_L_total / 3⌋ where
    new_L_total = (L_total − current) + new_amount. Sole/first activator: uncapped.
    """
    if share_cap_exempt(state, address):
        return _UNLIMITED
    t0 = _others_activated(state, address)
    # Max X ≥ 0 with X ≤ ⌊(t0 + X) / 3⌋  ⇒  X ≤ ⌊t0 / 2⌋.
    best = 0
    for x in range(0, t0 // 2 + 2):
        if x <= (t0 + x) // 3:
            best = x
    return best


def activation_headroom(state: AppState, address: str) -> int:
    """Additional micro-LZN this address may activate under the L_total share cap."""
    acc = state.accounts.get(address)
    current = int(acc.lzn_activated) if acc is not None else 0
    return max(0, max_allowed_activated(state, address) - current)


def assert_activation_allowed(state: AppState, address: str, new_activated: int) -> None:
    if share_cap_exempt(state, address):
        return
    current = int(state.accounts[address].lzn_activated)
    new_total = state.l_total() - current + new_activated
    max_allowed = max_activated_per_validator(new_total)
    if new_activated > max_allowed:
        raise LizenzError(
            "activation exceeds one-third of L_total "
            f"(new={new_activated} max={max_allowed} new_L_total={new_total})"
        )


def deliver_activate(state: AppState, msg: MsgActivateLZN) -> list[Event]:
    acc = state.accounts.get(msg.validator)
    if acc is None or acc.role != Role.VALIDATOR:
        raise LizenzError("only a validator can activate LZN")
    if msg.amount <= 0:
        raise LizenzError("amount must be positive")
    if acc.lzn < msg.amount:
        raise LizenzError("insufficient free LZN")
    new_activated = acc.lzn_activated + msg.amount
    assert_activation_allowed(state, msg.validator, new_activated)
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
