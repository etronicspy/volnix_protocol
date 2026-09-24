"""Stand helper: auto-enqueue MsgDeclareParticipation for known validator seeds.

Canon requires a declare tx each height for ANT burns (§5.4). Empty auto-produced
blocks therefore burn nothing unless the stand injects declares.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Optional

from volnix.app.genesis import load_genesis_doc
from volnix.app.modules.povb import suggest_declare_bs
from volnix.app.txutil import build_tx
from volnix.crypto.keys import derive_keypair
from volnix.types.msgs import MsgDeclareParticipation
from volnix.types.role import Role
from volnix.types.tx import Tx

if TYPE_CHECKING:
    from volnix.node.node import Node


def build_genesis_declare_tx(node: "Node") -> Optional[Tx]:
    """Build a §6.3(5)-style declare for the genesis validator, or None if skip."""
    doc = load_genesis_doc(node.genesis_path)
    seed = doc.get("app_state", {}).get("genesis_validator", {}).get(
        "seed", "volnix-genesis-validator-v2"
    )
    kp = derive_keypair(seed)
    acc = node.app.state.accounts.get(kp.address)
    if acc is None or acc.role != Role.VALIDATOR:
        return None
    if acc.lzn_activated <= 0:
        return None
    for existing in node.mempool.pending():
        if existing.signer() == kp.address and any(
            m.type == "povb/MsgDeclareParticipation" for m in existing.body.messages
        ):
            return None

    pair = suggest_declare_bs(acc.lzn_activated, acc.ant, node.app.state.params.alpha)
    if pair is None:
        return None
    b_i, s_i = pair
    return build_tx(
        node.app.state,
        kp,
        [MsgDeclareParticipation(validator=kp.address, b_i=b_i, s_i=s_i)],
    )
