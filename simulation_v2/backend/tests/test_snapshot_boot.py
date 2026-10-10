"""Snapshot fast boot vs replay fallback."""

from __future__ import annotations

import json
from pathlib import Path

from tests.conftest import write_genesis
from volnix.node.node import Node


def _make_node(tmp_path: Path, data_name: str = "data") -> Node:
    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(data_dir=tmp_path / data_name, genesis_path=g, auto_declare=True)
    n.load_or_init()
    return n


def test_snapshot_boot_skips_replay(tmp_path: Path):
    n1 = _make_node(tmp_path)
    for _ in range(3):
        assert n1.produce_block() is not None
    tip = n1.app.state.height
    app_hash = n1.app.state.app_hash()
    snap = n1.snapshot.load()
    assert snap is not None
    assert snap.get("version") == 1

    n2 = Node(data_dir=tmp_path / "data", genesis_path=tmp_path / "genesis.default.json", auto_declare=True)
    n2.load_or_init()
    assert n2.booted_from_snapshot is True
    assert n2.app.state.height == tip
    assert n2.app.state.app_hash() == app_hash
    # Can continue producing after snapshot boot.
    assert n2.produce_block() is not None
    assert n2.app.state.height == tip + 1


def test_corrupt_snapshot_falls_back_to_replay(tmp_path: Path):
    n1 = _make_node(tmp_path)
    for _ in range(2):
        assert n1.produce_block() is not None
    tip = n1.app.state.height
    app_hash = n1.app.state.app_hash()

    # Poison the snapshot envelope.
    bad = {
        "version": 1,
        "app": {"height": tip, "accounts": {}, "params": {}, "chain_id": "volnix-sim-2"},
        "validator_set": {"validators": [], "proposer": ""},
        "last_commit": None,
        "consensus_hash": "dead",
    }
    (tmp_path / "data" / "app_snapshot.json").write_text(json.dumps(bad), encoding="utf-8")

    n2 = Node(data_dir=tmp_path / "data", genesis_path=tmp_path / "genesis.default.json", auto_declare=True)
    n2.load_or_init()
    assert n2.booted_from_snapshot is False
    assert n2.app.state.height == tip
    assert n2.app.state.app_hash() == app_hash


def test_legacy_bare_snapshot_triggers_replay(tmp_path: Path):
    n1 = _make_node(tmp_path)
    assert n1.produce_block() is not None
    tip = n1.app.state.height
    # Old format: raw AppState snapshot without version envelope.
    bare = n1.app.state.to_snapshot()
    (tmp_path / "data" / "app_snapshot.json").write_text(json.dumps(bare), encoding="utf-8")

    n2 = Node(data_dir=tmp_path / "data", genesis_path=tmp_path / "genesis.default.json", auto_declare=True)
    n2.load_or_init()
    assert n2.booted_from_snapshot is False
    assert n2.app.state.height == tip
