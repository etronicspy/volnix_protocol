"""Кольцевой журнал проверок соответствия симуляции канону (docs/volnix_protocol.md)."""
from __future__ import annotations

import time
from collections import deque
from typing import Any, Dict, List, Optional


class CanonLogBuffer:
    def __init__(self, maxlen: int = 400):
        self._q: deque = deque(maxlen=maxlen)
        self._seq = 0

    def push(
        self,
        *,
        source: str,
        status: str,
        category: str,
        canon: str,
        title: str,
        detail: str = "",
        tx_hash: Optional[str] = None,
        block_height: Optional[int] = None,
        meta: Optional[Dict[str, Any]] = None,
    ) -> None:
        self._seq += 1
        self._q.append(
            {
                "id": self._seq,
                "ts": time.time(),
                "source": source,
                "status": status,
                "category": category,
                "canon": canon,
                "title": title,
                "detail": detail,
                "tx_hash": tx_hash or "",
                "block_height": block_height,
                "meta": meta or {},
            }
        )

    def clear(self) -> None:
        self._q.clear()
        self._seq = 0

    def to_list_newest_first(self) -> List[dict]:
        return list(reversed(self._q))
