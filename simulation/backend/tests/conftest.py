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


def strip_bootstrap_economy(state_manager, *, keep_seed_lzn: float = 0.0) -> None:
    """Убрать bootstrap-поставщиков и обнулить LZN bootstrap-валидаторов.

    Нужно тестам эпохи/cap, которым требуется контролируемый L_total и
    ровно N поставщиков, без genesis-пятёрки.
    """
    from core.state import (
        GENESIS_BOOTSTRAP_PROVIDERS,
        GENESIS_BOOTSTRAP_VALIDATORS,
        GENESIS_VALIDATOR_ADDR,
    )

    for addr in GENESIS_BOOTSTRAP_PROVIDERS:
        state_manager.accounts.pop(addr, None)
    for addr in GENESIS_BOOTSTRAP_VALIDATORS:
        acc = state_manager.accounts.get(addr)
        if acc is not None:
            acc.lzn_frozen_mining = 0.0
            acc.lzn_balance = 0.0
    gv = state_manager.accounts.get(GENESIS_VALIDATOR_ADDR)
    if gv is not None:
        gv.lzn_frozen_mining = float(keep_seed_lzn)
        gv.lzn_balance = 0.0


def seed_declare_tx(engine):
    """Canon-корректный declare seed-валидатора (b = λ·L_i, без undershoot)."""
    import time
    import uuid

    from core.engine import BURN_CAP_LAMBDA
    from core.models import Transaction, TransactionType
    from core.state import GENESIS_VALIDATOR_ADDR

    gv = engine.state.accounts[GENESIS_VALIDATOR_ADDR]
    if gv.lzn_frozen_mining <= 0:
        gv.lzn_frozen_mining = 1.0
    L_i = float(gv.lzn_frozen_mining)
    need = BURN_CAP_LAMBDA * L_i
    if float(gv.ant_balance) + 1e-12 < need:
        gv.ant_balance = need
    b = min(float(gv.ant_balance), need, L_i)
    return Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.DECLARE_PARTICIPATION,
        sender=GENESIS_VALIDATOR_ADDR,
        amount=float(b),
        stake_amount=0.0,
        asset_type="ant",
        timestamp=time.time(),
    )


def enqueue_corridor_declares(engine) -> int:
    """Declare b_i=λ·L_i для каждого валидатора с L_i>0 (набор низа коридора)."""
    import time
    import uuid

    from core.engine import BURN_CAP_LAMBDA
    from core.models import Role, Transaction, TransactionType

    n = 0
    for addr, acc in sorted(engine.state.accounts.items()):
        if acc.role != Role.VALIDATOR:
            continue
        L_i = float(acc.lzn_frozen_mining)
        if L_i <= 0:
            continue
        need = BURN_CAP_LAMBDA * L_i
        if float(acc.ant_balance) + 1e-12 < need:
            acc.ant_balance = need
        b = min(float(acc.ant_balance), need, L_i)
        if b <= 1e-12:
            continue
        engine.state.mempool.append(
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.DECLARE_PARTICIPATION,
                sender=addr,
                amount=float(b),
                stake_amount=0.0,
                asset_type="ant",
                timestamp=time.time(),
            )
        )
        n += 1
    return n
