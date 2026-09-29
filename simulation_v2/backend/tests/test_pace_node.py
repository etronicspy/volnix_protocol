"""Node integration for adaptive attempt window (§6.2) without last_applied replay."""

from volnix.app.modules.mint import block_reward
from volnix.app.state import Account
from volnix.app.txutil import build_tx
from volnix.crypto.keys import derive_keypair
from volnix.node.node import Node
from volnix.types.msgs import MsgDeclareParticipation, MsgVerifyIdentity
from volnix.types.role import Role
from tests.conftest import write_genesis


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


def test_invalid_only_declares_are_empty_attempt(tmp_path, genesis_kp):
    """Declare txs that fail EndBlocker validation must not finalize height (§5.4/§6.2)."""
    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(data_dir=tmp_path / "data", genesis_path=g, auto_declare=False)
    n.load_or_init()
    h0 = n.app.state.height
    wrt0 = n.app.state.wrt_supply
    t0 = n.pace.attempt_window_sec

    # Passes deliver_tx (recorded), fails step-1 validation: f+b+s > L_i
    tx = build_tx(
        n.app.state,
        genesis_kp,
        [MsgDeclareParticipation(validator=genesis_kp.address, b_i=900_000, s_i=900_000)],
    )
    assert n.broadcast_tx(tx)["code"] == 0
    assert n.produce_block() is None
    assert n.app.state.height == h0
    assert n.app.state.wrt_supply == wrt0

    n.pace.on_empty_attempt()
    assert n.pace.attempt_window_sec == t0 // 2
    assert n.pace.missed_budget_sec == t0


def test_valid_declare_finalizes_and_mints_subsidy(tmp_path, genesis_kp):
    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(data_dir=tmp_path / "data", genesis_path=g, auto_declare=False)
    n.load_or_init()
    h0 = n.app.state.height
    wrt0 = n.app.state.wrt_supply
    reward = block_reward(
        h0 + 1, n.app.state.params.base_block_reward, n.app.state.params.halving_interval
    )

    tx = build_tx(
        n.app.state,
        genesis_kp,
        [MsgDeclareParticipation(validator=genesis_kp.address, b_i=400_000, s_i=200_000)],
    )
    assert n.broadcast_tx(tx)["code"] == 0
    block = n.produce_block()
    assert block is not None
    assert n.app.state.height == h0 + 1
    assert n.app.state.wrt_supply == wrt0 + reward
    assert n.app.state.set_updated is True


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
    # missed=150 → debt=2; first success converts without consuming
    assert n.pace.pace_debt_blocks == 2
    assert n.pace.attempt_window_sec == 1
    b2 = declare_and_produce()
    assert b2 is not None
    assert n.pace.pace_debt_blocks == 1
    assert n.pace.attempt_window_sec == 1
    b3 = declare_and_produce()
    assert b3 is not None
    assert n.pace.pace_debt_blocks == 0
    assert n.pace.attempt_window_sec == 60


def test_validator_without_lzn_declare_no_height(tmp_path, genesis_kp):
    """Fresh validator with zero activated LZN cannot finalize a height alone."""
    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(data_dir=tmp_path / "data", genesis_path=g, auto_declare=False)
    n.load_or_init()
    newbie = derive_keypair("pace-node-no-lzn")
    n.app.state.accounts[newbie.address] = Account(
        address=newbie.address, pub_hex=newbie.pub_hex, role=Role.CITIZEN
    )
    vtx = build_tx(
        n.app.state,
        newbie,
        [
            MsgVerifyIdentity(
                address=newbie.address, desired_role="validator", zkp_proof="stub-zkp-no-lzn"
            )
        ],
    )
    gtx = build_tx(
        n.app.state,
        genesis_kp,
        [MsgDeclareParticipation(validator=genesis_kp.address, b_i=400_000, s_i=200_000)],
    )
    assert n.broadcast_tx(gtx)["code"] == 0
    assert n.broadcast_tx(vtx)["code"] == 0
    assert n.produce_block() is not None
    assert n.app.state.accounts[newbie.address].role == Role.VALIDATOR

    h = n.app.state.height
    bad = build_tx(
        n.app.state,
        newbie,
        [MsgDeclareParticipation(validator=newbie.address, b_i=1, s_i=1)],
    )
    assert n.broadcast_tx(bad)["code"] == 0
    assert n.produce_block() is None
    assert n.app.state.height == h
