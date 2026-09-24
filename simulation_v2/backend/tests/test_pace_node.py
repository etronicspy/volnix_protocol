"""Node integration for adaptive attempt window (§6.2) without last_applied replay."""

from volnix.app.txutil import build_tx
from volnix.types.msgs import MsgDeclareParticipation
from tests.conftest import write_genesis
from volnix.node.node import Node


def test_empty_attempts_shrink_window_without_height(tmp_path, genesis_kp):
    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(data_dir=tmp_path / "data", genesis_path=g, auto_declare=False)
    n.load_or_init()
    h0 = n.app.state.height
    assert n.produce_block() is None
    assert n.app.state.height == h0
    n.pace.on_empty_attempt()
    assert n.pace.attempt_window_sec == 30
    assert n.pace.missed_budget_sec == 60
    n.pace.on_empty_attempt()
    assert n.pace.attempt_window_sec == 15
    assert n.pace.missed_budget_sec == 90
    assert n.app.state.height == h0


def test_catchup_after_declares_return(tmp_path, genesis_kp):
    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(data_dir=tmp_path / "data", genesis_path=g, auto_declare=False)
    n.load_or_init()
    n.pace.missed_budget_sec = 150
    n.pace.attempt_window_sec = 1

    def declare_and_produce():
        tx = build_tx(
            n.app.state,
            genesis_kp,
            [MsgDeclareParticipation(validator=genesis_kp.address, b_i=400_000, s_i=200_000)],
        )
        assert n.broadcast_tx(tx)["code"] == 0
        return n.produce_block()

    b1 = declare_and_produce()
    assert b1 is not None
    assert n.pace.pace_debt_blocks == 1
    assert n.pace.attempt_window_sec == 1
    b2 = declare_and_produce()
    assert b2 is not None
    assert n.pace.pace_debt_blocks == 0
    assert n.pace.attempt_window_sec == 60
