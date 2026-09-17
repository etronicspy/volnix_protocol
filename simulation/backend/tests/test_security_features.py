"""
Tests for security subsystems (core/security.py + engine integration).

Covers:
  1. MOA — role strip on inactivity, ANT burn, LZN preserved
  2. Min-burn liveness — block stall when Σb_i < λ·L_total
  3. ValidatorSet recalculation by w_i = s_i/L_i
  4. ZKP nullifier — duplicate identity rejection
  5. Supplier cap — MAX_ACTIVE_SUPPLIERS limit
  6. Epoch wash-trade — self-trade inflates emission
  7. Proposer MEV — tx reordering
"""
from __future__ import annotations

import time
import uuid

import pytest

from core.engine import BURN_CAP_LAMBDA, SimulationEngine
from core.models import Order, OrderType, Role, Transaction, TransactionType
from core.security import (
    MOATracker,
    SecurityConfig,
    ZKPRegistry,
    check_min_burn_threshold,
    check_supplier_cap,
    recalculate_validator_set,
    reorder_mempool_mev,
)
from core.state import (
    GENESIS_VALIDATOR_ADDR,
    SIM_TREASURY_ADDR,
    StateManager,
)


def _mk(tx_type: TransactionType, **kw) -> Transaction:
    defaults = {"tx_hash": uuid.uuid4().hex, "timestamp": time.time()}
    defaults.update(kw)
    return Transaction(tx_type=tx_type, **defaults)


def _declare_for_gv(engine: SimulationEngine) -> Transaction:
    """Набор низа коридора: declare для всех валидаторов с L_i>0."""
    from tests.conftest import enqueue_corridor_declares, seed_declare_tx

    enqueue_corridor_declares(engine)
    return seed_declare_tx(engine)


@pytest.fixture
def sec_config():
    return SecurityConfig(
        moa_enabled=True,
        moa_window_supplier_blocks=10,
        moa_window_validator_blocks=5,
        min_burn_enabled=False,
        validator_set_update_enabled=True,
        max_active_validators_k=150,
        zkp_nullifier_enabled=True,
        supplier_cap_enabled=True,
        max_active_suppliers=3,
        mev_reorder_enabled=True,
        mev_reorder_mode="fee_priority",
    )


@pytest.fixture
def sec_engine(state_manager, sec_config):
    """SimulationEngine with all security features enabled."""
    return SimulationEngine(state_manager, security_config=sec_config)


# ============================================================================
# 1. MOA — Mechanism of Accountability (§3.3, §5.3)
# ============================================================================


class TestMOA:
    def test_moa_tracker_no_sanction_within_window(self, sec_config):
        tracker = MOATracker(config=sec_config)
        from core.state import StateManager
        sm = StateManager.__new__(StateManager)
        sm.accounts = {}
        from core.models import Account
        acc = Account(address="val1", role=Role.VALIDATOR, ant_balance=50.0, lzn_frozen_mining=100.0)
        sm.accounts["val1"] = acc
        tracker.record_activity("val1", 100)
        sanctions = tracker.check_inactivity(sm, 104)  # within 5
        assert sanctions == []
        assert acc.role == Role.VALIDATOR

    def test_moa_tracker_sanction_after_window(self, sec_config):
        tracker = MOATracker(config=sec_config)
        from core.state import StateManager
        sm = StateManager.__new__(StateManager)
        sm.accounts = {}
        from core.models import Account
        acc = Account(address="val1", role=Role.VALIDATOR, ant_balance=50.0, lzn_frozen_mining=100.0)
        sm.accounts["val1"] = acc
        tracker.record_activity("val1", 100)
        sanctions = tracker.check_inactivity(sm, 106)  # window=5, 106-100=6 > 5
        assert len(sanctions) == 1
        assert acc.role == Role.CITIZEN
        assert acc.ant_balance == 0.0
        assert acc.lzn_frozen_mining == 100.0  # LZN preserved!

    def test_moa_provider_longer_window(self, sec_config):
        tracker = MOATracker(config=sec_config)
        from core.state import StateManager
        sm = StateManager.__new__(StateManager)
        sm.accounts = {}
        from core.models import Account
        acc = Account(address="prov1", role=Role.PROVIDER, ant_balance=30.0)
        sm.accounts["prov1"] = acc
        tracker.record_activity("prov1", 50)
        # Still within 10-block window
        sanctions = tracker.check_inactivity(sm, 59)
        assert sanctions == []
        # Beyond window
        sanctions = tracker.check_inactivity(sm, 61)
        assert len(sanctions) == 1
        assert acc.role == Role.CITIZEN
        assert acc.ant_balance == 0.0

    def test_moa_citizen_not_affected(self, sec_config):
        tracker = MOATracker(config=sec_config)
        from core.state import StateManager
        sm = StateManager.__new__(StateManager)
        sm.accounts = {}
        from core.models import Account
        acc = Account(address="cit", role=Role.CITIZEN, wrt_balance=100.0)
        sm.accounts["cit"] = acc
        sanctions = tracker.check_inactivity(sm, 9999)
        assert sanctions == []

    @pytest.mark.asyncio
    async def test_moa_integrated_in_begin_block(self, sec_engine, mk_account):
        """MOA fires in BeginBlock and strips inactive validator."""
        # frozen=0: иначе коридорный declare за lazy_val считался бы MOA-активностью.
        v = mk_account("lazy_val", role=Role.VALIDATOR, frozen=0.0, lzn=10.0, ant=25.0, zkp=True)
        sec_engine.moa_tracker.record_activity(v.address, 0)
        for _ in range(6):
            sec_engine.state.mempool.append(_declare_for_gv(sec_engine))
            await sec_engine.produce_block()

        v_after = sec_engine.state.accounts[v.address]
        assert v_after.role == Role.CITIZEN
        assert v_after.ant_balance == 0.0
        assert v_after.lzn_balance == 10.0


