from __future__ import annotations

from pathlib import Path

from volnix.store.jsonl import JsonlStore
from volnix.types.validator import ValidatorSet


class ValidatorSetStore:
    def __init__(self, data_dir: Path) -> None:
        self._log = JsonlStore(data_dir / "validator_sets.jsonl")
        self._by_height: dict[int, ValidatorSet] = {}

    def load(self) -> None:
        self._by_height.clear()
        for rec in self._log.iter_records():
            height = int(rec["height"])
            self._by_height[height] = ValidatorSet.from_dict(rec["validator_set"])

    def append(self, height: int, vset: ValidatorSet) -> None:
        self._log.append({"height": height, "validator_set": vset.to_dict()})
        self._by_height[height] = vset

    def get(self, height: int) -> ValidatorSet | None:
        return self._by_height.get(height)

    def latest(self) -> tuple[int, ValidatorSet] | None:
        if not self._by_height:
            return None
        h = max(self._by_height)
        return h, self._by_height[h]

    def clear(self) -> None:
        self._log.clear()
        self._by_height.clear()
