"""§6.3 / §4.2 / §5.5: genesis-инварианты StateManager."""
from __future__ import annotations

from core.models import Role
from core.state import (
    BLOCKS_PER_EPOCH,
    GENESIS_PROVIDER_ADDR,
    GENESIS_VALIDATOR_ADDR,
    GENESIS_VALIDATOR_ANT_BALANCE,
    LZN_GENESIS_ACTIVATED,
    LZN_MAX_FROZEN_PER_ADDRESS,
    LZN_TOTAL_SUPPLY_REF,
    SIM_TREASURY_ADDR,
    consensus_validator_set_from_participation,
    default_consensus_validator_set,
    eligible_for_provider_role,
    eligible_for_validator_role,
)


def test_genesis_creates_three_accounts(state_manager):
    assert GENESIS_VALIDATOR_ADDR in state_manager.accounts
    assert GENESIS_PROVIDER_ADDR in state_manager.accounts
    assert SIM_TREASURY_ADDR in state_manager.accounts


def test_genesis_validator_balances(state_manager):
    """§6.3(3): полная одноразовая эмиссия LZN; §4.2 cap — frozen ≤ ⌊L/3⌋."""
    gv = state_manager.accounts[GENESIS_VALIDATOR_ADDR]
    assert gv.role == Role.VALIDATOR
    assert gv.lzn_balance == float(LZN_TOTAL_SUPPLY_REF - LZN_GENESIS_ACTIVATED)
    assert gv.lzn_frozen_mining == float(LZN_GENESIS_ACTIVATED)
    # genesis-исключение: 6667 > cap 3333 в §4.2, но в genesis допустимо
    assert gv.lzn_frozen_mining > LZN_MAX_FROZEN_PER_ADDRESS
    assert gv.ant_balance == float(GENESIS_VALIDATOR_ANT_BALANCE)
    assert gv.zkp_verified is True


def test_genesis_provider_ant_eq_epoch_blocks_times_l_total(state_manager):
    """§5.5 §6.3(4): ANT_genesis = EpochBlocks × L_total_genesis."""
    gp = state_manager.accounts[GENESIS_PROVIDER_ADDR]
    assert gp.role == Role.PROVIDER
    expected = float(BLOCKS_PER_EPOCH * LZN_GENESIS_ACTIVATED)
    assert gp.ant_balance == expected
    assert gp.zkp_verified is True


def test_genesis_blocks_per_epoch_constant():
    """§5.5: эталон 1 блок/мин × 7 дней."""
    assert BLOCKS_PER_EPOCH == 7 * 24 * 60 == 10080


def test_genesis_lzn_cap_per_address():
    """§4.2: cap = ⌊total/3⌋."""
    assert LZN_MAX_FROZEN_PER_ADDRESS == LZN_TOTAL_SUPPLY_REF // 3


def test_genesis_block_in_chain(state_manager):
    """Genesis-блок занимает высоту 0 и содержит протокольные tx §6.3."""
    assert len(state_manager.blocks) == 1
    g = state_manager.blocks[0]
    assert g["height"] == 0
    tx_types = {t["tx_type"] for t in g["transactions"]}
    assert "genesis_validator_lzn" in tx_types
    assert "genesis_lzn_activate" in tx_types
    assert "genesis_provider_ant" in tx_types


def test_default_validator_set_after_genesis(state_manager):
    """§6.3(5): ValidatorSet = единственный genesis-валидатор с power ≈ L_i."""
    vs = default_consensus_validator_set(state_manager.accounts)
    assert len(vs) == 1
    assert vs[0]["address"] == GENESIS_VALIDATOR_ADDR
    assert vs[0]["power"] > 0


def test_consensus_set_from_participation_picks_max_power():
    """power = max(s, w·L) для каждого declare-участника."""
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
    """§3.1: не-genesis валидатор требует ZKP И >0 LZN."""
    a = mk_account("notgen1", zkp=False, lzn=5.0)
    assert eligible_for_validator_role("notgen1", a) is False
    a.zkp_verified = True
    a.lzn_balance = 0.0
    a.lzn_frozen_mining = 0.0
    assert eligible_for_validator_role("notgen1", a) is False
    a.lzn_balance = 1.0
    assert eligible_for_validator_role("notgen1", a) is True


def test_eligible_provider_requires_zkp_only(mk_account):
    """§4.2: не-genesis поставщик требует ZKP, LZN не требуется."""
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
    ):
        assert k in s, k
    assert s["blocks_per_epoch"] == BLOCKS_PER_EPOCH
