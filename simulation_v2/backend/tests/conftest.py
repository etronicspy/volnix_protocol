from __future__ import annotations

import json
from pathlib import Path

import pytest

from volnix.app.genesis import load_genesis_doc
from volnix.crypto.keys import derive_keypair
from volnix.node.node import Node

BACKEND = Path(__file__).resolve().parent.parent
DEFAULT_GENESIS = BACKEND / "config" / "genesis.default.json"


def genesis_doc(**param_overrides) -> dict:
    doc = load_genesis_doc(DEFAULT_GENESIS)
    doc["app_state"]["params"].update(param_overrides)
    return doc


def write_genesis(path: Path, **param_overrides) -> Path:
    doc = genesis_doc(**param_overrides)
    path.write_text(json.dumps(doc), encoding="utf-8")
    return path


@pytest.fixture
def genesis_kp():
    return derive_keypair("volnix-genesis-validator-v2")


@pytest.fixture
def node(tmp_path: Path) -> Node:
    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(data_dir=tmp_path / "data", genesis_path=g)
    n.load_or_init()
    return n


@pytest.fixture
def fast_node(tmp_path: Path) -> Node:
    g = write_genesis(tmp_path / "genesis.default.json", epoch_blocks=3, moa_validator_window=10_000)
    n = Node(data_dir=tmp_path / "data", genesis_path=g)
    n.load_or_init()
    return n
