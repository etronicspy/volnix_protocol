"""BotEngine: разгон, единый admission, probes."""
from __future__ import annotations

import random

import pytest

from core.bot_engine import BOT_ADDRESS_PREFIX, BotEngine
from core.models import Role
from core.state import SIM_TREASURY_ADDR


@pytest.fixture
def bot(state_manager):
    random.seed(42)
    return BotEngine(state_manager)


def test_bot_creates_new_address_via_treasury_mint(bot, state_manager):
    """Стартовый разгон: бот должен поставить mint WRT из казны через единый admission."""
    initial_mempool = len(state_manager.mempool)
    bot.generate_traffic()
    assert len(state_manager.mempool) == initial_mempool + 1
    last_tx = state_manager.mempool[-1]
    assert last_tx.tx_type.value == "mint"
    assert last_tx.sender == SIM_TREASURY_ADDR
    assert last_tx.receiver.startswith(BOT_ADDRESS_PREFIX)


def test_bot_intensity_clamped(state_manager):
    b = BotEngine(state_manager)
    b.set_intensity(0.001)
    assert b.tx_per_second == 0.1
    b.set_intensity(9999.0)
    assert b.tx_per_second == 100.0


def test_step_once_idle_when_stopped(bot, state_manager):
    bot.is_running = False
    before = len(state_manager.mempool)
    assert bot.step_once() == 0
    assert len(state_manager.mempool) == before


def test_step_once_runs_actions_per_block(bot, state_manager):
    """intensity=1 → до 60 действий на блок (sim-sec × interval), не 1 wall-clock tx."""
    from core.bot_engine import MAX_BOT_ACTIONS_PER_BLOCK

    bot.is_running = True
    bot.set_intensity(1.0)
    bot.enable_probes = False
    n = bot.step_once()
    assert n == MAX_BOT_ACTIONS_PER_BLOCK
    assert len(state_manager.mempool) >= 1


def test_step_once_scales_with_low_intensity(bot, state_manager):
    bot.is_running = True
    bot.set_intensity(0.1)  # 0.1 × 60 = 6 действий / блок
    bot.enable_probes = False
    n = bot.step_once()
    assert n == 6


def test_bot_probe_settings_toggle(state_manager):
    b = BotEngine(state_manager)
    b.set_probe_settings(enable=False, ratio=0.4, transfer_ant=False)
    assert b.enable_probes is False
    assert b.probe_ratio == 0.4
    assert b.probe_transfer_ant is False


def test_bot_probe_ratio_clamped(state_manager):
    b = BotEngine(state_manager)
    b.set_probe_settings(ratio=2.0)
    assert b.probe_ratio == 1.0
    b.set_probe_settings(ratio=-1.0)
    assert b.probe_ratio == 0.0


def test_bot_submit_op_rejects_to_canon_log(bot, state_manager, mk_account):
    """Если admission отклоняет tx бота — запись попадает в canon_log."""
    initial = len(state_manager.canon_log.to_list_newest_first())
    # transfer to self — отклоняется в admission
    tx = bot._submit_op(
        op="transfer",
        address="bot_self",
        action="transfer",
        detail="self-transfer",
        to_address="bot_self",
        amount=1.0,
    )
    assert tx is None
    after = state_manager.canon_log.to_list_newest_first()
    assert len(after) > initial
    assert any(
        entry.get("status") == "reject" and "bot:" in (entry.get("detail") or "")
        for entry in after
    )


def test_bot_does_not_touch_genesis_accounts(state_manager, mk_account):
    """После разгона бот не должен модифицировать genesis/казну как создатель сделок."""
    for i in range(15):
        mk_account(f"{BOT_ADDRESS_PREFIX}{i:08x}", role=Role.CITIZEN, wrt=100.0)
    bot = BotEngine(state_manager)
    bot.enable_probes = False
    random.seed(7)
    for _ in range(20):
        bot.generate_traffic()
    for tx in state_manager.mempool:
        if not tx.sender:
            continue
        assert tx.sender == SIM_TREASURY_ADDR or tx.sender.startswith(BOT_ADDRESS_PREFIX), (
            f"unexpected sender {tx.sender}"
        )


def test_lzn_equipment_market_uses_order_book(state_manager, mk_account):
    """§5.2 5.0-sim: LZN покупается через книгу, не MsgSend."""
    seller = mk_account(
        f"{BOT_ADDRESS_PREFIX}prov_lzn",
        role=Role.PROVIDER,
        lzn=100.0,
        zkp=True,
    )
    buyer = mk_account(
        f"{BOT_ADDRESS_PREFIX}buyer01",
        role=Role.VALIDATOR,
        wrt=500.0,
        zkp=True,
    )
    state_manager.last_lzn_price = 2.0
    bot = BotEngine(state_manager)
    random.seed(3)

    bot._trade_lzn_equipment([buyer])

    orders = [tx for tx in state_manager.mempool if tx.tx_type.value == "create_order"]
    assert len(orders) >= 2
    sell = next(t for t in orders if t.order_type and t.order_type.value == "sell")
    buy = next(t for t in orders if t.order_type and t.order_type.value == "buy")
    assert sell.sender == seller.address
    assert sell.asset_type == "lzn"
    assert buy.sender == buyer.address
    assert buy.asset_type == "lzn"
    assert buy.market is True


def test_lzn_equipment_market_respects_activation_cap(state_manager, mk_account):
    """§4.2: покупателю сверх потолка ⌊L/3⌋ поставка не идёт."""
    from core.state import LZN_MAX_FROZEN_PER_ADDRESS

    mk_account(
        f"{BOT_ADDRESS_PREFIX}prov_full",
        role=Role.PROVIDER,
        lzn=100.0,
        zkp=True,
    )
    full = mk_account(
        f"{BOT_ADDRESS_PREFIX}full01",
        role=Role.VALIDATOR,
        wrt=500.0,
        frozen=float(LZN_MAX_FROZEN_PER_ADDRESS),
        zkp=True,
    )
    bot = BotEngine(state_manager)
    before = len(state_manager.mempool)
    bot._trade_lzn_equipment([full])
    assert len(state_manager.mempool) == before
