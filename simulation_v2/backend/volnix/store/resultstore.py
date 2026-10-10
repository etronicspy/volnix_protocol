from __future__ import annotations

from pathlib import Path

from volnix.store.jsonl import JsonlStore
from volnix.store.lru import LRUCache
from volnix.types.block import BlockResults


class ResultStore:
    def __init__(self, data_dir: Path, *, lru_size: int = 4) -> None:
        self._log = JsonlStore(data_dir / "block_results.jsonl")
        self._latest_height = -1
        self._tip: BlockResults | None = None
        self._lru: LRUCache[int, BlockResults] = LRUCache(lru_size)

    def load(self) -> None:
        self._lru.clear()
        self._tip = None
        self._latest_height = -1
        n = self._log.record_count()
        if n == 0:
            return
        tip = self._read_index(n - 1)
        if tip is None:
            return
        self._tip = tip
        self._latest_height = tip.height
        self._lru.put(tip.height, tip)

    def _read_index(self, index: int) -> BlockResults | None:
        rec = self._log.read_record(index)
        if rec is None:
            return None
        return BlockResults.from_dict(rec)

    def append(self, results: BlockResults) -> None:
        self._log.append(results.to_dict())
        self._latest_height = results.height
        self._tip = results
        self._lru.put(results.height, results)

    def flush(self) -> None:
        self._log.flush()

    def get(self, height: int) -> BlockResults | None:
        if height < 0 or (self._latest_height >= 0 and height > self._latest_height):
            return None
        cached = self._lru.get(height)
        if cached is not None:
            return cached
        if self._tip is not None and self._tip.height == height:
            return self._tip
        if height >= self._log.record_count():
            return None
        res = self._read_index(height)
        if res is None:
            return None
        if res.height != height:
            for i in range(self._log.record_count()):
                r = self._read_index(i)
                if r is not None and r.height == height:
                    self._lru.put(height, r)
                    return r
            return None
        self._lru.put(height, res)
        return res

    def clear(self) -> None:
        self._log.clear()
        self._lru.clear()
        self._tip = None
        self._latest_height = -1

    def drop_cache(self) -> None:
        self._lru.clear()
        self._tip = None
