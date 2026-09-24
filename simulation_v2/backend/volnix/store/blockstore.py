from __future__ import annotations

from pathlib import Path

from volnix.store.jsonl import JsonlStore
from volnix.types.block import Block


class BlockStore:
    def __init__(self, data_dir: Path) -> None:
        self._log = JsonlStore(data_dir / "blocks.jsonl")
        self._by_height: dict[int, Block] = {}
        self._by_hash: dict[str, int] = {}

    def load(self) -> None:
        self._by_height.clear()
        self._by_hash.clear()
        for rec in self._log.iter_records():
            block = Block.from_dict(rec)
            self._index(block)

    def append(self, block: Block) -> None:
        self._log.append(block.to_dict())
        self._index(block)

    def _index(self, block: Block) -> None:
        h = block.header.height
        self._by_height[h] = block
        self._by_hash[block.hash()] = h

    def get(self, height: int) -> Block | None:
        return self._by_height.get(height)

    def get_by_hash(self, block_hash: str) -> Block | None:
        height = self._by_hash.get(block_hash)
        if height is None:
            return None
        return self._by_height.get(height)

    def latest_height(self) -> int:
        return max(self._by_height) if self._by_height else 0

    def heights(self) -> list[int]:
        return sorted(self._by_height)

    def range(self, min_h: int, max_h: int) -> list[Block]:
        return [self._by_height[h] for h in self.heights() if min_h <= h <= max_h]

    def clear(self) -> None:
        self._log.clear()
        self._by_height.clear()
        self._by_hash.clear()