# ============================================================================
# 2. Min-burn liveness (canon v4.20 strict)
# ============================================================================


class TestBurnCorridor:
    def test_burn_floor_and_ceiling(self):
        from core.state import BURN_CAP_LAMBDA, BURN_CAP_LAMBDA_MAX, burn_ceiling, burn_floor

        L = 1200.0
        assert burn_floor(L) == pytest.approx(BURN_CAP_LAMBDA * L)
        assert burn_ceiling(L) == pytest.approx((1.0 - BURN_CAP_LAMBDA) * L)
        assert burn_floor(L) < burn_ceiling(L)
        assert BURN_CAP_LAMBDA <= BURN_CAP_LAMBDA_MAX + 1e-12
        # при λ_max=5/12 ширина ≥ 1/6
        width_at_max = burn_ceiling(L, BURN_CAP_LAMBDA_MAX) - burn_floor(L, BURN_CAP_LAMBDA_MAX)
        assert width_at_max == pytest.approx(L / 6.0)

    def test_min_burn_disabled_always_passes(self):
        cfg = SecurityConfig(min_burn_enabled=False)
        assert check_min_burn_threshold(0.0, 1000.0, 1 / 3, cfg) is True

    def test_min_burn_enabled_passes_at_threshold(self):
        cfg = SecurityConfig(min_burn_enabled=True, min_burn_mode="canon")
        L = 6667.0
        threshold = (1 / 3) * L
        assert check_min_burn_threshold(threshold, L, 1 / 3, cfg) is True

    def test_min_burn_canon_mode_fails_below_threshold(self):
        cfg = SecurityConfig(min_burn_enabled=True, min_burn_mode="canon")
        L = 6667.0
        threshold = (1 / 3) * L
        assert check_min_burn_threshold(threshold - 1.0, L, 1 / 3, cfg) is False

    def test_min_burn_positive_mode_accepts_any_burn(self):
        """Режим отладки positive: блок валиден при любом Σb_i > 0, но не при нуле."""
        cfg = SecurityConfig(min_burn_enabled=True, min_burn_mode="positive")
        assert cfg.min_burn_mode == "positive"
        assert check_min_burn_threshold(1.0, 6667.0, 1 / 3, cfg) is True
        assert check_min_burn_threshold(0.0, 6667.0, 1 / 3, cfg) is False

    def test_default_min_burn_mode_is_canon(self):
        cfg = SecurityConfig()
        assert cfg.min_burn_mode == "canon"

    @pytest.mark.asyncio
    async def test_min_burn_block_rejected_when_enabled(self, state_manager, mk_account):
        """With min_burn_enabled, empty mempool → RuntimeError (liveness stall)."""
        cfg = SecurityConfig(min_burn_enabled=True)
        engine = SimulationEngine(state_manager, security_config=cfg)
        # Empty mempool → Σb_i = 0 < λ·L
        with pytest.raises(RuntimeError, match="min-burn"):
            await engine.produce_block()


