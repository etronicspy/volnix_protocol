"""AutoMarketDaemon — эталонная стратегия внутреннего рынка ANT (§5.2).

`AutoDeclareDaemon` сжигает ANT, но ничего его не пополняет: у genesis-валидатора
бюджет ровно на одну эпоху (§6.3), после чего «электричество» кончается и по §5.4
блоки перестают производиться. Канон закрывает контур рынком — Поставщик продаёт
эмитированный ANT, Валидатор покупает его за WRT, — но в симуляторе книгу двигал
только `BotEngine`: он ходит по реальному времени и не трогает genesis-аккаунты.

Демон реализует то, что в реальной цепи делает клиент каждого участника:

* Поставщик выставляет SELL на свой запас — до §5.5 его всё равно сожжёт wipe;
* Валидатор с активированным LZN держит запас ANT на `ANT_BUFFER_BLOCKS` вперёд
  и докупает недостающее за WRT.

Цена **нигде не ограничена** — ни в консенсусе, ни правилом у продавца.
Поставщик ведёт себя как обычный маркет-мейкер: разбирают партию — пробует
дороже, висит непроданной у границы эпохи — уступает. Потолок возникает сам,
из отказа покупателя: майнер не платит выше `miner_reservation_price`, потому
что выше него выгоднее простаивать. Вставший майнер выпадает из `L_active`,
доля оставшихся в награде §5.1 растёт вместе с их пределом — рынок находит
равновесие без внешнего регулятора.
"""
from __future__ import annotations

import asyncio
import math
from typing import Dict, Optional, Tuple

from core.canon_audit import log_wallet_rejection
from core.engine import BURN_CAP_LAMBDA, SimulationEngine
from core.models import OrderType, Role
from core.profit_strategy import BLOCK_REWARD_WRT
from core.state import BLOCKS_PER_EPOCH, StateManager
from core.wallet_validate import validate_and_build_tx

MIN_TICK_SEC = 0.5

# Запас «электричества», который валидатор держит на балансе, в блоках вперёд.
ANT_BUFFER_BLOCKS = 64.0
# Доля запаса, ниже которой валидатор идёт докупать (гистерезис против спама заявок).
REFILL_TRIGGER = 0.75
# Шаг пересмотра котировки: вверх — когда берут, вниз — когда стоит.
ASK_STEP = 0.04
# Как часто Поставщик пересматривает цену. Майнер закупается не каждый блок:
# он держит буфер на `ANT_BUFFER_BLOCKS` и приходит, когда тот просядет до
# `REFILL_TRIGGER`, то есть раз в ~16 блоков. Более частый пересмотр принимает
# обычную паузу между закупками за отказ покупать и гонит цену вниз.
REVIEW_INTERVAL_BLOCKS = 32
# Относительное отклонение котировки от цели, после которого ask переставляется.
REQUOTE_THRESHOLD = 0.03
# Доля запаса, которую Поставщик выставляет одной заявкой.
PROVIDER_LOT_FRACTION = 0.1
MIN_LOT_ANT = 0.01
PRICE_DECIMALS = 6


def _floor_price(value: float) -> float:
    """Округление цены строго вниз: к ближайшему — пробивает лимиты на 1e-8."""
    scale = 10.0 ** PRICE_DECIMALS
    return max(1.0 / scale, math.floor(float(value) * scale) / scale)


def miner_reservation_price(sm: StateManager, acc) -> float:
    """Предельная цена, выше которой майнеру выгоднее простаивать.

    Награда §5.1 делится ∝ `L_i` между валидаторами с `b_i > 0`, а сжигает майнер
    `λ·L_i`, поэтому предел это `BLOCK_REWARD / (λ · L_active)`, где `L_active` —
    активированный LZN тех, кто реально жжёт. Это **решение покупателя**, а не
    правило протокола: в консенсусе цена ничем не ограничена.

    Отсюда же саморегуляция рынка: майнер, которому цена не по карману, выпадает
    из `L_active`, доля оставшихся в награде растёт, и их предел поднимается —
    цена находит равновесие без внешнего потолка.
    """
    l_i = float(acc.lzn_frozen_mining)
    l_active = l_i + sum(
        float(a.lzn_frozen_mining)
        for a in sm.accounts.values()
        if a.role == Role.VALIDATOR
        and a.address != acc.address
        and a.ant_balance > MIN_LOT_ANT
    )
    if l_active <= 0:
        return 1.0
    return max(10.0 ** -PRICE_DECIMALS, BLOCK_REWARD_WRT / (BURN_CAP_LAMBDA * l_active))


