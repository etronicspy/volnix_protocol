"""Тесты CSV/JSONL экспортёров."""
from __future__ import annotations

import json

from core import exporters
from core.models import Role


def test_blocks_jsonl_contains_genesis(state_manager):
    text = exporters.blocks_jsonl(state_manager)
    lines = [line for line in text.split("\n") if line]
    assert len(lines) == 1
    blk = json.loads(lines[0])
    assert blk["height"] == 0


def test_blocks_jsonl_range_slice(state_manager):
    state_manager.blocks.append({"height": 1, "transactions": []})
    state_manager.blocks.append({"height": 2, "transactions": []})
    state_manager.current_height = 2
    text = exporters.blocks_jsonl(state_manager, from_height=1, to_height=1)
    lines = [line for line in text.split("\n") if line]
    assert len(lines) == 1
    assert json.loads(lines[0])["height"] == 1


def test_balances_csv_format(state_manager, mk_account):
    a = mk_account("alice", role=Role.PROVIDER)
    a.wrt_balance = 100
    a.ant_balance = 5
    csv_text = exporters.balances_csv(state_manager)
    lines = csv_text.strip().split("\n")
    assert lines[0] == "address,role,zkp_verified,wrt,lzn,lzn_frozen_mining,ant"
    assert any("alice,provider" in row for row in lines[1:])


def test_ticks_csv_includes_header(state_manager):
    state_manager.price_history.append({"time": 1.0, "price": 1.23, "ts": 100})
    csv_text = exporters.ticks_csv(state_manager)
    assert csv_text.split("\n")[0] == "time,price,ts"
    assert "1.23" in csv_text


def test_canon_log_jsonl_respects_limit(state_manager):
    for _ in range(10):
        state_manager.canon_log.push(
            source="test",
            status="ok",
            category="declare",
            canon="§5.4",
            title="test",
            detail="",
        )
    text = exporters.canon_log_jsonl(state_manager, limit=3)
    lines = [line for line in text.split("\n") if line]
    assert len(lines) == 3
    entry = json.loads(lines[0])
    assert entry["status"] == "ok"
