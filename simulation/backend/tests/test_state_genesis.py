"""§6.3 / sim bootstrap: genesis = seed only; когорта 5+5 на высоте 2."""
from __future__ import annotations

import pytest

from core.models import Role
from core.state import (
    ANT_GENESIS_PER_PROVIDER,
    ANT_GENESIS_TOTAL,
    BLOCKS_PER_EPOCH,
    BURN_CAP_LAMBDA,
    GENESIS_BOOTSTRAP_PROVIDERS,
    GENESIS_BOOTSTRAP_VALIDATOR_LZN,
    GENESIS_BOOTSTRAP_VALIDATORS,
    GENESIS_PROVIDER_ADDR,
    GENESIS_VALIDATOR_ADDR,
    GENESIS_VALIDATOR_ANT_BALANCE,
    L_TOTAL_GENESIS,
    LZN_BOOTSTRAP_MARKET_EXTRA,
    LZN_GENESIS_ACTIVATED,
    LZN_MAX_FROZEN_PER_ADDRESS,
    LZN_TOTAL_SUPPLY_REF,
    SIM_BOOTSTRAP_INJECT_HEIGHT,
    SIM_TREASURY_ADDR,
    consensus_validator_set_from_participation,
    default_consensus_validator_set,
    eligible_for_provider_role,
    eligible_for_validator_role,
)


def test_genesis_only_seed_and_treasury(state_manager):
    assert set(state_manager.accounts) == {GENESIS_VALIDATOR_ADDR, SIM_TREASURY_ADDR}
    assert state_manager.sim_bootstrap_injected is False
    for addr in GENESIS_BOOTSTRAP_VALIDATORS:
        assert addr not in state_manager.accounts
    for addr in GENESIS_BOOTSTRAP_PROVIDERS:
        assert addr not in state_manager.accounts
    assert "volnix1gprov0provider00genesis0" not in state_manager.accounts
    assert GENESIS_PROVIDER_ADDR == GENESIS_BOOTSTRAP_PROVIDERS[0]


def test_genesis_seed_balances(state_manager):
    gv = state_manager.accounts[GENESIS_VALIDATOR_ADDR]
    assert gv.role == Role.VALIDATOR
    assert gv.lzn_balance == 0.0
    assert gv.lzn_frozen_mining == float(LZN_GENESIS_ACTIVATED) == 1.0
    assert gv.ant_balance == float(GENESIS_VALIDATOR_ANT_BALANCE) == 1.0
    assert gv.zkp_verified is True


def test_genesis_block_has_no_cohort_txs(state_manager):
    assert len(state_manager.blocks) == 1
    g = state_manager.blocks[0]
    assert g["height"] == 0
    tx_types = {t["tx_type"] for t in g["transactions"]}
    assert "genesis_validator_lzn" in tx_types
    assert "zkp_verify" not in tx_types
    assert "genesis_provider_ant" not in tx_types
    assert "set_role" not in tx_types


def test_sim_bootstrap_inject_height_constant():
    assert SIM_BOOTSTRAP_INJECT_HEIGHT == 2


def test_inject_sim_bootstrap_cohort(state_manager):
    txs: list = []
    assert state_manager.inject_sim_bootstrap_cohort(txs) is True
    assert state_manager.sim_bootstrap_injected is True
    assert state_manager.inject_sim_bootstrap_cohort(txs) is False  # idempotent

    for addr in GENESIS_BOOTSTRAP_VALIDATORS:
        acc = state_manager.accounts[addr]
        assert acc.role == Role.VALIDATOR
        assert acc.zkp_verified is True
        assert acc.lzn_frozen_mining == float(GENESIS_BOOTSTRAP_VALIDATOR_LZN)
        assert acc.lzn_frozen_mining <= LZN_MAX_FROZEN_PER_ADDRESS
        assert acc.ant_balance == pytest.approx(
            BURN_CAP_LAMBDA * float(GENESIS_BOOTSTRAP_VALIDATOR_LZN)
        )

    expected_total = float(BLOCKS_PER_EPOCH) * BURN_CAP_LAMBDA * L_TOTAL_GENESIS
    assert ANT_GENESIS_TOTAL == expected_total
    for addr in GENESIS_BOOTSTRAP_PROVIDERS:
        acc = state_manager.accounts[addr]
        assert acc.role == Role.PROVIDER
        assert acc.zkp_verified is True
        assert acc.ant_balance == float(ANT_GENESIS_PER_PROVIDER)
        # После sim-fill остаётся рыночный остаток LZN.
        assert acc.lzn_balance == pytest.approx(LZN_BOOTSTRAP_MARKET_EXTRA)

    types = [t["tx_type"] for t in txs]
    assert "sim_bootstrap_message" in types
    assert types.count("zkp_verify") == 10
    assert types.count("set_role") == 10
    assert types.count("genesis_provider_ant") == 5
    assert types.count("epoch_lzn_credit") == 5
    assert types.count("sim_bootstrap_lzn_fill") == 5
    assert types.count("sim_bootstrap_ant_fill") == 5
    assert "genesis_validator_lzn" not in types


