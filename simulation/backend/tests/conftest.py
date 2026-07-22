"""Fixtures для pytest: изолированный StateManager + Engine в tmp data_dir."""
from __future__ import annotations

import os
import sys
from pathlib import Path

import pytest

ROOT = Path(__file__).resolve().parents[1]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))


@pytest.fixture(autouse=True)
def _isolated_data_dir(tmp_path, monkeypatch):
    """Каждый тест получает свой data_dir; сбрасываем кеш settings."""
    monkeypatch.setenv("VOLNIX_SIM_DATA_DIR", str(tmp_path))
    monkeypatch.setenv("VOLNIX_SIM_CANON_LOG_CAPACITY", "200")

    from core import settings as settings_module

    settings_module.reload_settings()
    yield
    settings_module.reload_settings()


@pytest.fixture
def state_manager(tmp_path):
    """Чистый StateManager с инициализированным genesis."""
    from core.state import StateManager

    sm = StateManager(data_dir=str(tmp_path))
    sm.init_genesis()
    return sm


@pytest.fixture
def engine(state_manager):
    """SimulationEngine привязан к state_manager без запущенного loop."""
    from core.engine import SimulationEngine

    return SimulationEngine(state_manager)


def make_account(state_manager, address, role=None, *, wrt=0.0, lzn=0.0, frozen=0.0, ant=0.0, zkp=False):
    """Утилита: создать аккаунт с балансами/ролью для тестов."""
    from core.models import Role

    acc = state_manager.create_account(address)
    if role is not None:
        acc.role = role if isinstance(role, Role) else Role(role)
    acc.wrt_balance = float(wrt)
    acc.lzn_balance = float(lzn)
    acc.lzn_frozen_mining = float(frozen)
    acc.ant_balance = float(ant)
    acc.zkp_verified = bool(zkp)
    return acc


@pytest.fixture
def mk_account(state_manager):
    """Хелпер make_account, связанный с фикстурным state_manager."""

    def _factory(address, **kwargs):
        return make_account(state_manager, address, **kwargs)

    return _factory
