"""Canonical encoding and SHA-256 helpers."""

from __future__ import annotations

import hashlib
import json
from typing import Any


def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def sha256_bytes(data: bytes) -> bytes:
    return hashlib.sha256(data).digest()


def canonical_bytes(obj: Any) -> bytes:
    """Deterministic JSON encoding used for hashes (sorted keys, no whitespace)."""
    return json.dumps(obj, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode("utf-8")


def hash_obj(obj: Any) -> str:
    return sha256_hex(canonical_bytes(obj))
