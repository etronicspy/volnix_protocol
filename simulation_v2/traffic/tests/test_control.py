"""Control API smoke tests."""

from __future__ import annotations

import sys
from pathlib import Path
from unittest.mock import AsyncMock

from fastapi.testclient import TestClient

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from volnix_traffic.control import create_control_app
from volnix_traffic.registry import BotWallet
from volnix_traffic.runtime import TrafficRuntime
from volnix_traffic.settings import TrafficSettings


def test_control_status_start_stop(tmp_path: Path):
    settings = TrafficSettings(
        autostart=False,
        node_url="http://127.0.0.1:9",
        poll_interval_sec=0.05,
    )
    runtime = TrafficRuntime(settings, state_path=tmp_path / "wallets.json")
    runtime.client.chain_summary = AsyncMock(return_value={"height": 0})  # type: ignore
    runtime.tick_once = AsyncMock(return_value={"height": 0})  # type: ignore

    app = create_control_app(runtime)
    with TestClient(app) as client:
        st = client.get("/status")
        assert st.status_code == 200
        assert st.json()["running"] is False

        r = client.post("/start", json={"intensity": 3.5})
        assert r.status_code == 200
        body = r.json()
        assert body["running"] is True
        assert body["intensity"] == 3.5

        w = client.get("/wallets")
        assert w.status_code == 200
        assert w.json()["count"] == 0

        runtime.registry.add(BotWallet(seed="bot-0", address="volnix1x", role="citizen"))
        w2 = client.get("/wallets")
        assert w2.json()["count"] == 1

        client.post("/intensity", json={"intensity": 1.0})
        assert client.get("/status").json()["intensity"] == 1.0

        client.post("/stop")
        assert client.get("/status").json()["running"] is False
