"""Append-only JSONL ledger для блоков и canon-журнала.

Заменяет полный snapshot `state.json` на каждый блок:
- `blocks.jsonl` — по одной строке на блок (read-on-demand для replay/export);
- `canon_log.jsonl` — по одной строке на запись canon-аудита;
- snapshot `state.json` пишется раз в N блоков и при reset.

В RAM держим последние settings.blocks_in_memory блоков для UI/explorer.
"""
from __future__ import annotations

import json
import os
import threading
from typing import Any, Iterable, Iterator, List, Optional


class JsonlAppender:
    """Тонкая обёртка над append-only JSONL с потокобезопасной записью."""

    def __init__(self, path: str) -> None:
        self.path = path
        self._lock = threading.Lock()
        d = os.path.dirname(path)
        if d:
            os.makedirs(d, exist_ok=True)

    def append(self, record: dict) -> None:
        line = json.dumps(record, ensure_ascii=False, separators=(",", ":"))
        with self._lock, open(self.path, "a", encoding="utf-8") as f:
            f.write(line)
            f.write("\n")

    def append_many(self, records: Iterable[dict]) -> int:
        n = 0
        with self._lock, open(self.path, "a", encoding="utf-8") as f:
            for rec in records:
                f.write(json.dumps(rec, ensure_ascii=False, separators=(",", ":")))
                f.write("\n")
                n += 1
        return n

    def iter_records(self) -> Iterator[dict]:
        if not os.path.exists(self.path):
            return
        with open(self.path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    yield json.loads(line)
                except json.JSONDecodeError:
                    continue

    def count(self) -> int:
        if not os.path.exists(self.path):
            return 0
        n = 0
        with open(self.path, encoding="utf-8") as f:
            for line in f:
                if line.strip():
                    n += 1
        return n

    def truncate(self) -> None:
        with self._lock:
            if os.path.exists(self.path):
                os.remove(self.path)

    def rewrite_from(self, records: Iterable[dict]) -> int:
        tmp = self.path + ".tmp"
        with self._lock:
            with open(tmp, "w", encoding="utf-8") as f:
                n = 0
                for rec in records:
                    f.write(json.dumps(rec, ensure_ascii=False, separators=(",", ":")))
                    f.write("\n")
                    n += 1
            os.replace(tmp, self.path)
        return n


class BlockLedger:
    """JSONL по 1 блоку на строку (height + tx_count для быстрых выборок)."""

    def __init__(self, path: str) -> None:
        self.path = path
        self._writer = JsonlAppender(path)

    def append_block(self, block: dict) -> None:
        self._writer.append(block)

    def all_heights(self) -> List[int]:
        return [int(b.get("height", -1)) for b in self._writer.iter_records()]

    def read_range(self, from_height: int, to_height: int) -> List[dict]:
        """Закрытый интервал [from_height, to_height]; включающий обе границы."""
        out: List[dict] = []
        for blk in self._writer.iter_records():
            h = int(blk.get("height", -1))
            if from_height <= h <= to_height:
                out.append(blk)
        return out

    def read_tail(self, n: int) -> List[dict]:
        if n <= 0:
            return []
        buf: List[dict] = []
        for blk in self._writer.iter_records():
            buf.append(blk)
            if len(buf) > n:
                buf.pop(0)
        return buf

    def get_by_height(self, height: int) -> Optional[dict]:
        for blk in self._writer.iter_records():
            if int(blk.get("height", -1)) == height:
                return blk
        return None

    def count(self) -> int:
        return self._writer.count()

    def truncate(self) -> None:
        self._writer.truncate()


class CanonLogLedger:
    """JSONL для записей canon-аудита; поддерживает выборку `since_id`."""

    def __init__(self, path: str) -> None:
        self.path = path
        self._writer = JsonlAppender(path)

    def append(self, entry: dict) -> None:
        self._writer.append(entry)

    def append_many(self, entries: Iterable[dict]) -> int:
        return self._writer.append_many(entries)

    def read_since(self, since_id: int, limit: int = 200) -> List[dict]:
        out: List[dict] = []
        for entry in self._writer.iter_records():
            try:
                eid = int(entry.get("id", -1))
            except (TypeError, ValueError):
                continue
            if eid > since_id:
                out.append(entry)
            if len(out) >= limit:
                break
        return out

    def read_tail(self, n: int) -> List[dict]:
        if n <= 0:
            return []
        buf: List[dict] = []
        for entry in self._writer.iter_records():
            buf.append(entry)
            if len(buf) > n:
                buf.pop(0)
        return buf

    def max_id(self) -> int:
        m = -1
        for entry in self._writer.iter_records():
            try:
                eid = int(entry.get("id", -1))
                if eid > m:
                    m = eid
            except (TypeError, ValueError):
                continue
        return m

    def count(self) -> int:
        return self._writer.count()

    def truncate(self) -> None:
        self._writer.truncate()


def index_block_txs(block: dict) -> List[dict]:
    """Извлечь tx-записи блока для построения tx-индекса.

    Возвращает список dict со схемой:
    {"tx_hash": ..., "height": ..., "tx_idx": ..., "tx_type": ..., "sender": ..., "receiver": ...}
    """
    out: List[dict] = []
    height = int(block.get("height", -1))
    for idx, tx in enumerate(block.get("transactions") or []):
        h = tx.get("tx_hash")
        if not h:
            continue
        out.append(
            {
                "tx_hash": str(h),
                "height": height,
                "tx_idx": idx,
                "tx_type": tx.get("tx_type"),
                "sender": tx.get("sender"),
                "receiver": tx.get("receiver"),
            }
        )
    return out