# ============================================================================
# 3. ValidatorSet recalculation by w_i = s_i / L_i (§6.1)
# ============================================================================


class TestValidatorSet:
    def test_recalculate_sorts_by_weight(self):
        cfg = SecurityConfig(validator_set_update_enabled=True, max_active_validators_k=3)
        participation = {
            "v1": {"b": 10, "s": 50, "L_i": 100},
            "v2": {"b": 10, "s": 80, "L_i": 100},
            "v3": {"b": 10, "s": 30, "L_i": 100},
        }
        result = recalculate_validator_set(participation, None, cfg)
        assert len(result) == 3
        assert result[0]["address"] == "v2"  # w=0.8
        assert result[1]["address"] == "v1"  # w=0.5
        assert result[2]["address"] == "v3"  # w=0.3

    def test_recalculate_k_cap(self):
        cfg = SecurityConfig(validator_set_update_enabled=True, max_active_validators_k=2)
        participation = {
            f"v{i}": {"b": 10, "s": 100 - i, "L_i": 100}
            for i in range(5)
        }
        result = recalculate_validator_set(participation, None, cfg)
        assert len(result) == 2  # capped at K=2

    def test_recalculate_excludes_zero_burn(self):
        cfg = SecurityConfig(validator_set_update_enabled=True)
        participation = {
            "active": {"b": 10, "s": 50, "L_i": 100},
            "passive": {"b": 0, "s": 50, "L_i": 100},
        }
        result = recalculate_validator_set(participation, None, cfg)
        assert len(result) == 1
        assert result[0]["address"] == "active"

    @pytest.mark.asyncio
    async def test_validator_set_updated_after_block(self, sec_engine, mk_account):
        """After produce_block with declares, consensus_validator_set is updated."""
        v2 = mk_account("v2_vs", role=Role.VALIDATOR, frozen=10.0, ant=100.0, zkp=True)
        sec_engine.moa_tracker.record_activity(v2.address, 0)
        sec_engine.moa_tracker.record_activity(GENESIS_VALIDATOR_ADDR, 0)

        gv = sec_engine.state.accounts[GENESIS_VALIDATOR_ADDR]
        L = gv.lzn_frozen_mining + v2.lzn_frozen_mining
        # Both declare in same block
        sec_engine.state.mempool.append(_mk(
            TransactionType.DECLARE_PARTICIPATION,
            sender=GENESIS_VALIDATOR_ADDR, amount=5.0, stake_amount=50.0, asset_type="ant",
        ))
        sec_engine.state.mempool.append(_mk(
            TransactionType.DECLARE_PARTICIPATION,
            sender=v2.address, amount=5.0, stake_amount=8.0, asset_type="ant",
        ))
        await sec_engine.produce_block()

        valset = sec_engine.state.consensus_validator_set
        assert len(valset) >= 1
        addrs = [v["address"] for v in valset]
        assert GENESIS_VALIDATOR_ADDR in addrs


# ============================================================================
# 4. ZKP nullifier uniqueness (§3.1)
# ============================================================================