def demand_pressure(sm: StateManager) -> float:
    """Перекос книги в пользу спроса, 0…1 (0 — покупателей нет, 1 — товара нет)."""
    bid_vol = 0.0
    ask_vol = 0.0
    for o in sm.orders.values():
        rem = max(0.0, o.amount - o.filled)
        if o.order_type == OrderType.BUY:
            bid_vol += rem
        else:
            ask_vol += rem
    total = bid_vol + ask_vol
    if total <= 0:
        return 0.0
    return bid_vol / total


# Состояние котировок Поставщиков: адрес → (высота пересмотра, выручка WRT на
# тот момент, выставленная цена). В реальной цепи это память клиента продавца,
# ончейн её нет.
_quotes: Dict[str, Tuple[int, float, float]] = {}


def reset_quotes() -> None:
    """Забыть котировки — при перезапуске симуляции с чистого genesis."""
    _quotes.clear()


def provider_ask_price(sm: StateManager, acc, bootstrap: float) -> float:
    """Котировка Поставщика — обычный маркет-мейкер, без потолка.

    Никакого знания о «безубыточности майнера» у продавца нет: он видит только,
    разбирают его партию или она висит. Разобрали — пробует дороже; висит —
    уступает.

    Шаг вниз симметричен шагу вверх и не зависит от размера запаса. Скидка не
    увеличивает объём: спрос на «электричество» неэластичен по количеству —
    майнеры возьмут ровно `λ·L_total` за блок независимо от цены, — поэтому
    сбрасывать цену из-за большого остатка бессмысленно, он сгорит в любом
    случае. Снижать имеет смысл ровно до уровня, на котором товар берут.

    Противостояния продавца и покупателя при этом не возникает: себестоимость
    ANT нулевая (эмиссия §5.5), а непроданный остаток сгорает на границе эпохи,
    так что не продать всегда хуже, чем продать дёшево — Поставщик уступает
    первым.

    Потолок цены не задаётся ни здесь, ни в консенсусе. Его формирует отказ
    покупателя: майнер не платит выше `miner_reservation_price`.
    """
    h = int(getattr(sm, "current_height", 0) or 0)
    prev = _quotes.get(acc.address)
    if prev is None or h < prev[0]:
        # Первая котировка (или сброс симуляции — высота ушла назад).
        ask = _floor_price(float(getattr(sm, "last_price", 0.0) or 0.0) or bootstrap)
        _quotes[acc.address] = (h, float(acc.wrt_balance), ask)
        return ask

    reviewed_at, wrt_at_review, ask = prev
    if h - reviewed_at < REVIEW_INTERVAL_BLOCKS:
        return ask

    # Продал ли он хоть что-то за окно. Считаем по выручке, а не по `filled`
    # текущей заявки: при перестановке котировки ордер пересоздаётся, и его
    # `filled` обнуляется. Раньше сигнал читался именно оттуда, поэтому каждая
    # удачная сделка сразу после requote выглядела как «не берут» — цена уезжала
    # вниз на шаг за сделку, монотонным храповиком до пола.
    sold = float(acc.wrt_balance) > wrt_at_review + 1e-9
    if sold:
        # Берут — нащупываем цену вверх; при перекосе книги в спрос смелее.
        adj = ASK_STEP * (1.0 + demand_pressure(sm))
    else:
        # Не берут — шагаем вниз, пока не начнут.
        adj = -ASK_STEP

    ask = _floor_price(ask * (1.0 + adj))
    _quotes[acc.address] = (h, float(acc.wrt_balance), ask)
    return ask


def _has_pending_market_tx(sm: StateManager, address: str) -> bool:
    """Есть ли у адреса неисполненная заявка/отмена в мемпуле.

    Без этой проверки демон на каждом тике пересчитывает лот от ещё не
    списанного баланса и шлёт дубли: первая заявка забирает ANT в эскроу,
    остальные отбиваются «недостаточно ANT», а отмены — «ордер не найден».
    """
    pending = list(sm.mempool)
    net = getattr(sm, "network", None)
    if net is not None:
        try:
            pending.extend(net.iter_pending_txs())
        except Exception:
            pass
    return any(
        tx.sender == address
        and tx.tx_type.value in ("create_order", "cancel_order")
        for tx in pending
    )


