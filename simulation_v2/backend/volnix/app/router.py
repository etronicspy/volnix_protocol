from __future__ import annotations

from volnix.app.modules import anteil, bank, gov, ident, lizenz, povb
from volnix.app.state import AppState
from volnix.types.events import Event
from volnix.types.msgs import (
    MsgActivateLZN,
    MsgCancelOrder,
    MsgDeactivateLZN,
    MsgDeclareParticipation,
    MsgMigrateRole,
    MsgPlaceOrder,
    MsgSend,
    MsgSubmitProposal,
    MsgVerifyIdentity,
    MsgVote,
)
from volnix.types.tx import Tx


GAS_PER_MSG = {
    "bank/MsgSend": 50_000,
    "ident/MsgVerifyIdentity": 80_000,
    "ident/MsgMigrateRole": 80_000,
    "lizenz/MsgActivateLZN": 60_000,
    "lizenz/MsgDeactivateLZN": 60_000,
    "anteil/MsgPlaceOrder": 70_000,
    "anteil/MsgCancelOrder": 40_000,
    "povb/MsgDeclareParticipation": 40_000,
    "gov/MsgSubmitProposal": 80_000,
    "gov/MsgVote": 40_000,
}


def estimate_gas(tx: Tx) -> int:
    return sum(GAS_PER_MSG.get(m.type, 50_000) for m in tx.body.messages)


def dispatch(state: AppState, tx: Tx) -> list[Event]:
    events: list[Event] = []
    for msg in tx.body.messages:
        if isinstance(msg, MsgSend):
            events.extend(bank.deliver_send(state, msg))
        elif isinstance(msg, MsgVerifyIdentity):
            events.extend(ident.deliver_verify(state, msg))
        elif isinstance(msg, MsgMigrateRole):
            events.extend(ident.deliver_migrate(state, msg))
        elif isinstance(msg, MsgActivateLZN):
            events.extend(lizenz.deliver_activate(state, msg))
        elif isinstance(msg, MsgDeactivateLZN):
            events.extend(lizenz.deliver_deactivate(state, msg))
        elif isinstance(msg, MsgPlaceOrder):
            events.extend(anteil.deliver_place(state, msg))
        elif isinstance(msg, MsgCancelOrder):
            events.extend(anteil.deliver_cancel(state, msg))
        elif isinstance(msg, MsgDeclareParticipation):
            events.extend(povb.deliver_declare(state, msg))
        elif isinstance(msg, MsgSubmitProposal):
            events.extend(gov.deliver_submit(state, msg))
        elif isinstance(msg, MsgVote):
            events.extend(gov.deliver_vote(state, msg))
        else:
            raise ValueError(f"unhandled message type {getattr(msg, 'type', type(msg))}")
    return events
