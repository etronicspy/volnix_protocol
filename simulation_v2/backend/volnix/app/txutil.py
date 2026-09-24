"""Helpers to build and sign transactions."""

from __future__ import annotations

from volnix.app.state import AppState
from volnix.crypto.keys import KeyPair
from volnix.types.msgs import Msg
from volnix.types.tx import AuthInfo, Fee, SignerInfo, Tx, TxBody


def build_tx(
    state: AppState,
    keypair: KeyPair,
    messages: list[Msg],
    fee: int = 0,
    gas_limit: int = 200_000,
    memo: str = "",
) -> Tx:
    acc = state.accounts.get(keypair.address)
    sequence = acc.sequence if acc else 0
    tx = Tx(
        body=TxBody(messages=list(messages), memo=memo),
        auth_info=AuthInfo(
            signer_infos=[
                SignerInfo(address=keypair.address, pub_hex=keypair.pub_hex, sequence=sequence)
            ],
            fee=Fee(amount=fee, gas_limit=gas_limit),
        ),
        signatures=[],
    )
    tx.signatures = [keypair.sign(tx.sign_bytes())]
    return tx
