"""Operator HTTP must not write app state outside a committed block."""

from pathlib import Path

from fastapi.testclient import TestClient

from config.settings import Settings
from tests.conftest import write_genesis
from volnix.api.app import create_app


def test_operator_does_not_touch_state_until_block(tmp_path: Path):
    g = write_genesis(tmp_path / "genesis.default.json")
    settings = Settings(
        data_dir=tmp_path / "data",
        genesis_path=g,
        auto_produce=False,
        auto_declare=True,
    )
    app = create_app(settings, auto_produce=False)
    with TestClient(app) as client:
        produced = client.post("/api/v1/operator/produce", json={"count": 1})
        assert produced.status_code == 200
        assert produced.json()["produced"] == [1]
        node = app.state.node
        before = set(node.app.state.accounts)

        derived = client.post("/api/v1/operator/account", json={"seed": "wallet-consensus-seed"})
        assert derived.status_code == 200
        body = derived.json()
        address = body["address"]
        assert body["account"] is None
        assert address not in node.app.state.accounts
        assert set(node.app.state.accounts) == before

        missing = client.get(f"/api/v1/accounts/{address}")
        assert missing.status_code == 404

        minted = client.post(
            "/api/v1/operator/mint",
            json={"address": address, "denom": "uwrt", "amount": 10_000_000},
        )
        assert minted.status_code == 200
        assert minted.json()["broadcast"]["code"] == 0
        assert address not in node.app.state.accounts

        produced2 = client.post("/api/v1/operator/produce", json={"count": 1})
        assert produced2.json()["produced"] == [2]
        acc = node.app.state.accounts[address]
        assert acc.wrt == 10_000_000
        assert acc.created_height == 2
        assert acc.sequence == 0
