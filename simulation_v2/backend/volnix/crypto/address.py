"""Bech32-like Volnix addresses: volnix1 + 40 hex chars of SHA-256(pubkey)[:20]."""

from __future__ import annotations

import re

from volnix.crypto.hashing import sha256_bytes

HRP = "volnix"
_ADDR_RE = re.compile(r"^volnix1[0-9a-f]{40}$")


def address_from_pubkey(pubkey_hex: str) -> str:
    digest = sha256_bytes(bytes.fromhex(pubkey_hex))[:20]
    return f"{HRP}1{digest.hex()}"


def is_valid_address(address: str) -> bool:
    return bool(_ADDR_RE.match(address))
