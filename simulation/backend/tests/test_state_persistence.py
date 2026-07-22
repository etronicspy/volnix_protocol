"""save_state / load_state / reset_state — round-trip и миграции."""
from __future__ import annotations

import json
import os

from core.models import Role
from core.state import (
    GENESIS_VALIDATOR_ADDR,
    SIM_TREASURY_ADDR,
    SIM_TREASURY_ADDR_LEGACY,
    StateManager,
)


def test_save_and_load_roundtrip(state_manager, tmp_path):
    fp = str(tmp_path / "saved.json")
    state_manager.last_price = 12.5
    state_manager.price_history.append({"time": "00:00:01", "price": 12.5, "ts": 1.0})
    state_manager.save_state(fp)
    assert os.path.exists(fp)

    other = StateManager(data_dir=str(tmp_path))
    other.load_state(fp)
    assert other.last_price == 12.5
    assert GENESIS_VALIDATOR_ADDR in other.accounts
    assert other.accounts[GENESIS_VALIDATOR_ADDR].role == Role.VALIDATOR
    assert any(row.get("price") == 12.5 for row in other.price_history)


def test_load_state_corrupted_falls_back_to_genesis(tmp_path):
    fp = str(tmp_path / "broken.json")
    with open(fp, "w") as f:
        f.write("{ not json ")
    sm = StateManager(data_dir=str(tmp_path))
    sm.load_state(fp)
    # Файл повреждён, без .bak — должны получить пустое состояние
    assert sm.accounts == {}
    assert sm.current_height == 0


def test_load_state_recovers_from_bak(tmp_path):
    sm = StateManager(data_dir=str(tmp_path))
    sm.init_genesis()
    sm.last_price = 99.0
    fp = str(tmp_path / "state.json")
    sm.save_state(fp)
    # копия → .bak, основной файл рвём
    import shutil

    shutil.copy2(fp, fp + ".bak")
    with open(fp, "w") as f:
        f.write("{ broken")

    other = StateManager(data_dir=str(tmp_path))
    other.load_state(fp)
    assert other.last_price == 99.0
    assert GENESIS_VALIDATOR_ADDR in other.accounts


def test_legacy_treasury_addr_migrated_on_load(tmp_path):
    """Старый ключ казны godmode → миграция в SIM_TREASURY_ADDR."""
    fp = str(tmp_path / "old.json")
    payload = {
        "current_height": 0,
        "accounts": {
            SIM_TREASURY_ADDR_LEGACY: {
                "address": SIM_TREASURY_ADDR_LEGACY,
                "wrt_balance": 100.0,
                "lzn_balance": 50.0,
                "ant_balance": 7.0,
                "role": "citizen",
                "lzn_frozen_mining": 0.0,
                "zkp_verified": False,
            },
        },
        "orders": {},
        "blocks": [],
        "mempool": [],
    }
    with open(fp, "w") as f:
        json.dump(payload, f)

    sm = StateManager(data_dir=str(tmp_path))
    sm.load_state(fp)
    assert SIM_TREASURY_ADDR in sm.accounts
    assert SIM_TREASURY_ADDR_LEGACY not in sm.accounts
    assert sm.accounts[SIM_TREASURY_ADDR].wrt_balance == 100.0


def test_old_guest_role_migrated_to_citizen(tmp_path):
    fp = str(tmp_path / "old_guest.json")
    payload = {
        "current_height": 0,
        "accounts": {
            "x1": {
                "address": "x1",
                "wrt_balance": 0.0,
                "lzn_balance": 0.0,
                "ant_balance": 0.0,
                "role": "guest",
                "lzn_frozen_mining": 0.0,
                "zkp_verified": False,
            },
        },
        "orders": {},
        "blocks": [],
        "mempool": [],
    }
    with open(fp, "w") as f:
        json.dump(payload, f)

    sm = StateManager(data_dir=str(tmp_path))
    sm.load_state(fp)
    assert sm.accounts["x1"].role == Role.CITIZEN


def test_reset_state_wipes_and_reinit_genesis(state_manager, tmp_path):
    state_manager.last_price = 50.0
    state_manager.accounts["temp"] = state_manager.create_account("temp")
    fp = str(tmp_path / "state.json")
    state_manager.save_state(fp)

    state_manager.reset_state(fp)
    assert state_manager.current_height == 0
    assert state_manager.last_price == 0.0
    assert "temp" not in state_manager.accounts
    assert GENESIS_VALIDATOR_ADDR in state_manager.accounts


def test_sim_block_interval_clamped_on_load(tmp_path):
    fp = str(tmp_path / "bad_iv.json")
    payload = {
        "current_height": 0,
        "accounts": {},
        "orders": {},
        "blocks": [],
        "mempool": [],
        "sim_block_interval_sec": 9999.0,
    }
    with open(fp, "w") as f:
        json.dump(payload, f)
    sm = StateManager(data_dir=str(tmp_path))
    sm.load_state(fp)
    assert sm.sim_block_interval_sec == 300.0
    # Legacy state без sim_speed — скорость выводится из интервала блока
    assert sm.sim_speed == 60.0 / 300.0


def test_sim_speed_roundtrip_and_clamp(state_manager, tmp_path):
    fp = str(tmp_path / "speed.json")
    state_manager.sim_speed = 604_800.0  # 1 с = 1 неделя
    state_manager.save_state(fp)

    other = StateManager(data_dir=str(tmp_path))
    other.load_state(fp)
    assert other.sim_speed == 604_800.0

    # Значение вне диапазона клиппится при загрузке
    with open(fp) as f:
        payload = json.load(f)
    payload["sim_speed"] = 10.0 ** 9
    with open(fp, "w") as f:
        json.dump(payload, f)
    clamped = StateManager(data_dir=str(tmp_path))
    clamped.load_state(fp)
    assert clamped.sim_speed == 604_800.0
