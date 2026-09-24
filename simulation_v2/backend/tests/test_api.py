from pathlib import Path

from fastapi.testclient import TestClient

from config.settings import Settings
from tests.conftest import write_genesis
from volnix.api.app import create_app


def test_explorer_and_rpc(tmp_path: Path):
    g = write_genesis(tmp_path / "genesis.default.json", epoch_blocks=10)
    settings = Settings(
        data_dir=tmp_path / "data",
        genesis_path=g,
        auto_produce=False,
    )
    app = create_app(settings, auto_produce=False)
    with TestClient(app) as client:
        r = client.get("/")
        assert r.status_code == 200
        assert r.json()["chain_id"] == "volnix-sim-2"
        st = client.get("/status")
        assert st.status_code == 200
        assert st.json()["result"]["sync_info"]["latest_block_height"] == "0"
        summary = client.get("/api/v1/chain/summary")
        assert summary.status_code == 200
        body = summary.json()
        assert body["height"] == 0
        assert body["params"]["k"] == 150
        assert body["attempt_window_sec"] == 60
        assert body["pace_debt_blocks"] == 0
        assert body["missed_budget_sec"] == 0
        assert body["time_scale"] == 60.0
        assert body["wall_sleep_sec"] == 1.0
        assert body["produce_interval_sec"] == 1.0
        assert body["auto_produce"] is False
        blocks = client.get("/api/v1/blocks")
        assert blocks.status_code == 200
        assert blocks.json()["latest"] == 0
        produced = client.post("/api/v1/operator/produce", json={"count": 1})
        assert produced.status_code == 200
        assert produced.json()["produced"] == [1]

        pace = client.get("/api/v1/operator/pace")
        assert pace.status_code == 200
        assert pace.json()["attempt_window_sec"] == 60
        assert pace.json()["base_block_time"] == 60
        assert pace.json()["time_scale"] == 60.0
        assert pace.json()["wall_sleep_sec"] == 1.0

        ts = client.post("/api/v1/operator/time-scale", json={"time_scale": 120})
        assert ts.status_code == 200
        assert ts.json()["time_scale"] == 120.0
        assert ts.json()["wall_sleep_sec"] == 0.5
        assert ts.json()["produce_interval_sec"] == 0.5
        bad_ts = client.post("/api/v1/operator/time-scale", json={"time_scale": 0})
        assert bad_ts.status_code == 422

        interval = client.get("/api/v1/operator/produce-interval")
        assert interval.status_code == 200
        assert interval.json()["interval_sec"] == 0.5
        assert interval.json()["attempt_window_sec"] == 60
        assert interval.json()["time_scale"] == 120.0

        # Simulate empty attempts then reset via POST
        node = app.state.node
        node.pace.on_empty_attempt()
        assert node.pace.attempt_window_sec == 30
        reset = client.post("/api/v1/operator/pace", json={"reset": True})
        assert reset.status_code == 200
        assert reset.json()["attempt_window_sec"] == 60
        assert reset.json()["missed_budget_sec"] == 0
        assert reset.json()["time_scale"] == 120.0
        assert reset.json()["wall_sleep_sec"] == 0.5

        summary_iv = client.get("/api/v1/chain/summary")
        assert summary_iv.status_code == 200
        assert summary_iv.json()["attempt_window_sec"] == 60
        assert summary_iv.json()["time_scale"] == 120.0
        assert summary_iv.json()["produce_interval_sec"] == 0.5

        accounts = client.get("/api/v1/accounts")
        assert accounts.status_code == 200
        body = accounts.json()
        assert body["count"] >= 1
        row = body["accounts"][0]
        assert "address" in row and "role" in row
        assert "wrt_display" in row and "ant_display" in row
        assert row["role"] == "validator"
        assert row["in_validator_set"] is True
        b1 = client.get("/api/v1/blocks/1")
        assert b1.status_code == 200
        body1 = b1.json()
        assert body1["block"]["header"]["height"] == 1
        assert "consensus" in body1
        cons = body1["consensus"]
        assert cons["next_set"] is not None
        assert cons["signing_set"] is not None
        assert cons["commit"] is not None
        assert cons["commit"]["height"] == 1
        assert cons["total_power"] >= 1
        assert cons["voted_power"] >= 1

        tape = client.get("/api/v1/blocks?tail=10")
        assert tape.status_code == 200
        rows = tape.json()["blocks"]
        assert any(r["height"] == 1 for r in rows)
        row1 = next(r for r in rows if r["height"] == 1)
        assert "set_updated" in row1
        assert "n_declares" in row1
        assert "sum_b" in row1
        assert "fill" in row1

        # Second block stores commit for height 1 on last_commit
        produced2 = client.post("/api/v1/operator/produce", json={"count": 1})
        assert produced2.status_code == 200
        b1b = client.get("/api/v1/blocks/1").json()
        assert b1b["consensus"]["commit"]["height"] == 1
        b2 = client.get("/api/v1/blocks/2").json()
        assert b2["consensus"]["commit"]["height"] == 2
        assert b2["block"]["last_commit"]["height"] == 1

        rpc_block = client.get("/block", params={"height": 1})
        assert rpc_block.status_code == 200
        search = client.get("/api/v1/search", params={"q": "1"})
        assert search.json()["kind"] == "block"
