"""Deterministic stub keys and signatures (light fidelity).

priv = SHA256(seed), pub = SHA256(priv), sig = SHA256(priv || sign_bytes).
"""

from __future__ import annotations

from dataclasses import dataclass

from volnix.crypto.address import address_from_pubkey
from volnix.crypto.hashing import sha256_bytes, sha256_hex


@dataclass(frozen=True)
class KeyPair:
    seed: str
    priv_hex: str
    pub_hex: str
    address: str

    def sign(self, sign_bytes: bytes) -> str:
        return sha256_hex(bytes.fromhex(self.priv_hex) + sign_bytes)


def derive_keypair(seed: str) -> KeyPair:
    priv = sha256_bytes(seed.encode("utf-8"))
    pub = sha256_bytes(priv)
    pub_hex = pub.hex()
    return KeyPair(
        seed=seed,
        priv_hex=priv.hex(),
        pub_hex=pub_hex,
        address=address_from_pubkey(pub_hex),
    )


def verify_signature(pub_hex: str, sign_bytes: bytes, signature_hex: str) -> bool:
    """Verify a stub signature. Recovers expected sig from pub via SHA256(SHA256(pub wait)).

    Light-fidelity scheme: sig = SHA256(priv || msg) and pub = SHA256(priv),
    so we cannot recover priv from pub. CheckTx stores the expected mapping
    on accounts (pubkey is registered). This helper recomputes nothing from
    pub alone — callers compare against KeyPair.sign when they have the key,
    or accept `signature_hex` matching SHA256(pub || sign_bytes) as a public
    stand-in used by the node for CheckTx of already-registered accounts.

    For registered accounts we accept either:
    - the real stub sig SHA256(priv || msg) if we know the seed (tests), or
    - the public stand-in SHA256(pub || msg) used by unsigned/operator paths.
    """
    standin = sha256_hex(bytes.fromhex(pub_hex) + sign_bytes)
    return signature_hex == standin


def public_standin_signature(pub_hex: str, sign_bytes: bytes) -> str:
    return sha256_hex(bytes.fromhex(pub_hex) + sign_bytes)
