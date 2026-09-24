from __future__ import annotations

from volnix.app.modules.supplier_gate import check_new_supplier
from volnix.app.state import Account, AppState
from volnix.types.events import Event, ev
from volnix.types.msgs import MsgMigrateRole, MsgVerifyIdentity
from volnix.types.role import Role


class IdentError(ValueError):
    pass


def deliver_verify(state: AppState, msg: MsgVerifyIdentity) -> list[Event]:
    if msg.desired_role not in (Role.SUPPLIER, Role.VALIDATOR, "supplier", "validator"):
        raise IdentError("desired_role must be supplier or validator")
    desired = Role(msg.desired_role)
    if not msg.zkp_proof:
        raise IdentError("zkp_proof required")
    if msg.zkp_proof in state.nullifiers:
        raise IdentError("zkp proof already used")
    acc = state.ensure_account(msg.address)
    if acc.role != Role.CITIZEN:
        raise IdentError("account already has a verified role")
    if desired == Role.SUPPLIER:
        n = len(state.active_suppliers())
        if n >= state.params.max_active_suppliers:
            raise IdentError("supplier slots full")
        try:
            check_new_supplier(state)
        except ValueError as exc:
            raise IdentError(str(exc)) from exc
    acc.role = desired
    acc.zkp_id = msg.zkp_proof
    acc.last_tx_height = state.height
    state.nullifiers.add(msg.zkp_proof)
    return [
        ev(
            "ident.identity_verified",
            account=msg.address,
            new_role=str(desired),
            identity_hash=msg.zkp_proof,
            block_height=state.height,
        )
    ]


def deliver_migrate(state: AppState, msg: MsgMigrateRole) -> list[Event]:
    from volnix.app.modules.anteil import cancel_order

    src = state.accounts.get(msg.from_address)
    if src is None or src.role == Role.CITIZEN:
        raise IdentError("source has no verified role")
    if src.genesis_no_zkp and not src.zkp_id:
        raise IdentError("genesis validator without ZKP cannot migrate via ZKP")
    if not msg.zkp_proof:
        raise IdentError("zkp_proof required")
    dst = state.ensure_account(msg.to_address)
    if dst.role != Role.CITIZEN:
        raise IdentError("destination must be a new citizen wallet")

    # Cancel open orders on source — return escrow before transfer (§3.2)
    for o in list(state.orders.values()):
        if o.owner == src.address and o.status == "open":
            cancel_order(state, o.order_id, protocol=True)

    activated = src.lzn_activated
    free = src.lzn
    if activated > state.params.lzn_max_frozen_per_address:
        # Overflow beyond per-address ceiling becomes free LZN on destination
        overflow = activated - state.params.lzn_max_frozen_per_address
        activated = state.params.lzn_max_frozen_per_address
        free += overflow

    old_role = src.role
    dst.role = old_role
    dst.zkp_id = src.zkp_id
    dst.ant = src.ant
    dst.lzn = free
    dst.lzn_activated = activated
    dst.lzn_freeze_until = src.lzn_freeze_until
    dst.last_tx_height = state.height
    dst.pub_hex = dst.pub_hex or ""

    src.ant = 0
    src.lzn = 0
    src.lzn_activated = 0
    src.role = Role.CITIZEN
    src.zkp_id = ""
    src.lzn_freeze_until = 0
    # WRT stays on the lost wallet
    return [
        ev(
            "ident.role_migrated",
            from_address=msg.from_address,
            to_address=msg.to_address,
            old_role=str(old_role),
            new_role=str(dst.role),
            block_height=state.height,
        )
    ]


def check_moa(state: AppState) -> list[Event]:
    events: list[Event] = []
    for acc in list(state.accounts.values()):
        if acc.role == Role.CITIZEN:
            continue
        window = (
            state.params.moa_supplier_window
            if acc.role == Role.SUPPLIER
            else state.params.moa_validator_window
        )
        last = acc.last_tx_height if acc.last_tx_height > 0 else acc.created_height
        if state.height - last < window:
            continue
        events.extend(_strip_role_moa(state, acc))
    return events


def _strip_role_moa(state: AppState, acc: Account) -> list[Event]:
    """Strip role: burn ANT (free+escrow), keep LZN, unfreeze activated→free for sellability."""
    from volnix.app.modules.anteil import cancel_order

    events: list[Event] = []
    old = acc.role

    # Cancel all open orders (return escrow), then burn all ANT
    for o in list(state.orders.values()):
        if o.owner == acc.address and o.status == "open":
            cancel_order(state, o.order_id, protocol=True)

    burned = acc.ant
    acc.ant = 0

    # Unfreeze activated LZN so citizen can SELL LZN on the book (§4.1 / §5.3)
    if acc.lzn_activated > 0:
        acc.lzn += acc.lzn_activated
        acc.lzn_activated = 0
        acc.lzn_freeze_until = 0

    if acc.zkp_id:
        state.nullifiers.discard(acc.zkp_id)
        acc.zkp_id = ""
    acc.role = Role.CITIZEN

    events.append(
        ev(
            "ident.role_changed",
            account=acc.address,
            old_role=str(old),
            new_role="citizen",
            role_change_reason="moa",
            burn_amount=burned,
            block_height=state.height,
        )
    )
    return events
