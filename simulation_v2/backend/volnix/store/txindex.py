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

    def load(self) -> None:
        self._by_hash.clear()
        self._by_account.clear()
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
        self._by_hash[loc.tx_hash] = loc
        self._by_account.setdefault(loc.sender, []).append(loc)

    def get(self, tx_hash: str) -> TxLocation | None:
        return self._by_hash.get(tx_hash)

    def by_account(self, address: str) -> list[TxLocation]:
        return list(self._by_account.get(address, []))

    def all_hashes_newest_first(self) -> list[str]:
        locs = sorted(self._by_hash.values(), key=lambda x: (x.height, x.index), reverse=True)
        return [l.tx_hash for l in locs]

    def clear(self) -> None:
        self._log.clear()
        self._by_hash.clear()
        self._by_account.clear()
