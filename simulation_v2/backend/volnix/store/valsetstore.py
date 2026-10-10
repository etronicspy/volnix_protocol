from __future__ import annotations

from pathlib import Path

from volnix.store.jsonl import JsonlStore
from volnix.store.lru import LRUCache
from volnix.types.validator import ValidatorSet


class ValidatorSetStore:
    def __init__(self, data_dir: Path, *, lru_size: int = 4) -> None:
        self._log = JsonlStore(data_dir / "validator_sets.jsonl")
        self._latest_height = -1
        self._tip: ValidatorSet | None = None
        self._lru: LRUCache[int, ValidatorSet] = LRUCache(lru_size)

    def load(self) -> None:
        self._lru.clear()
        self._tip = None
        self._latest_height = -1
        n = self._log.record_count()
        if n == 0:
            return
        tip_h, tip_vs = self._read_index(n - 1)
        if tip_vs is None:
            return
        self._tip = tip_vs
        self._latest_height = tip_h
        self._lru.put(tip_h, tip_vs)

    def _read_index(self, index: int) -> tuple[int, ValidatorSet | None]:
        rec = self._log.read_record(index)
        if rec is None:
            return -1, None
        height = int(rec["height"])
        return height, ValidatorSet.from_dict(rec["validator_set"])

    def append(self, height: int, vset: ValidatorSet) -> None:
        self._log.append({"height": height, "validator_set": vset.to_dict()})
        self._latest_height = height
        self._tip = vset
        self._lru.put(height, vset)

    def flush(self) -> None:
        self._log.flush()

    def get(self, height: int) -> ValidatorSet | None:
        if height < 0 or (self._latest_height >= 0 and height > self._latest_height):
            return None
        cached = self._lru.get(height)
        if cached is not None:
            return cached
        if self._tip is not None and self._latest_height == height:
            return self._tip
        if height >= self._log.record_count():
            return None
        h, vs = self._read_index(height)
        if vs is None:
            return None
        if h != height:
            for i in range(self._log.record_count()):
                rh, rvs = self._read_index(i)
                if rvs is not None and rh == height:
                    self._lru.put(height, rvs)
                    return rvs
            return None
        self._lru.put(height, vs)
        return vs

    def latest(self) -> tuple[int, ValidatorSet] | None:
        if self._latest_height < 0 or self._tip is None:
            n = self._log.record_count()
            if n == 0:
                return None
            h, vs = self._read_index(n - 1)
            if vs is None:
                return None
            self._latest_height = h
            self._tip = vs
            return h, vs
        return self._latest_height, self._tip

    def clear(self) -> None:
        self._log.clear()
        self._lru.clear()
        self._tip = None
        self._latest_height = -1

    def drop_cache(self) -> None:
        self._lru.clear()
        self._tip = None
