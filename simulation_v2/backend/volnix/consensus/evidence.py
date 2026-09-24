from __future__ import annotations

from volnix.types.block import Evidence


class EvidencePool:
    def __init__(self) -> None:
        self._pending: list[Evidence] = []

    def add(self, evd: Evidence) -> None:
        self._pending.append(evd)

    def drain(self) -> list[Evidence]:
        items = list(self._pending)
        self._pending.clear()
        return items
