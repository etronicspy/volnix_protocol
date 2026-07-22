"""Smoke-тесты для REST API (новые explorer endpoints Этапа 2)."""
from __future__ import annotations

import pytest
from fastapi.testclient import TestClient


@pytest.fixture
def client(_isolated_data_dir):
    # Перезагружаем модуль main, чтобы поднять FastAPI-приложение в свежем VOLNIX_SIM_DATA_DIR
    import importlib

    import main as main_module

    importlib.reload(main_module)
    sm = main_module.state_manager
    sm.reset_state()
    yield TestClient(main_module.app)


def test_root_status(client):
    r = client.get("/")
    assert r.status_code == 200
    j = r.json()
    assert "block_height" in j


def test_state_returns_full_state(client):
    r = client.get("/api/state")
    assert r.status_code == 200
    j = r.json()
    assert "accounts" in j
    assert "blocks" in j


def test_blocks_default_tail(client):
    r = client.get("/api/blocks?tail=5")
    assert r.status_code == 200
    j = r.json()
    assert "blocks" in j
    assert isinstance(j["blocks"], list)


def test_block_by_height_genesis(client):
    r = client.get("/api/blocks/0")
    assert r.status_code == 200
    j = r.json()
    assert j["found"] is True
    assert j["height"] == 0
    assert j["transactions"]


def test_block_by_height_missing(client):
    r = client.get("/api/blocks/99999")
    assert r.status_code == 200
    j = r.json()
    assert j["found"] is False


def test_tx_found_for_genesis_tx(client):
    state = client.get("/api/state").json()
    first = state["blocks"][0]["transactions"][0]
    h = first["tx_hash"]
    r = client.get(f"/api/tx/{h}")
    assert r.status_code == 200
    j = r.json()
    assert j["found"] is True
    assert j["height"] == 0
    assert j["tx"]["tx_hash"] == h


def test_tx_not_found(client):
    r = client.get("/api/tx/0000deadbeef")
    assert r.status_code == 200
    assert r.json()["found"] is False


def test_account_history_genesis_validator(client):
    state = client.get("/api/state").json()
    gv = state["genesis_validator"]
    r = client.get(f"/api/account/{gv}/history?limit=20")
    assert r.status_code == 200
    j = r.json()
    assert j["address"] == gv
    assert isinstance(j["history"], list)
    assert j["history"], "у genesis-валидатора должна быть история"


def test_canon_log_endpoint(client):
    r = client.get("/api/canon-log?limit=5")
    assert r.status_code == 200
    assert "entries" in r.json()


def test_canon_log_since_id(client):
    # делаем какие-то изменения, чтобы запись попала в canon_log
    body = {"op": "transfer", "address": "bogus_addr", "to_address": "bogus_addr", "amount": 1.0}
    client.post("/api/wallet/submit", json=body)
    r = client.get("/api/canon-log?since_id=0&limit=200")
    assert r.status_code == 200


def test_wallet_submit_invalid(client):
    body = {"op": "transfer", "address": "u1", "to_address": "u1", "amount": 1.0}
    r = client.post("/api/wallet/submit", json=body)
    assert r.status_code == 200
    j = r.json()
    assert j["accepted"] is False


# --- Этап 4: KPI / scenarios / export ---


def test_kpi_endpoint(client):
    r = client.get("/api/analytics/kpi")
    assert r.status_code == 200
    j = r.json()
    for key in ("supply", "gini", "burn", "role_counts", "height"):
        assert key in j


def test_gini_endpoint_wrt(client):
    r = client.get("/api/analytics/gini?asset=wrt")
    assert r.status_code == 200
    j = r.json()
    assert j["asset"] == "wrt"
    assert 0.0 <= j["gini"] <= 1.0


def test_scenarios_list(client):
    r = client.get("/api/scenarios")
    assert r.status_code == 200
    j = r.json()
    assert "scenarios" in j
    assert "epoch_wipe_stress.yaml" in j["scenarios"]


def test_scenario_inline_yaml(client):
    inline = """
    name: test_inline
    steps:
      - create_account: { address: alice }
      - mint: { receiver: alice, amount: 10, asset: wrt }
    asserts:
      - balance: { address: alice, asset: wrt, op: ">=", value: 0 }
    """
    r = client.post("/api/scenarios/run", json={"yaml": inline, "reset_state": True})
    assert r.status_code == 200
    j = r.json()
    assert "report" in j
    assert j["report"]["name"] == "test_inline"


def test_export_blocks_jsonl(client):
    r = client.get("/api/export/blocks.jsonl")
    assert r.status_code == 200
    assert r.headers["content-type"].startswith("application/x-ndjson")
    assert r.text.strip()


def test_export_balances_csv(client):
    r = client.get("/api/export/balances.csv")
    assert r.status_code == 200
    assert r.headers["content-type"].startswith("text/csv")
    assert "address,role" in r.text


def test_export_ticks_csv(client):
    r = client.get("/api/export/ticks.csv")
    assert r.status_code == 200
    assert "time,price,ts" in r.text


def test_export_canon_log(client):
    r = client.get("/api/export/canon_log.jsonl?limit=5")
    assert r.status_code == 200
    assert r.headers["content-type"].startswith("application/x-ndjson")


def test_metrics_endpoint(client):
    r = client.get("/metrics")
    assert r.status_code == 200
    # Если prometheus_client установлен — отдаёт собственный content-type;
    # если отключён через env — текст с пометкой.
    body = r.text
    assert "volnix_sim" in body or "disabled" in body or "not installed" in body
