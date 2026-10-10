from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path

from volnix.store.jsonl import JsonlStore


@dataclass
class TxLocation:
    tx_hash: str
    height: int
    index: int
    sender: str


class TxIndex:
    def __init__(self, data_dir: Path) -> None:
        self._log = JsonlStore(data_dir / "tx_index.jsonl")
        self._by_hash: dict[str, TxLocation] = {}
        self._by_account: dict[str, list[TxLocation]] = {}
        self._order: list[str] = []  # append order (oldest → newest)

    def load(self) -> None:
        self._by_hash.clear()
        self._by_account.clear()
        self._order.clear()
        for rec in self._log.iter_records():
            loc = TxLocation(
                tx_hash=rec["tx_hash"],
                height=int(rec["height"]),
                index=int(rec["index"]),
                sender=rec.get("sender", ""),
            )
            self._add(loc)

    def append(self, loc: TxLocation) -> None:
        self._log.append(
            {"tx_hash": loc.tx_hash, "height": loc.height, "index": loc.index, "sender": loc.sender}
        )
        self._add(loc)

    def flush(self) -> None:
        self._log.flush()

    def _add(self, loc: TxLocation) -> None:
        if loc.tx_hash not in self._by_hash:
            self._order.append(loc.tx_hash)
        self._by_hash[loc.tx_hash] = loc
        self._by_account.setdefault(loc.sender, []).append(loc)

    def get(self, tx_hash: str) -> TxLocation | None:
        return self._by_hash.get(tx_hash)

    def by_account(self, address: str) -> list[TxLocation]:
        return list(self._by_account.get(address, []))

    def newest_hashes(self, limit: int, offset: int = 0) -> list[str]:
        """Newest-first slice without sorting the full index."""
        if limit <= 0:
            return []
        n = len(self._order)
        if n == 0:
            return []
        # order is oldest→newest; reverse index walk
        start = max(0, n - 1 - offset)
        out: list[str] = []
        i = start
        while i >= 0 and len(out) < limit:
            out.append(self._order[i])
            i -= 1
        return out

    def __len__(self) -> int:
        return len(self._order)

    def all_hashes_newest_first(self) -> list[str]:
        return list(reversed(self._order))

    def clear(self) -> None:
        self._log.clear()
        self._by_hash.clear()
        self._by_account.clear()
        self._order.clear()