class TestZKPNullifier:
    def test_registry_allows_first(self):
        cfg = SecurityConfig(zkp_nullifier_enabled=True)
        reg = ZKPRegistry(config=cfg)
        assert reg.can_verify("hash_abc") is True
        assert reg.register("hash_abc") is True
        assert reg.can_verify("hash_abc") is False
        assert reg.register("hash_abc") is False

    def test_registry_disabled_allows_all(self):
        cfg = SecurityConfig(zkp_nullifier_enabled=False)
        reg = ZKPRegistry(config=cfg)
        assert reg.register("same") is True
        assert reg.register("same") is True

    @pytest.mark.asyncio
    async def test_duplicate_zkp_identity_rejected_in_block(self, sec_engine, mk_account):
        """Two accounts trying to verify with same identity hash → second rejected."""
        a1 = mk_account("user_a", role=Role.CITIZEN, zkp=False)
        a2 = mk_account("user_b", role=Role.CITIZEN, zkp=False)

        sec_engine.moa_tracker.record_activity(a1.address, 0)
        sec_engine.moa_tracker.record_activity(a2.address, 0)

        identity_hash = "unique_person_nullifier_123"
        sec_engine.state.mempool.append(_declare_for_gv(sec_engine))
        # Both try to verify with same identity
        sec_engine.state.mempool.append(_mk(
            TransactionType.ZKP_VERIFY, sender=a1.address, receiver=a1.address,
            details=identity_hash,
        ))
        sec_engine.state.mempool.append(_mk(
            TransactionType.ZKP_VERIFY, sender=a2.address, receiver=a2.address,
            details=identity_hash,
        ))
        await sec_engine.produce_block()

        assert sec_engine.state.accounts[a1.address].zkp_verified is True
        assert sec_engine.state.accounts[a2.address].zkp_verified is False

    @pytest.mark.asyncio
    async def test_different_identities_both_pass(self, sec_engine, mk_account):
        """Different identity hashes → both pass."""
        a1 = mk_account("user_c", role=Role.CITIZEN, zkp=False)
        a2 = mk_account("user_d", role=Role.CITIZEN, zkp=False)
        sec_engine.moa_tracker.record_activity(a1.address, 0)
        sec_engine.moa_tracker.record_activity(a2.address, 0)

        sec_engine.state.mempool.append(_declare_for_gv(sec_engine))
        sec_engine.state.mempool.append(_mk(
            TransactionType.ZKP_VERIFY, sender=a1.address, receiver=a1.address,
            details="person_1",
        ))
        sec_engine.state.mempool.append(_mk(
            TransactionType.ZKP_VERIFY, sender=a2.address, receiver=a2.address,
            details="person_2",
        ))
        await sec_engine.produce_block()

        assert sec_engine.state.accounts[a1.address].zkp_verified is True
        assert sec_engine.state.accounts[a2.address].zkp_verified is True


# ============================================================================
# 5. Supplier cap (§7.2 item 8)
# ============================================================================


class TestSupplierCap:
    def test_check_supplier_cap_under_limit(self, state_manager, sec_config):
        from tests.conftest import strip_bootstrap_economy

        # 5 bootstrap-поставщиков уже ≥ cap=3 — убираем их для чистого under-limit.
        strip_bootstrap_economy(state_manager)
        assert check_supplier_cap(state_manager, sec_config) is True

    def test_check_supplier_cap_at_limit(self, state_manager, sec_config):
        from tests.conftest import make_account, strip_bootstrap_economy

        strip_bootstrap_economy(state_manager)
        # max_active_suppliers = 3 in sec_config
        make_account(state_manager, "prov_a", role=Role.PROVIDER, zkp=True)
        make_account(state_manager, "prov_b", role=Role.PROVIDER, zkp=True)
        make_account(state_manager, "prov_c", role=Role.PROVIDER, zkp=True)
        assert check_supplier_cap(state_manager, sec_config) is False

    @pytest.mark.asyncio
    async def test_supplier_cap_rejects_role_change(self, sec_engine, mk_account):
        """When supplier cap is reached, set_role→Provider is rejected."""
        from tests.conftest import strip_bootstrap_economy

        strip_bootstrap_economy(sec_engine.state)
        mk_account("sp1", role=Role.PROVIDER, zkp=True)
        mk_account("sp2", role=Role.PROVIDER, zkp=True)
        mk_account("sp3", role=Role.PROVIDER, zkp=True)
        candidate = mk_account("sp4_cand", role=Role.CITIZEN, zkp=True)
        sec_engine.moa_tracker.record_activity(candidate.address, 0)
        sec_engine.moa_tracker.record_activity("sp1", 0)
        sec_engine.moa_tracker.record_activity("sp2", 0)
        sec_engine.moa_tracker.record_activity("sp3", 0)

        sec_engine.state.mempool.append(_declare_for_gv(sec_engine))
        sec_engine.state.mempool.append(_mk(
            TransactionType.SET_ROLE, sender=candidate.address,
            receiver=candidate.address, role=Role.PROVIDER,
        ))
        await sec_engine.produce_block()

        assert sec_engine.state.accounts[candidate.address].role == Role.CITIZEN


# ============================================================================
# 6. Epoch wash-trade — self-trade inflates emission coefficient
# ============================================================================