@pytest.mark.asyncio
async def test_engine_injects_cohort_on_height_2(engine):
    """BeginBlock высоты 2 вводит когорту; до этого её нет."""
    from tests.conftest import seed_declare_tx

    assert all(a not in engine.state.accounts for a in GENESIS_BOOTSTRAP_VALIDATORS)
    engine.state.accounts[GENESIS_VALIDATOR_ADDR].ant_balance = 10.0

    engine.state.mempool.append(seed_declare_tx(engine))
    await engine.produce_block()
    assert engine.state.current_height == 1
    assert engine.state.sim_bootstrap_injected is False
    assert all(a not in engine.state.accounts for a in GENESIS_BOOTSTRAP_PROVIDERS)

    engine.state.mempool.append(seed_declare_tx(engine))
    await engine.produce_block()
    assert engine.state.current_height == 2
    assert engine.state.sim_bootstrap_injected is True
    for addr in GENESIS_BOOTSTRAP_VALIDATORS:
        assert addr in engine.state.accounts
    for addr in GENESIS_BOOTSTRAP_PROVIDERS:
        assert addr in engine.state.accounts
    last = engine.state.blocks[-1]
    assert any(t.get("tx_type") == "sim_bootstrap_message" for t in last["transactions"])


def test_genesis_blocks_per_epoch_constant():
    assert BLOCKS_PER_EPOCH == 7 * 24 * 60 == 10080


def test_genesis_lzn_cap_per_address():
    assert LZN_MAX_FROZEN_PER_ADDRESS == LZN_TOTAL_SUPPLY_REF // 3


def test_default_validator_set_after_genesis(state_manager):
    vs = default_consensus_validator_set(state_manager.accounts)
    assert len(vs) == 1
    assert vs[0]["address"] == GENESIS_VALIDATOR_ADDR
    assert vs[0]["power"] == 1.0


def test_consensus_set_from_participation_picks_max_power():
    participation = {
        "v1": {"s": 10.0, "w_i": 0.5, "L_i": 100.0},
        "v2": {"s": 5.0, "w_i": 0.9, "L_i": 100.0},
    }
    vs = consensus_validator_set_from_participation(participation)
    by_addr = {v["address"]: v["power"] for v in vs}
    assert by_addr["v1"] == max(10.0, 0.5 * 100.0)
    assert by_addr["v2"] == max(5.0, 0.9 * 100.0)


def test_eligible_validator_genesis_bypasses_zkp(state_manager):
    gv = state_manager.accounts[GENESIS_VALIDATOR_ADDR]
    gv.zkp_verified = False
    assert eligible_for_validator_role(GENESIS_VALIDATOR_ADDR, gv) is True


def test_eligible_validator_requires_zkp_and_lzn(mk_account):
    a = mk_account("notgen1", zkp=False, lzn=5.0)
    assert eligible_for_validator_role("notgen1", a) is False
    a.zkp_verified = True
    a.lzn_balance = 0.0
    a.lzn_frozen_mining = 0.0
    assert eligible_for_validator_role("notgen1", a) is False
    a.lzn_balance = 1.0
    assert eligible_for_validator_role("notgen1", a) is True


def test_eligible_provider_requires_zkp_only(mk_account):
    a = mk_account("notgen2", zkp=False)
    assert eligible_for_provider_role("notgen2", a) is False
    a.zkp_verified = True
    assert eligible_for_provider_role("notgen2", a) is True


def test_get_full_state_shape(state_manager):
    s = state_manager.get_full_state()
    for k in (
        "height",
        "accounts",
        "market",
        "blocks",
        "consensus_validators",
        "next_proposer",
        "blocks_per_epoch",
        "sim_block_interval_sec",
        "genesis_bootstrap_validators",
        "genesis_bootstrap_providers",
        "sim_bootstrap_inject_height",
        "sim_bootstrap_injected",
    ):
        assert k in s, k
    assert s["blocks_per_epoch"] == BLOCKS_PER_EPOCH
    assert s["sim_bootstrap_inject_height"] == 2
    assert s["sim_bootstrap_injected"] is False
