"""RFC 6962 (Certificate Transparency) Merkle tree, as used by CometBFT."""

from __future__ import annotations

import hashlib

LEAF_PREFIX = b"\x00"
INNER_PREFIX = b"\x01"
EMPTY_HASH = hashlib.sha256(b"").digest()


def _sha256(data: bytes) -> bytes:
    return hashlib.sha256(data).digest()


def leaf_hash(leaf: bytes) -> bytes:
    return _sha256(LEAF_PREFIX + leaf)


def inner_hash(left: bytes, right: bytes) -> bytes:
    return _sha256(INNER_PREFIX + left + right)


def merkle_root_bytes(items: list[bytes]) -> bytes:
    """Compute the RFC 6962 Merkle root of `items` (empty list → SHA256(""))."""
    if not items:
        return EMPTY_HASH
    hashes = [leaf_hash(item) for item in items]
    while len(hashes) > 1:
        nxt: list[bytes] = []
        for i in range(0, len(hashes), 2):
            if i + 1 < len(hashes):
                nxt.append(inner_hash(hashes[i], hashes[i + 1]))
            else:
                nxt.append(hashes[i])
        hashes = nxt
    return hashes[0]


def merkle_root(items: list[bytes]) -> str:
    return merkle_root_bytes(items).hex()


def merkle_root_hex_leaves(hex_items: list[str]) -> str:
    return merkle_root([bytes.fromhex(h) if h else b"" for h in hex_items])
