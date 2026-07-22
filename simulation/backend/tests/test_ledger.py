"""Append-only JSONL ledger + tx-индекс + WS-дельта."""
from __future__ import annotations

import time
import uuid

import pytest

from core.engine import BURN_CAP_LAMBDA
from core.ledger import BlockLedger, CanonLogLedger, JsonlAppender, index_block_txs
from core.models import Role, Transaction, TransactionType
from core.state import GENESIS_VALIDATOR_ADDR


def test_jsonl_append_and_iter(tmp_path):
    path = str(tmp_path / "blocks.jsonl")
    w = JsonlAppender(path)
    w.append({"height": 1, "hash": "a"})
    w.append({"height": 2, "hash": "b"})
    rows = list(w.iter_records())
    assert rows == [{"height": 1, "hash": "a"}, {"height": 2, "hash": "b"}]
    assert w.count() == 2
    w.truncate()
    assert w.count() == 0


def test_jsonl_skip_malformed_lines(tmp_path):
    path = str(tmp_path / "broken.jsonl")
    w = JsonlAppender(path)
    w.append({"id": 1})
    with open(path, "a") as f:
        f.write("not-json\n")
    w.append({"id": 2})
    assert [r["id"] for r in w.iter_records()] == [1, 2]


def test_block_ledger_range_and_tail(tmp_path):
    bl = BlockLedger(str(tmp_path / "blocks.jsonl"))
    for h in range(10):
        bl.append_block({"height": h, "hash": f"h{h}"})
    assert bl.count() == 10
    rng = bl.read_range(3, 5)
    assert [b["height"] for b in rng] == [3, 4, 5]
    tail = bl.read_tail(3)
    assert [b["height"] for b in tail] == [7, 8, 9]
    assert bl.get_by_height(4) == {"height": 4, "hash": "h4"}
    assert bl.get_by_height(99) is None


def test_canon_log_ledger_since(tmp_path):
    cl = CanonLogLedger(str(tmp_path / "canon.jsonl"))
    for i in range(1, 6):
        cl.append({"id": i, "title": f"e{i}"})
    assert cl.count() == 5
    assert cl.max_id() == 5
    entries = cl.read_since(2, limit=10)
    assert [e["id"] for e in entries] == [3, 4, 5]


def test_index_block_txs():
    block = {
        "height": 7,
        "transactions": [
            {"tx_hash": "h1", "tx_type": "transfer", "sender": "a", "receiver": "b"},
            {"tx_hash": "h2", "tx_type": "mint", "sender": "x", "receiver": "a"},
            {"tx_type": "no_hash"},  # пропускаем
        ],
    }
    out = index_block_txs(block)
    assert [r["tx_hash"] for r in out] == ["h1", "h2"]
    assert out[0]["height"] == 7
    assert out[0]["tx_idx"] == 0
    assert out[1]["tx_idx"] == 1


def test_state_manager_writes_jsonl_on_genesis(state_manager, tmp_path):
    """init_genesis должен записать genesis-блок в blocks.jsonl."""
    assert state_manager.block_ledger.count() >= 1
    g = state_manager.block_ledger.get_by_height(0)
    assert g is not None
    assert g["tx_count"] >= 3


def test_tx_index_built_from_genesis(state_manager):
    """tx_index должен содержать genesis-транзакции."""
    g = state_manager.blocks[0]
    for tx in g["transactions"]:
        h = tx.get("tx_hash")
        if h:
            assert h in state_manager.tx_index
            rec = state_manager.tx_index[h]
            assert rec["height"] == 0


def test_get_tx_record_returns_full_tx(state_manager):
    g = state_manager.blocks[0]
    first_tx = g["transactions"][0]
    rec = state_manager.get_tx_record(first_tx["tx_hash"])
    assert rec is not None
    assert rec["height"] == 0
    assert "tx" in rec


def test_account_history_for_genesis_validator(state_manager):
    history = state_manager.get_account_tx_history(GENESIS_VALIDATOR_ADDR, limit=10)
    assert history, "genesis validator должен иметь начальные tx"
    for rec in history:
        assert rec["height"] == 0


@pytest.mark.asyncio
async def test_block_ledger_grows_after_produce(engine):
    """Каждый успешный блок добавляет строку в blocks.jsonl."""
    initial = engine.state.block_ledger.count()
    gv = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    L = engine._network_lzn_total_validators()
    tx = Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=gv.address,
        amount=BURN_CAP_LAMBDA * L,
        stake_amount=0.0,
        asset_type="ant",
        timestamp=time.time(),
    )
    engine.state.mempool.append(tx)
    await engine.produce_block()
    assert engine.state.block_ledger.count() == initial + 1


def test_compute_delta_detects_changes(state_manager, mk_account):
    a = mk_account("acc1", wrt=10.0)
    before_accounts = state_manager.snapshot_accounts()
    before_orders = state_manager.snapshot_orders()
    a.wrt_balance = 25.0
    mk_account("acc2", wrt=5.0)
    delta = state_manager.compute_delta(before_accounts, before_orders)
    assert "acc1" in delta["accounts_changed"]
    assert "acc2" in delta["accounts_changed"]
    assert delta["accounts_removed"] == []
    assert delta["orders_changed"] == {}


def test_compute_delta_detects_removed_account(state_manager, mk_account):
    a = mk_account("acc_doomed", wrt=10.0)
    before_accounts = state_manager.snapshot_accounts()
    before_orders = state_manager.snapshot_orders()
    del state_manager.accounts[a.address]
    delta = state_manager.compute_delta(before_accounts, before_orders)
    assert "acc_doomed" in delta["accounts_removed"]


def test_canon_log_persists_to_jsonl(state_manager, tmp_path):
    """Записи canon_log должны попадать в canon_log.jsonl, если persist включён."""
    state_manager.canon_log.push(
        source="test",
        status="ok",
        category="test",
        canon="§0",
        title="hello",
    )
    ledger = state_manager._canon_ledger
    assert ledger is not None
    assert ledger.count() >= 1
    tail = ledger.read_tail(1)
    assert tail and tail[0]["title"] == "hello"


def test_reset_state_truncates_ledgers(state_manager):
    initial_blocks = state_manager.block_ledger.count()
    assert initial_blocks >= 1
    state_manager.reset_state()
    # После reset → genesis заново → ровно 1 блок в ledger
    assert state_manager.block_ledger.count() == 1
