"""Append-only JSONL file with sequential reads."""

from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Iterator


class JsonlStore:
    def __init__(self, path: Path) -> None:
        self.path = path
        self.path.parent.mkdir(parents=True, exist_ok=True)

    def append(self, record: dict[str, Any]) -> None:
        with self.path.open("a", encoding="utf-8") as fh:
            fh.write(json.dumps(record, separators=(",", ":"), ensure_ascii=True) + "\n")
            fh.flush()

    def iter_records(self) -> Iterator[dict[str, Any]]:
        if not self.path.exists():
            return
        with self.path.open("r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line:
                    continue
                yield json.loads(line)

    def clear(self) -> None:
        if self.path.exists():
            self.path.unlink()
