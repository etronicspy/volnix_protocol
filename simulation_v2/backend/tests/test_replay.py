from volnix.app.txutil import build_tx
from volnix.crypto.keys import derive_keypair
from volnix.types.msgs import MsgDeclareParticipation, MsgSend, MsgVerifyIdentity


def test_auto_declare_burns_entry_and_bs(node, genesis_kp):
    """Stand auto-declare: each block burns f_i + b_i + s_i (§6.3(5) package)."""
    acc = node.app.state.accounts[genesis_kp.address]
    before = acc.ant
    block = node.produce_block()
    assert block is not None
    assert len(block.data.get("txs", [])) == 1
    povb = node.app.state.last_povb
    assert povb["set_updated"] is True
    assert povb["declares"][0]["f_i"] == 20_000
    assert povb["declares"][0]["b_i"] == 500_000
    assert povb["declares"][0]["s_i"] == 400_000
    # f + b + s = 920_000 micro
    assert node.app.state.accounts[genesis_kp.address].ant == before - 920_000
    res = node.results.get(1)
    assert res is not None
    burns = [e for e in res.end_block_events if e.type == "consensus.burn_executed"]
    kinds = {
        a.value for e in burns for a in e.attributes if a.key == "kind"
    }
    assert "f_i" in kinds
    assert "b_s" in kinds


def test_auto_declare_disabled_no_burn(tmp_path, genesis_kp):
    from tests.conftest import write_genesis
    from volnix.node.node import Node

    g = write_genesis(tmp_path / "genesis.default.json")
    n = Node(
        data_dir=tmp_path / "data",
        genesis_path=g,
        produce_interval=0.01,
        auto_declare=False,
    )
    n.load_or_init()
    before = n.app.state.accounts[genesis_kp.address].ant
    block = n.produce_block()
    assert block is not None
    assert block.data.get("txs", []) == []
    assert n.app.state.accounts[genesis_kp.address].ant == before


def test_produce_and_replay_app_hash(fast_node, genesis_kp):
    st = fast_node.app.state
    # declare so corridor can pass: L=1e6, need b in ~[333334, 666666]
    tx = build_tx(
        st,
        genesis_kp,
        [MsgDeclareParticipation(validator=genesis_kp.address, b_i=400_000, s_i=200_000)],
    )
    out = fast_node.broadcast_tx(tx)
    assert out["code"] == 0, out
    block = fast_node.produce_block()
    assert block is not None
    assert block.header.height == 1
    assert block.header.app_hash
    assert block.header.data_hash
    assert block.header.validators_hash
    # second empty-ish block
    tx2 = build_tx(
        fast_node.app.state,
        genesis_kp,
        [MsgDeclareParticipation(validator=genesis_kp.address, b_i=400_000, s_i=200_000)],
    )
    fast_node.broadcast_tx(tx2)
    b2 = fast_node.produce_block()
    assert b2 is not None
    expected = fast_node.app.state.app_hash()
    verified = fast_node.replay_verify()
    assert verified == expected
    assert verified == b2.header.app_hash


def test_msgsend_wrt_via_node(fast_node, genesis_kp):
    # Canon §6.3: no WRT premint — earn subsidy, then MsgSend.
    other = derive_keypair("citizen-1")
    fast_node.app.state.ensure_account(other.address, other.pub_hex)
    assert fast_node.app.state.accounts[genesis_kp.address].wrt == 0
    assert fast_node.produce_block() is not None  # auto-declare → subsidy
    assert fast_node.app.state.accounts[genesis_kp.address].wrt >= 100
    tx = build_tx(
        fast_node.app.state,
        genesis_kp,
        [MsgSend(from_address=genesis_kp.address, to_address=other.address, denom="uwrt", amount=100)],
    )
    assert fast_node.broadcast_tx(tx)["code"] == 0
    block = fast_node.produce_block()
    assert block is not None
    assert fast_node.app.state.accounts[other.address].wrt == 100
    # ANT send rejected
    tx_bad = build_tx(
        fast_node.app.state,
        genesis_kp,
        [MsgSend(from_address=genesis_kp.address, to_address=other.address, denom="uant", amount=1)],
    )
    assert fast_node.broadcast_tx(tx_bad)["code"] == 0  # mempool check is light
    b2 = fast_node.produce_block()
    assert b2 is not None
    res = fast_node.results.get(b2.header.height)
    assert res is not None
    assert any(r.code != 0 for r in res.txs_results)


def test_verify_supplier_and_epoch_boundary(fast_node, genesis_kp):
    sup = derive_keypair("supplier-1")
    fast_node.app.state.ensure_account(sup.address, sup.pub_hex)
    tx = build_tx(
        fast_node.app.state,
        sup,
        [
            MsgVerifyIdentity(
                address=sup.address,
                zkp_proof="zkp-sup-1",
                verification_provider="sim",
                desired_role="supplier",
            )
        ],
    )
    assert fast_node.broadcast_tx(tx)["code"] == 0
    # produce until epoch boundary (epoch_blocks=3)
    for _ in range(3):
        d = build_tx(
            fast_node.app.state,
            genesis_kp,
            [MsgDeclareParticipation(validator=genesis_kp.address, b_i=400_000, s_i=200_000)],
        )
        fast_node.broadcast_tx(d)
        block = fast_node.produce_block()
        assert block is not None
    assert fast_node.app.state.height == 3
    assert fast_node.app.state.epoch == 1
    # genesis ANT wiped; supplier received ANT_emit = L_total * 3
    assert fast_node.app.state.accounts[genesis_kp.address].ant == 0
    assert fast_node.app.state.accounts[sup.address].role == "supplier"
    assert fast_node.app.state.accounts[sup.address].ant == fast_node.app.state.l_total() * 3
