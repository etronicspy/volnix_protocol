from __future__ import annotations

import json
from pathlib import Path
from typing import Any


class AppSnapshotStore:
    """Optional cache. Source of truth remains blocks.jsonl."""

    def __init__(self, data_dir: Path) -> None:
        self.path = data_dir / "app_snapshot.json"

    def save(self, snapshot: dict[str, Any]) -> None:
        self.path.parent.mkdir(parents=True, exist_ok=True)
        tmp = self.path.with_suffix(".json.tmp")
        tmp.write_text(json.dumps(snapshot, separators=(",", ":"), ensure_ascii=True), encoding="utf-8")
        tmp.replace(self.path)

    def load(self) -> dict[str, Any] | None:
        if not self.path.exists():
            return None
        return json.loads(self.path.read_text(encoding="utf-8"))

    def clear(self) -> None:
        if self.path.exists():
            self.path.unlink()
