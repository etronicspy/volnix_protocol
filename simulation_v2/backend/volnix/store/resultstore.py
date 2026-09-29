from __future__ import annotations

from pathlib import Path

from volnix.store.jsonl import JsonlStore
from volnix.types.block import BlockResults


class ResultStore:
    def __init__(self, data_dir: Path) -> None:
        self._log = JsonlStore(data_dir / "block_results.jsonl")
        self._by_height: dict[int, BlockResults] = {}

    def load(self) -> None:
        self._by_height.clear()
        for rec in self._log.iter_records():
            res = BlockResults.from_dict(rec)
            self._by_height[res.height] = res

    def append(self, results: BlockResults) -> None:
        self._log.append(results.to_dict())
        self._by_height[results.height] = results

    def flush(self) -> None:
        self._log.flush()

    def get(self, height: int) -> BlockResults | None:
        return self._by_height.get(height)

    def clear(self) -> None:
        self._log.clear()
        self._by_height.clear()
