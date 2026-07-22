"""Кольцевой журнал проверок соответствия симуляции канону (docs/volnix_protocol.md).

В RAM держим последние `maxlen` записей (быстрый WS-init и UI), на диск пишем
полностью в `canon_log.jsonl` (включается через `attach_ledger`).
"""
from __future__ import annotations

import time
from collections import deque
from typing import Any, Dict, List, Optional


class CanonLogBuffer:
    def __init__(self, maxlen: int = 400):
        self._q: deque = deque(maxlen=maxlen)
        self._seq = 0
        self._ledger = None  # type: Optional[Any]

    def attach_ledger(self, ledger) -> None:  # type: ignore[no-untyped-def]
        """Опционально: дописывать каждую запись на диск (CanonLogLedger)."""
        self._ledger = ledger
        # Если в ledger уже есть записи — продолжим нумерацию с max(id)
        try:
            mx = ledger.max_id()
            if mx > self._seq:
                self._seq = mx
        except Exception:
            pass

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
        entry = {
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
        self._q.append(entry)
        if self._ledger is not None:
            try:
                self._ledger.append(entry)
            except OSError:
                pass

    def clear(self) -> None:
        self._q.clear()
        self._seq = 0
        if self._ledger is not None:
            try:
                self._ledger.truncate()
            except OSError:
                pass

    def to_list_newest_first(self) -> List[dict]:
        return list(reversed(self._q))

    def max_id(self) -> int:
        return self._seq
