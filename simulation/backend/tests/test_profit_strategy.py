"""ProfitStrategy: role choice by expected WRT, not at random."""
from __future__ import annotations

from core.models import Order, OrderType, Role
from core.profit_strategy import ProfitStrategy, top_switch_candidates
from core.state import eligible_for_provider_role, eligible_for_validator_role


def test_citizen_without_zkp_cannot_switch(mk_account, state_manager):
    acc = mk_account("bot_plain", role=Role.CITIZEN, wrt=100.0)
    strategy = ProfitStrategy()
    assert strategy.best_role(acc.address, acc, state_manager) is None
    assert not eligible_for_provider_role(acc.address, acc)
    assert not eligible_for_validator_role(acc.address, acc)


def test_zkp_citizen_prefers_provider_over_idle(mk_account, state_manager):
    acc = mk_account("bot_zkp", role=Role.CITIZEN, wrt=80.0, zkp=True)
    state_manager.last_price = 10.0
    state_manager.epoch_ant_sold_last = 200.0
    state_manager.epoch_emission_coefficient = 1.0
    strategy = ProfitStrategy()
    assert strategy.best_role(acc.address, acc, state_manager) == Role.PROVIDER
    assert strategy.desired_role(acc.address, acc, state_manager) == Role.PROVIDER


def test_lzn_heavy_bot_prefers_validator_when_set_is_thin(mk_account, state_manager):
    acc = mk_account(
        "bot_miner",
        role=Role.PROVIDER,
        wrt=200.0,
        lzn=0.0,
        frozen=80.0,
        ant=400.0,
        zkp=True,
    )
    state_manager.last_price = 1.0
    state_manager.epoch_ant_sold_last = 1.0
    state_manager.epoch_emission_coefficient = 0.75
    strategy = ProfitStrategy()
    est = strategy.estimates(acc, state_manager)
    assert est[Role.VALIDATOR].net > est[Role.PROVIDER].net
    assert strategy.best_role(acc.address, acc, state_manager) == Role.VALIDATOR


def test_provider_crowding_lowers_roi(mk_account, state_manager):
    lonely = mk_account("bot_lone", role=Role.CITIZEN, wrt=50.0, zkp=True)
    state_manager.last_price = 8.0
    state_manager.epoch_ant_sold_last = 100.0
    strategy = ProfitStrategy()
    lone_provider = strategy.estimates(lonely, state_manager)[Role.PROVIDER].net

    for i in range(20):
        mk_account(f"bot_crowd{i:02d}", role=Role.PROVIDER, zkp=True)
    crowded = strategy.estimates(lonely, state_manager)[Role.PROVIDER].net
    assert crowded < lone_provider


def test_hysteresis_skips_tiny_edge(mk_account, state_manager):
    acc = mk_account("bot_hold", role=Role.PROVIDER, wrt=10.0, ant=1.0, zkp=True)
    state_manager.last_price = 5.0
    state_manager.epoch_ant_sold_last = 0.01
    strategy = ProfitStrategy()
    est = strategy.estimates(acc, state_manager)
    # Force almost-equal nets so margin blocks the flip.
    est[Role.CITIZEN].income_wrt = est[Role.PROVIDER].net + 0.5
    strategy._roi_cache[acc.address] = {r.value: roi for r, roi in est.items()}
    # Re-run best_role with a patched estimate by monkeypatching estimates
    original = strategy.estimates
    strategy.estimates = lambda account, state: est  # type: ignore[method-assign]
    try:
        assert strategy.best_role(acc.address, acc, state_manager) is None
    finally:
        strategy.estimates = original  # type: ignore[method-assign]


def test_citizen_switch_costs_ant(mk_account, state_manager):
    acc = mk_account("bot_v", role=Role.VALIDATOR, ant=20.0, frozen=5.0, zkp=True)
    state_manager.last_price = 4.0
    strategy = ProfitStrategy()
    citizen = strategy.estimates(acc, state_manager)[Role.CITIZEN]
    assert citizen.cost_wrt == 80.0


def test_top_switch_candidates_caps_and_orders(mk_account, state_manager):
    state_manager.last_price = 10.0
    state_manager.epoch_ant_sold_last = 300.0
    bots = [
        mk_account(f"bot_c{i}", role=Role.CITIZEN, wrt=50.0, zkp=True)
        for i in range(8)
    ]
    strategy = ProfitStrategy()
    picked = top_switch_candidates(strategy, bots, state_manager, limit=3)
    assert len(picked) <= 3
    assert all(role == Role.PROVIDER for _, role, _ in picked)
    deltas = [d for _, _, d in picked]
    assert deltas == sorted(deltas, reverse=True)


def test_bid_heavy_book_raises_provider_efficiency(state_manager, mk_account):
    strategy = ProfitStrategy()
    empty = strategy._provider_sell_efficiency(state_manager)
    state_manager.orders["b1"] = Order(
        id="b1",
        owner="x",
        order_type=OrderType.BUY,
        price=10.0,
        amount=100.0,
        timestamp=0.0,
    )
    state_manager.orders["s1"] = Order(
        id="s1",
        owner="y",
        order_type=OrderType.SELL,
        price=11.0,
        amount=10.0,
        timestamp=0.0,
    )
    tight = strategy._provider_sell_efficiency(state_manager)
    assert tight > empty


