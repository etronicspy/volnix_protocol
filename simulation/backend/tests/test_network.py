"""NetworkSim: per-node mempool, gossip latency/loss, flush_to_global ≥ quorum."""
from __future__ import annotations

import time
import uuid

import pytest

from core.models import Transaction, TransactionType
from core.network import NetworkSim, _stable_node_for_addr
from core.state import GENESIS_VALIDATOR_ADDR, SIM_TREASURY_ADDR, StateManager


def _mk_tx(sender: str = "alice") -> Transaction:
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.TRANSFER,
        sender=sender,
        receiver="bob",
        amount=1.0,
        asset_type="wrt",
        timestamp=time.time(),
    )


def test_genesis_addresses_pinned_to_node_0():
    net = NetworkSim(num_nodes=4)
    sm = StateManager()
    sm.init_genesis()
    net.attach(sm)
    assert net.node_id_for_addr(GENESIS_VALIDATOR_ADDR) == "node_0"
    assert net.node_id_for_addr(SIM_TREASURY_ADDR) == "node_0"


def test_register_address_stable():
    net = NetworkSim(num_nodes=4)
    a1 = net.register_address("bot_aaa")
    a2 = net.register_address("bot_aaa")
    assert a1 == a2
    # детерминированно одинаков с pure-функцией
    assert a1 == _stable_node_for_addr("bot_aaa", 4)


def test_single_node_fastpath_immediate_flush():
    net = NetworkSim(num_nodes=1)
    tx = _mk_tx()
    net.submit_to("node_0", tx)
    out = net.flush_to_global()
    assert [t.tx_hash for t in out] == [tx.tx_hash]


def test_quorum_blocks_flush_until_majority():
    net = NetworkSim(num_nodes=4, gossip_latency_ms=0, gossip_loss_pct=100.0, quorum_pct=2.0 / 3.0)
    tx = _mk_tx()
    # Submit on one node only; loss=100% означает, что gossip-доставка не пройдёт.
    net.submit_to("node_0", tx)
    assert net.flush_to_global() == []
    # принудительно "доставим" tx ещё на 2 узла → 3/4 ≥ 2/3 → flush
    net._deliver(tx, "node_1", time.time())
    net._deliver(tx, "node_2", time.time())
    out = net.flush_to_global()
    assert [t.tx_hash for t in out] == [tx.tx_hash]
    # повторный flush уже не отдаст ту же tx (consumed)
    assert net.flush_to_global() == []


def test_loss_zero_means_full_propagation_with_zero_latency():
    net = NetworkSim(num_nodes=3, gossip_latency_ms=0, gossip_loss_pct=0.0)
    tx = _mk_tx()
    net.submit_to("node_0", tx)
    # event-loop вне теста → fastpath fallback в _schedule_gossip синхронно доставляет.
    out = net.flush_to_global()
    assert [t.tx_hash for t in out] == [tx.tx_hash]


def test_set_config_runtime():
    net = NetworkSim(num_nodes=2, gossip_latency_ms=100, gossip_loss_pct=0.0)
    cfg = net.set_config(gossip_latency_ms=250, gossip_loss_pct=10.0, quorum_pct=0.5)
    assert cfg == {
        "num_nodes": 2,
        "gossip_latency_ms": 250,
        "gossip_loss_pct": 10.0,
        "quorum_pct": 0.5,
    }


def test_nodes_summary_marks_proposer():
    net = NetworkSim(num_nodes=2)
    sm = StateManager()
    sm.init_genesis()
    net.attach(sm)
    summary = net.nodes_summary({"proposer": GENESIS_VALIDATOR_ADDR, "consensus_round_log": []})
    n0 = next(n for n in summary if n["id"] == "node_0")
    assert n0["is_proposer"] is True


def test_attach_imports_existing_mempool():
    sm = StateManager()
    sm.init_genesis()
    tx = _mk_tx(sender=GENESIS_VALIDATOR_ADDR)
    sm.mempool.append(tx)
    net = NetworkSim(num_nodes=3)
    net.attach(sm)
    # legacy импорт → full coverage → сразу доступен через flush
    out = net.flush_to_global()
    assert [t.tx_hash for t in out] == [tx.tx_hash]
    assert sm.mempool == []
