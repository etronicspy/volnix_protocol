"""Admission в мемпул через validate_and_build_tx / validate_treasury_mint."""
from __future__ import annotations

import pytest

from core.models import OrderType, Role, TransactionType
from core.state import SIM_TREASURY_ADDR
from core.wallet_validate import validate_and_build_tx, validate_treasury_mint


@pytest.fixture
def addr1(mk_account):
    return mk_account("user_alpha", role=Role.CITIZEN, wrt=100.0, lzn=50.0).address


@pytest.fixture
def addr2(mk_account):
    return mk_account("user_beta", role=Role.CITIZEN, wrt=10.0).address


def test_transfer_accepts_positive_amount(state_manager, addr1, addr2):
    ok, msg, tx = validate_and_build_tx(
        state_manager, "transfer", addr1, to_address=addr2, amount=1.5, asset="wrt"
    )
    assert ok and tx is not None
    assert tx.tx_type == TransactionType.TRANSFER
    assert tx.asset_type == "wrt"
    assert tx.amount == 1.5
    assert "next block" in msg


def test_transfer_rejects_self(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(
        state_manager, "transfer", addr1, to_address=addr1, amount=1.0
    )
    assert not ok and "self" in msg


def test_transfer_rejects_zero_amount(state_manager, addr1, addr2):
    ok, msg, _ = validate_and_build_tx(
        state_manager, "transfer", addr1, to_address=addr2, amount=0.0
    )
    assert not ok and "positive" in msg


def test_treasury_address_cannot_submit(state_manager, addr2):
    ok, msg, _ = validate_and_build_tx(
        state_manager, "transfer", SIM_TREASURY_ADDR, to_address=addr2, amount=1.0
    )
    assert not ok and "treasury" in msg.lower()


def test_create_order_limit_buy_ok(state_manager, mk_account):
    """admission всегда пропускает; engine применит каноничные проверки роли в блоке."""
    v = mk_account("v_buyer", role=Role.VALIDATOR, wrt=1000.0, frozen=10.0, zkp=True)
    ok, _, tx = validate_and_build_tx(
        state_manager,
        "create_order",
        v.address,
        side="buy",
        price=5.0,
        amount=2.0,
    )
    assert ok and tx is not None
    assert tx.order_type == OrderType.BUY
    assert tx.price == 5.0
    assert tx.amount == 2.0
    assert tx.market is False


def test_create_order_rejects_bad_side(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(
        state_manager,
        "create_order",
        addr1,
        side="hodl",
        price=1.0,
        amount=1.0,
    )
    assert not ok and "side" in msg


def test_create_order_market_requires_amount(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(
        state_manager,
        "create_order",
        addr1,
        side="buy",
        market=True,
        amount=0,
    )
    assert not ok and "positive" in msg


def test_create_order_limit_requires_price_and_amount(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(
        state_manager,
        "create_order",
        addr1,
        side="buy",
        price=0,
        amount=1.0,
    )
    assert not ok and "positive" in msg


def test_cancel_order_requires_order_id(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(
        state_manager, "cancel_order", addr1
    )
    assert not ok and "order_id" in msg


def test_declare_requires_positive_total(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(
        state_manager, "declare", addr1, burn_b=0, stake_s=0
    )
    assert not ok and "positive" in msg


def test_declare_builds_burn_and_stake(state_manager, mk_account):
    v = mk_account("v_decl", role=Role.VALIDATOR, frozen=10.0, ant=20.0, zkp=True)
    ok, _, tx = validate_and_build_tx(
        state_manager, "declare", v.address, burn_b=1.0, stake_s=2.0
    )
    assert ok and tx is not None
    assert tx.amount == 1.0
    assert tx.stake_amount == 2.0
    assert tx.tx_type == TransactionType.DECLARE_PARTICIPATION


def test_activate_lzn_admission_accepts(state_manager, mk_account):
    """admission всегда принимает; engine применит роль-валидатор §4.2."""
    a = mk_account("u_act", role=Role.CITIZEN, lzn=10.0)
    ok, _, tx = validate_and_build_tx(
        state_manager, "activate_lzn", a.address, amount=1.0
    )
    assert ok and tx is not None
    assert tx.tx_type == TransactionType.ACTIVATE_LZN


def test_set_role_requires_role(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(state_manager, "set_role", addr1)
    assert not ok and "role is required" in msg


def test_set_role_rejects_same_role(state_manager, mk_account):
    a = mk_account("u_role", role=Role.CITIZEN)
    ok, msg, _ = validate_and_build_tx(
        state_manager, "set_role", a.address, role=Role.CITIZEN
    )
    assert not ok and "Already" in msg


def test_set_role_builds_tx(state_manager, mk_account):
    a = mk_account("u_role2", role=Role.CITIZEN, zkp=True)
    ok, _, tx = validate_and_build_tx(
        state_manager, "set_role", a.address, role=Role.PROVIDER
    )
    assert ok and tx is not None
    assert tx.role == Role.PROVIDER


def test_verify_zkp_builds_tx(state_manager, addr1):
    ok, _, tx = validate_and_build_tx(state_manager, "verify_zkp", addr1)
    assert ok and tx is not None
    assert tx.tx_type == TransactionType.ZKP_VERIFY


def test_unknown_op(state_manager, addr1):
    ok, msg, _ = validate_and_build_tx(state_manager, "do_thing", addr1)
    assert not ok and "Unknown op" in msg


# === validate_treasury_mint ===

def test_treasury_mint_wrt_ok(state_manager, mk_account):
    a = mk_account("u_mint", role=Role.CITIZEN)
    ok, msg, tx = validate_treasury_mint(state_manager, a.address, 50.0, "wrt")
    assert ok and tx is not None
    assert tx.sender == SIM_TREASURY_ADDR
    assert tx.amount == 50.0
    assert tx.asset_type == "wrt"


def test_treasury_mint_ant_rejected_for_citizen(state_manager, mk_account):
    """§4.1–4.2: ANT не доступен у Гражданина."""
    a = mk_account("u_mint2", role=Role.CITIZEN)
    ok, msg, _ = validate_treasury_mint(state_manager, a.address, 1.0, "ant")
    assert not ok and "Гражданина" in msg


def test_treasury_mint_ant_ok_for_validator(state_manager, mk_account):
    v = mk_account("v_mint", role=Role.VALIDATOR, frozen=10.0, zkp=True)
    ok, _, tx = validate_treasury_mint(state_manager, v.address, 1.0, "ant")
    assert ok and tx is not None
    assert tx.asset_type == "ant"


def test_treasury_mint_unknown_asset(state_manager, mk_account):
    a = mk_account("u_mint3", role=Role.CITIZEN)
    ok, msg, _ = validate_treasury_mint(state_manager, a.address, 1.0, "btc")
    assert not ok and "asset must be" in msg


def test_treasury_mint_insufficient_reserves(state_manager, mk_account):
    a = mk_account("u_mint4", role=Role.CITIZEN)
    tr = state_manager.accounts[SIM_TREASURY_ADDR]
    tr.wrt_balance = 1.0
    ok, msg, _ = validate_treasury_mint(state_manager, a.address, 100.0, "wrt")
    assert not ok and "insufficient" in msg
