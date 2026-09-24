"""Canonical application messages (canon 5.1-sim)."""

from __future__ import annotations

from dataclasses import asdict, dataclass, field
from typing import Any


@dataclass
class Msg:
    type: str = ""

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class MsgSend(Msg):
    from_address: str = ""
    to_address: str = ""
    denom: str = ""
    amount: int = 0
    type: str = "bank/MsgSend"


@dataclass
class MsgVerifyIdentity(Msg):
    address: str = ""
    zkp_proof: str = ""
    verification_provider: str = ""
    desired_role: str = ""
    type: str = "ident/MsgVerifyIdentity"


@dataclass
class MsgMigrateRole(Msg):
    from_address: str = ""
    to_address: str = ""
    zkp_proof: str = ""
    type: str = "ident/MsgMigrateRole"


@dataclass
class MsgActivateLZN(Msg):
    validator: str = ""
    amount: int = 0
    type: str = "lizenz/MsgActivateLZN"


@dataclass
class MsgDeactivateLZN(Msg):
    validator: str = ""
    amount: int = 0
    reason: str = ""
    type: str = "lizenz/MsgDeactivateLZN"


@dataclass
class MsgPlaceOrder(Msg):
    owner: str = ""
    market: str = ""
    side: str = ""
    order_type: str = ""
    amount: int = 0
    price: int = 0
    type: str = "anteil/MsgPlaceOrder"


@dataclass
class MsgCancelOrder(Msg):
    owner: str = ""
    order_id: str = ""
    type: str = "anteil/MsgCancelOrder"


@dataclass
class MsgDeclareParticipation(Msg):
    validator: str = ""
    b_i: int = 0
    s_i: int = 0
    type: str = "povb/MsgDeclareParticipation"


@dataclass
class MsgSubmitProposal(Msg):
    proposer: str = ""
    title: str = ""
    description: str = ""
    parameter_changes: dict[str, Any] = field(default_factory=dict)
    deposit: int = 0
    type: str = "gov/MsgSubmitProposal"


@dataclass
class MsgVote(Msg):
    proposal_id: int = 0
    voter: str = ""
    option: str = ""
    type: str = "gov/MsgVote"


MSG_TYPES: dict[str, type[Msg]] = {
    "bank/MsgSend": MsgSend,
    "ident/MsgVerifyIdentity": MsgVerifyIdentity,
    "ident/MsgMigrateRole": MsgMigrateRole,
    "lizenz/MsgActivateLZN": MsgActivateLZN,
    "lizenz/MsgDeactivateLZN": MsgDeactivateLZN,
    "anteil/MsgPlaceOrder": MsgPlaceOrder,
    "anteil/MsgCancelOrder": MsgCancelOrder,
    "povb/MsgDeclareParticipation": MsgDeclareParticipation,
    "gov/MsgSubmitProposal": MsgSubmitProposal,
    "gov/MsgVote": MsgVote,
}


def msg_from_dict(d: dict[str, Any]) -> Msg:
    typ = d.get("type", "")
    cls = MSG_TYPES.get(typ)
    if cls is None:
        raise ValueError(f"unknown message type: {typ}")
    fields = {k: v for k, v in d.items() if k in cls.__dataclass_fields__}
    return cls(**fields)
