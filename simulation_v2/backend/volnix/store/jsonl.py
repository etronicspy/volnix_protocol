"""Append-only JSONL file with uint64 offset sidecar for O(1) record seeks."""

from __future__ import annotations

import json
import struct
from pathlib import Path
from typing import Any, BinaryIO, Iterator, TextIO

IDX_FMT = "<Q"
IDX_SIZE = struct.calcsize(IDX_FMT)


class JsonlStore:
    def __init__(self, path: Path) -> None:
        self.path = path
        self.idx_path = Path(str(path) + ".idx")
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._fh: TextIO | None = None
        self._idx_fh: BinaryIO | None = None
        self._count = 0
        self._ensure_index()

    def _ensure_index(self) -> None:
        if not self.path.exists():
            self._count = 0
            return
        if self.idx_path.exists():
            idx_size = self.idx_path.stat().st_size
            if idx_size % IDX_SIZE == 0:
                self._count = idx_size // IDX_SIZE
                # Cheap sync check: last offset + line must not past EOF.
                if self._count == 0:
                    if self.path.stat().st_size == 0:
                        return
                    self.rebuild_index()
                    return
                last_off = self._read_offset(self._count - 1)
                data_size = self.path.stat().st_size
                if last_off < data_size:
                    return
        self.rebuild_index()

    def _idx_writer(self) -> BinaryIO:
        if self._idx_fh is None or self._idx_fh.closed:
            self._idx_fh = self.idx_path.open("ab")
        return self._idx_fh

    def _writer(self) -> TextIO:
        if self._fh is None or self._fh.closed:
            self._fh = self.path.open("a", encoding="utf-8")
        return self._fh

    def _read_offset(self, index: int) -> int:
        with self.idx_path.open("rb") as fh:
            fh.seek(index * IDX_SIZE)
            raw = fh.read(IDX_SIZE)
            if len(raw) != IDX_SIZE:
                raise IndexError(f"idx record {index} missing")
            return struct.unpack(IDX_FMT, raw)[0]

    def rebuild_index(self) -> None:
        """Scan JSONL and rewrite `.idx` (byte offsets of each non-empty line)."""
        self.flush()
        if self._idx_fh is not None and not self._idx_fh.closed:
            self._idx_fh.close()
        self._idx_fh = None
        count = 0
        if not self.path.exists():
            if self.idx_path.exists():
                self.idx_path.unlink()
            self._count = 0
            return
        with self.path.open("rb") as data, self.idx_path.open("wb") as idx:
            while True:
                offset = data.tell()
                line = data.readline()
                if not line:
                    break
                if not line.strip():
                    continue
                idx.write(struct.pack(IDX_FMT, offset))
                count += 1
        self._count = count

    def append(self, record: dict[str, Any]) -> None:
        line = json.dumps(record, separators=(",", ":"), ensure_ascii=True) + "\n"
        fh = self._writer()
        fh.flush()
        offset = self.path.stat().st_size if self.path.exists() else 0
        self._idx_writer().write(struct.pack(IDX_FMT, offset))
        self._idx_writer().flush()
        fh.write(line)
        self._count += 1

    def flush(self) -> None:
        if self._fh is not None and not self._fh.closed:
            self._fh.flush()
        if self._idx_fh is not None and not self._idx_fh.closed:
            self._idx_fh.flush()

    def record_count(self) -> int:
        return self._count

    def read_record(self, index: int) -> dict[str, Any] | None:
        if index < 0 or index >= self._count:
            return None
        return self.read_at(self._read_offset(index))

    def read_at(self, offset: int) -> dict[str, Any] | None:
        self.flush()
        if not self.path.exists():
            return None
        with self.path.open("rb") as fh:
            fh.seek(offset)
            line = fh.readline()
        if not line:
            return None
        text = line.decode("utf-8").strip()
        if not text:
            return None
        return json.loads(text)

    def iter_indices_reversed(self) -> Iterator[int]:
        for i in range(self._count - 1, -1, -1):
            yield i

    def iter_records(self) -> Iterator[dict[str, Any]]:
        self.flush()
        if not self.path.exists():
            return
        with self.path.open("r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line:
                    continue
                yield json.loads(line)

    def clear(self) -> None:
        if self._fh is not None and not self._fh.closed:
            self._fh.close()
        self._fh = None
        if self._idx_fh is not None and not self._idx_fh.closed:
            self._idx_fh.close()
        self._idx_fh = None
        if self.path.exists():
            self.path.unlink()
        if self.idx_path.exists():
            self.idx_path.unlink()
        self._count = 0
