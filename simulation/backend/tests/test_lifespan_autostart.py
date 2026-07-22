"""Lifespan auto-start: bot, AutoDeclareDaemon, NetworkSim — высота растёт сама.

Smoke-тесты на FastAPI lifespan:
- если `VOLNIX_SIM_NUM_NODES>=2`, NetworkSim инициализируется и `/api/network/nodes` отдаёт enabled=True;
- если `VOLNIX_SIM_BOT_AUTOSTART=true`, бот запускается;
- если `VOLNIX_SIM_AUTO_DECLARE=true`, AutoDeclareDaemon подаёт declare и блоки рождаются без явных API-вызовов.
"""
from __future__ import annotations

import asyncio
import importlib
import json
import os

import pytest
from fastapi.testclient import TestClient


def _reload_main_with_env(monkeypatch, tmp_path, **env):
    """Перезагрузить main.py с заданным окружением и tmp data_dir."""
    monkeypatch.setenv("VOLNIX_SIM_DATA_DIR", str(tmp_path))
    for k, v in env.items():
        monkeypatch.setenv(k, str(v))

    # Подготовим state.json с коротким block_time, чтобы тест был быстрым.
    state_path = tmp_path / "state.json"
    if not state_path.exists():
        state_path.write_text(json.dumps({"sim_block_interval_sec": 0.2}))

    from core import settings as settings_module

    settings_module.reload_settings()

    import main as main_module

    importlib.reload(main_module)
    return main_module


def test_lifespan_enables_network_when_num_nodes_ge_2(monkeypatch, tmp_path):
    main_module = _reload_main_with_env(
        monkeypatch,
        tmp_path,
        VOLNIX_SIM_NUM_NODES=2,
        VOLNIX_SIM_AUTO_DECLARE="false",
        VOLNIX_SIM_BOT_AUTOSTART="false",
    )
    with TestClient(main_module.app) as client:
        r = client.get("/api/network/nodes")
        assert r.status_code == 200
        body = r.json()
        assert body["enabled"] is True
        assert len(body["nodes"]) == 2


def test_lifespan_disabled_network_when_num_nodes_1(monkeypatch, tmp_path):
    main_module = _reload_main_with_env(
        monkeypatch,
        tmp_path,
        VOLNIX_SIM_NUM_NODES=1,
        VOLNIX_SIM_AUTO_DECLARE="false",
        VOLNIX_SIM_BOT_AUTOSTART="false",
    )
    with TestClient(main_module.app) as client:
        body = client.get("/api/network/nodes").json()
        assert body["enabled"] is False


def test_lifespan_bot_autostart(monkeypatch, tmp_path):
    main_module = _reload_main_with_env(
        monkeypatch,
        tmp_path,
        VOLNIX_SIM_NUM_NODES=1,
        VOLNIX_SIM_BOT_AUTOSTART="true",
        VOLNIX_SIM_AUTO_DECLARE="false",
    )
    with TestClient(main_module.app) as client:
        body = client.get("/api/bot/status").json()
        assert body["is_running"] is True


def test_lifespan_auto_declare_grows_height(monkeypatch, tmp_path):
    """С AutoDeclareDaemon + коротким block_time за ~2*block_time высота > 0 без API-вызовов."""
    main_module = _reload_main_with_env(
        monkeypatch,
        tmp_path,
        VOLNIX_SIM_NUM_NODES=1,
        VOLNIX_SIM_AUTO_DECLARE="true",
        VOLNIX_SIM_BOT_AUTOSTART="false",
    )
    import time

    with TestClient(main_module.app) as client:
        # block_time = 0.2 → за ~2 секунды должно вырасти на несколько блоков.
        deadline = time.time() + 5.0
        height = 0
        while time.time() < deadline:
            j = client.get("/").json()
            height = int(j.get("block_height", 0))
            if height > 0:
                break
            time.sleep(0.2)
        assert height > 0, "height did not grow within 5s (block_time=0.2s)"


def test_network_config_endpoint(monkeypatch, tmp_path):
    main_module = _reload_main_with_env(
        monkeypatch,
        tmp_path,
        VOLNIX_SIM_NUM_NODES=3,
        VOLNIX_SIM_AUTO_DECLARE="false",
        VOLNIX_SIM_BOT_AUTOSTART="false",
        VOLNIX_SIM_GOSSIP_LATENCY_MS=50,
    )
    with TestClient(main_module.app) as client:
        topo = client.get("/api/network/topology").json()
        assert topo["enabled"] is True
        assert topo["config"]["num_nodes"] == 3
        assert topo["config"]["gossip_latency_ms"] == 50

        r = client.post("/api/network/config", json={"gossip_loss_pct": 12.5})
        assert r.status_code == 200
        body = r.json()
        assert body["ok"] is True
        assert body["config"]["gossip_loss_pct"] == 12.5
