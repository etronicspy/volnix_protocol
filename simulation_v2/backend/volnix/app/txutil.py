"""Helpers to build and sign transactions."""

from __future__ import annotations

from volnix.app.state import AppState
from volnix.crypto.keys import KeyPair
from volnix.types.msgs import Msg
from volnix.types.tx import AuthInfo, Fee, SignerInfo, Tx, TxBody


def next_account_sequence(state: AppState, address: str, pending: int = 0) -> int:
    """Next signer sequence: committed account.sequence + txs already in mempool."""
    acc = state.accounts.get(address)
    base = acc.sequence if acc else 0
    return base + max(0, int(pending))


def build_tx(
    state: AppState,
    keypair: KeyPair,
    messages: list[Msg],
    fee: int = 0,
    gas_limit: int = 200_000,
    memo: str = "",
    sequence: int | None = None,
) -> Tx:
    acc = state.accounts.get(keypair.address)
    if sequence is None:
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