def _open_volume(sm: StateManager, address: str, side: OrderType) -> float:
    return sum(
        o.amount - o.filled
        for o in sm.orders.values()
        if o.owner == address and o.order_type == side
    )


def _cancel_order(sm: StateManager, address: str, order_id: str) -> bool:
    ok, msg, tx = validate_and_build_tx(sm, "cancel_order", address, order_id=order_id)
    if not ok or tx is None:
        log_wallet_rejection(sm, "cancel_order", f"auto_market: {msg}", address)
        return False
    sm.submit_tx(tx)
    return True


def _submit_order(
    sm: StateManager, address: str, side: str, price: float, amount: float
) -> bool:
    ok, msg, tx = validate_and_build_tx(
        sm, "create_order", address, side=side, price=float(price), amount=float(amount)
    )
    if not ok or tx is None:
        log_wallet_rejection(sm, "create_order", f"auto_market: {msg}", address)
        return False
    sm.submit_tx(tx)
    return True


def step_once(sm: StateManager, engine: SimulationEngine) -> int:
    """Один проход по обеим сторонам книги; вернуть число поданных заявок."""
    l_total = engine._network_lzn_total_validators()
    if l_total <= 0:
        return 0

    submitted = 0
    # Стартовая догадка до первой сделки: во что «электричество» обходится сети.
    bootstrap = BLOCK_REWARD_WRT / (BURN_CAP_LAMBDA * l_total)

    # Сторона предложения: Поставщик выставляет часть запаса и переставляет
    # котировку, когда цена ушла — иначе в книге навсегда висит устаревший ask.
    for addr, acc in sorted(sm.accounts.items()):
        if acc.role != Role.PROVIDER or acc.ant_balance <= MIN_LOT_ANT:
            continue
        if _has_pending_market_tx(sm, addr):
            continue
        ask = provider_ask_price(sm, acc, bootstrap)
        stale = [
            o for o in sm.orders.values()
            if o.owner == addr and o.order_type == OrderType.SELL
            and abs(o.price - ask) > ask * REQUOTE_THRESHOLD
        ]
        for o in stale:
            _cancel_order(sm, addr, o.id)
        if _open_volume(sm, addr, OrderType.SELL) > 0 and not stale:
            continue
        lot = round(acc.ant_balance * PROVIDER_LOT_FRACTION, 6)
        if lot < MIN_LOT_ANT:
            continue
        if _submit_order(sm, addr, "sell", ask, lot):
            submitted += 1

    # Сторона спроса: Валидатор добирает запас до ANT_BUFFER_BLOCKS блоков вперёд.
    best_ask = min(
        (o.price for o in sm.orders.values() if o.order_type == OrderType.SELL),
        default=bootstrap,
    )
    for addr, acc in sorted(sm.accounts.items()):
        if acc.role != Role.VALIDATOR:
            continue
        l_i = float(acc.lzn_frozen_mining)
        if l_i <= 0 or _has_pending_market_tx(sm, addr):
            continue
        target = BURN_CAP_LAMBDA * l_i * ANT_BUFFER_BLOCKS
        held = float(acc.ant_balance) + _open_volume(sm, addr, OrderType.BUY)
        if held >= target * REFILL_TRIGGER:
            continue
        # Собственный предел покупателя: выше него «оборудование» выгоднее
        # остановить, чем работать в минус. Это и есть регулятор цены.
        reservation = miner_reservation_price(sm, acc)
        bid = _floor_price(min(best_ask, reservation))
        want = target - held
        affordable = float(acc.wrt_balance) / bid if bid > 0 else 0.0
        lot = round(min(want, affordable), 6)
        if lot < MIN_LOT_ANT:
            continue
        if _submit_order(sm, addr, "buy", bid, lot):
            submitted += 1

    return submitted


async def run(
    sm: StateManager,
    engine: SimulationEngine,
    *,
    stop_event: Optional[asyncio.Event] = None,
) -> None:
    """Бесконечный цикл: каждые `block_time/2` (но не реже MIN_TICK_SEC) — step_once."""
    while True:
        if stop_event is not None and stop_event.is_set():
            return
        try:
            step_once(sm, engine)
        except Exception as e:
            print(f"AutoMarketDaemon tick error: {e}")
        delay = max(MIN_TICK_SEC, float(engine.block_time) / 2.0)
        try:
            await asyncio.sleep(delay)
        except asyncio.CancelledError:
            return