class TestWashTradeEpoch:
    @pytest.mark.asyncio
    async def test_self_trade_inflates_epoch_sold_volume(self, state_manager, mk_account):
        """Wash trading between V and P increases epoch_ant_sold_volume → affects emission."""
        engine = SimulationEngine(state_manager)
        actor_v = mk_account("wash_v", role=Role.VALIDATOR, frozen=10.0, wrt=500.0, zkp=True)
        actor_p = mk_account("wash_p", role=Role.PROVIDER, ant=100.0, zkp=True)

        vol_before = state_manager.epoch_ant_sold_volume

        # P sells to V via limit orders across blocks
        engine.state.mempool.append(
            _mk(TransactionType.CREATE_ORDER, sender=actor_p.address,
                order_type=OrderType.SELL, price=3.0, amount=20.0)
        )
        engine.state.mempool.append(_declare_for_gv(engine))
        await engine.produce_block()

        engine.state.mempool.append(
            _mk(TransactionType.CREATE_ORDER, sender=actor_v.address,
                order_type=OrderType.BUY, price=4.0, amount=20.0)
        )
        engine.state.mempool.append(_declare_for_gv(engine))
        await engine.produce_block()

        vol_after = state_manager.epoch_ant_sold_volume
        # Volume increased by trade amount
        assert vol_after > vol_before
        assert vol_after - vol_before == pytest.approx(20.0)


# ============================================================================
# 7. Proposer MEV — tx reordering (§5.2)
# ============================================================================


class TestMEV:
    def test_fee_priority_reorder(self):
        cfg = SecurityConfig(mev_reorder_enabled=True, mev_reorder_mode="fee_priority")
        txs = [
            _mk(TransactionType.TRANSFER, sender="a", receiver="b", amount=10.0),
            _mk(TransactionType.TRANSFER, sender="a", receiver="c", amount=100.0),
            _mk(TransactionType.TRANSFER, sender="a", receiver="d", amount=50.0),
        ]
        result = reorder_mempool_mev(txs, "proposer", cfg)
        # Highest amount first
        amounts = [tx.amount for tx in result]
        assert amounts == [100.0, 50.0, 10.0]

    def test_proposer_front_reorder(self):
        cfg = SecurityConfig(mev_reorder_enabled=True, mev_reorder_mode="proposer_front")
        txs = [
            _mk(TransactionType.TRANSFER, sender="user1", receiver="b", amount=10.0),
            _mk(TransactionType.TRANSFER, sender="proposer", receiver="c", amount=5.0),
            _mk(TransactionType.TRANSFER, sender="user2", receiver="d", amount=100.0),
        ]
        result = reorder_mempool_mev(txs, "proposer", cfg)
        assert result[0].sender == "proposer"
        assert len(result) == 3

    def test_mev_disabled_preserves_order(self):
        cfg = SecurityConfig(mev_reorder_enabled=False)
        txs = [
            _mk(TransactionType.TRANSFER, sender="a", amount=10.0),
            _mk(TransactionType.TRANSFER, sender="b", amount=100.0),
        ]
        result = reorder_mempool_mev(txs, "proposer", cfg)
        assert result[0].sender == "a"

    @pytest.mark.asyncio
    async def test_mev_reorder_affects_execution_order(self, sec_engine, mk_account):
        """With MEV enabled, higher-amount transfer executes first (gets priority)."""
        sender = mk_account("mev_sender", role=Role.CITIZEN, wrt=60.0)
        bob = mk_account("mev_bob", role=Role.CITIZEN)
        carol = mk_account("mev_carol", role=Role.CITIZEN)
        sec_engine.moa_tracker.record_activity(sender.address, 0)

        sec_engine.state.mempool.append(_declare_for_gv(sec_engine))
        # Small tx first in mempool, but MEV will reorder by amount
        sec_engine.state.mempool.append(
            _mk(TransactionType.TRANSFER, sender=sender.address,
                receiver=bob.address, amount=10.0, asset_type="wrt")
        )
        sec_engine.state.mempool.append(
            _mk(TransactionType.TRANSFER, sender=sender.address,
                receiver=carol.address, amount=50.0, asset_type="wrt")
        )
        await sec_engine.produce_block()

        # With fee_priority, 50 WRT tx goes first → carol gets 50
        # Then 10 WRT tx → only 10 left → bob gets 10
        assert sec_engine.state.accounts[carol.address].wrt_balance == 50.0
        assert sec_engine.state.accounts[bob.address].wrt_balance == 10.0
        assert sec_engine.state.accounts[sender.address].wrt_balance == 0.0
