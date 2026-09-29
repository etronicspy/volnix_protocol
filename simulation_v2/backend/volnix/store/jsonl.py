"""Append-only JSONL file with sequential reads."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Iterator, TextIO


class JsonlStore:
    def __init__(self, path: Path) -> None:
        self.path = path
        self.path.parent.mkdir(parents=True, exist_ok=True)
        self._fh: TextIO | None = None

    def _writer(self) -> TextIO:
        if self._fh is None or self._fh.closed:
            self._fh = self.path.open("a", encoding="utf-8")
        return self._fh

    def append(self, record: dict[str, Any]) -> None:
        self._writer().write(json.dumps(record, separators=(",", ":"), ensure_ascii=True) + "\n")

    def flush(self) -> None:
        if self._fh is not None and not self._fh.closed:
            self._fh.flush()

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
        if self.path.exists():
            self.path.unlink()