def test_bot_role_switch_only_at_half_epoch(mk_account, state_manager):
    """Citizen+ZKP → provider only on height % (BLOCKS_PER_EPOCH/2) == 0."""
    import random

    from core.bot_engine import ROLE_REEVAL_INTERVAL_BLOCKS, BotEngine
    from core.state import BLOCKS_PER_EPOCH

    assert ROLE_REEVAL_INTERVAL_BLOCKS == BLOCKS_PER_EPOCH // 2

    for i in range(15):
        mk_account(f"bot_{i:08x}", role=Role.CITIZEN, wrt=80.0, zkp=True)
    state_manager.last_price = 9.0
    state_manager.epoch_ant_sold_last = 150.0
    bot = BotEngine(state_manager)
    bot.enable_probes = False

    original_choices = random.choices
    original_random = random.random
    random.choices = lambda seq, weights=None: ["transfer"]  # type: ignore[assignment]
    random.random = lambda: 1.0  # type: ignore[assignment]
    try:
        state_manager.current_height = ROLE_REEVAL_INTERVAL_BLOCKS - 1
        bot.generate_traffic()
        mid_txs = [tx for tx in state_manager.mempool if tx.tx_type.value == "set_role"]
        assert mid_txs == []

        state_manager.current_height = ROLE_REEVAL_INTERVAL_BLOCKS
        before = len(state_manager.mempool)
        bot.generate_traffic()
        role_txs = [
            tx for tx in state_manager.mempool[before:]
            if tx.tx_type.value == "set_role"
        ]
        assert role_txs
        assert all(tx.role == Role.PROVIDER for tx in role_txs)

        before = len(state_manager.mempool)
        bot.generate_traffic()
        again = [
            tx for tx in state_manager.mempool[before:]
            if tx.tx_type.value == "set_role"
        ]
        assert again == []
    finally:
        random.choices = original_choices  # type: ignore[assignment]
        random.random = original_random  # type: ignore[assignment]


def test_bot_role_reeval_fires_when_height_jumps_over_boundary(mk_account, state_manager):
    """На высоких скоростях бот не видит точное кратное — переоценка всё равно нужна."""
    import random

    from core.bot_engine import ROLE_REEVAL_INTERVAL_BLOCKS, BotEngine

    for i in range(15):
        mk_account(f"bot_{i:08x}", role=Role.CITIZEN, wrt=80.0, zkp=True)
    state_manager.last_price = 9.0
    state_manager.epoch_ant_sold_last = 150.0
    bot = BotEngine(state_manager)
    bot.enable_probes = False

    original_choices = random.choices
    original_random = random.random
    random.choices = lambda seq, weights=None: ["transfer"]  # type: ignore[assignment]
    random.random = lambda: 1.0  # type: ignore[assignment]
    try:
        # Высота перескочила границу: ни разу не была кратна интервалу.
        state_manager.current_height = ROLE_REEVAL_INTERVAL_BLOCKS + 137
        bot.generate_traffic()
        role_txs = [tx for tx in state_manager.mempool if tx.tx_type.value == "set_role"]
        assert role_txs
    finally:
        random.choices = original_choices  # type: ignore[assignment]
        random.random = original_random  # type: ignore[assignment]


def test_bot_set_role_action_is_prep_only(mk_account, state_manager):
    """Random set_role traffic must not flip roles between half-epoch windows."""
    import random

    from core.bot_engine import BotEngine

    mk_account("bot_aaaa0001", role=Role.CITIZEN, wrt=80.0, zkp=True)
    state_manager.current_height = 100
    state_manager.last_price = 9.0
    state_manager.epoch_ant_sold_last = 150.0
    bot = BotEngine(state_manager)
    bot.enable_probes = False
    original_choices = random.choices
    original_random = random.random
    random.choices = lambda seq, weights=None: ["set_role"]  # type: ignore[assignment]
    random.random = lambda: 1.0  # type: ignore[assignment]
    try:
        bot.generate_traffic()
    finally:
        random.choices = original_choices  # type: ignore[assignment]
        random.random = original_random  # type: ignore[assignment]
    assert not any(tx.tx_type.value == "set_role" for tx in state_manager.mempool)


def test_bot_prep_submits_zkp_when_ineligible(mk_account, state_manager):
    from core.bot_engine import BotEngine

    acc = mk_account("bot_needzkp", role=Role.CITIZEN, wrt=60.0, zkp=False)
    state_manager.last_price = 8.0
    state_manager.epoch_ant_sold_last = 120.0
    bot = BotEngine(state_manager)
    assert bot._try_profit_prep(acc) is True
    assert any(tx.tx_type.value == "zkp_verify" for tx in state_manager.mempool)
