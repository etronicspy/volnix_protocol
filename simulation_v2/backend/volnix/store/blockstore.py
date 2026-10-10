from __future__ import annotations

import json
from pathlib import Path

from volnix.store.jsonl import JsonlStore
from volnix.store.lru import LRUCache
from volnix.types.block import Block


class BlockStore:
    """Disk-backed blocks.jsonl; tip + small LRU hot. Hash→height in sidecar."""

    def __init__(self, data_dir: Path, *, lru_size: int = 8) -> None:
        self._log = JsonlStore(data_dir / "blocks.jsonl")
        self._hash_path = data_dir / "blocks.hash.idx"
        self._by_hash: dict[str, int] = {}
        self._latest_height = -1
        self._tip: Block | None = None
        self._lru: LRUCache[int, Block] = LRUCache(lru_size)

    def load(self) -> None:
        """Open indexes; do not materialize all Block objects."""
        self._lru.clear()
        self._tip = None
        self._by_hash.clear()
        self._latest_height = -1
        self._load_hash_index()
        n = self._log.record_count()
        if n == 0:
            return
        tip = self._read_index(n - 1)
        if tip is None:
            return
        self._tip = tip
        self._latest_height = tip.header.height
        self._lru.put(tip.header.height, tip)
        # Contiguous heights 0..n-1 expected; rebuild hash map if sidecar short.
        if len(self._by_hash) < n:
            self._rebuild_hash_index()

    def _load_hash_index(self) -> None:
        if not self._hash_path.exists():
            return
        with self._hash_path.open("r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line:
                    continue
                rec = json.loads(line)
                self._by_hash[str(rec["hash"])] = int(rec["height"])

    def _rebuild_hash_index(self) -> None:
        self._by_hash.clear()
        tmp = self._hash_path.with_suffix(".idx.tmp")
        with tmp.open("w", encoding="utf-8") as out:
            for i in range(self._log.record_count()):
                block = self._read_index(i)
                if block is None:
                    continue
                h = block.hash()
                height = block.header.height
                self._by_hash[h] = height
                out.write(json.dumps({"hash": h, "height": height}, separators=(",", ":")) + "\n")
        tmp.replace(self._hash_path)

    def _append_hash(self, block: Block) -> None:
        self._by_hash[block.hash()] = block.header.height
        with self._hash_path.open("a", encoding="utf-8") as fh:
            fh.write(
                json.dumps({"hash": block.hash(), "height": block.header.height}, separators=(",", ":"))
                + "\n"
            )

    def _read_index(self, index: int) -> Block | None:
        rec = self._log.read_record(index)
        if rec is None:
            return None
        return Block.from_dict(rec)

    def append(self, block: Block) -> None:
        self._log.append(block.to_dict())
        self._append_hash(block)
        h = block.header.height
        self._latest_height = h
        self._tip = block
        self._lru.put(h, block)

    def flush(self) -> None:
        self._log.flush()

    def get(self, height: int) -> Block | None:
        if height < 0 or (self._latest_height >= 0 and height > self._latest_height):
            return None
        cached = self._lru.get(height)
        if cached is not None:
            return cached
        if self._tip is not None and self._tip.header.height == height:
            return self._tip
        # Contiguous store: record index == height when genesis at 0.
        if height >= self._log.record_count():
            return None
        block = self._read_index(height)
        if block is None:
            return None
        if block.header.height != height:
            # Non-contiguous fallback: linear scan (should not happen on live chain).
            for i in range(self._log.record_count()):
                b = self._read_index(i)
                if b is not None and b.header.height == height:
                    self._lru.put(height, b)
                    return b
            return None
        self._lru.put(height, block)
        return block

    def get_by_hash(self, block_hash: str) -> Block | None:
        height = self._by_hash.get(block_hash)
        if height is None:
            return None
        return self.get(height)

    def latest_height(self) -> int:
        return max(0, self._latest_height) if self._latest_height >= 0 else 0

    def heights(self) -> list[int]:
        if self._latest_height < 0:
            return []
        return list(range(0, self._latest_height + 1))

    def range(self, min_h: int, max_h: int) -> list[Block]:
        out: list[Block] = []
        if self._latest_height < 0:
            return out
        lo = max(0, min_h)
        hi = min(self._latest_height, max_h)
        for h in range(lo, hi + 1):
            b = self.get(h)
            if b is not None:
                out.append(b)
        return out

    def clear(self) -> None:
        self._log.clear()
        if self._hash_path.exists():
            self._hash_path.unlink()
        self._by_hash.clear()
        self._lru.clear()
        self._tip = None
        self._latest_height = -1

    def drop_cache(self) -> None:
        """Test helper: forget hot objects (disk + indexes remain)."""
        self._lru.clear()
        self._tip = None
