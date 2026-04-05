import asyncio
import random
import time
import uuid
from core.state import (
    StateManager,
    SIM_TREASURY_ADDR,
    BLOCKS_PER_EPOCH,
    eligible_for_provider_role,
    eligible_for_validator_role,
)
from core.models import Role, Transaction, TransactionType, OrderType
from core.canon_audit import log_bot_queue

# Бот создаёт только адреса с префиксом bot_<hex>; не трогает genesis / казну / чужие кошельки.
BOT_ADDRESS_PREFIX = "bot_"


def is_bot_created_address(address: str) -> bool:
    return bool(address) and address.startswith(BOT_ADDRESS_PREFIX)


# Эпоха ANT ≈ календарная неделя (BLOCKS_PER_EPOCH); валидатор целится в ~10% WRT-выгоды за эпоху
# на капитал, уходящий в закупку ANT (лимит цены относительно last_price).
VALIDATOR_TARGET_WEEKLY_WRT_RETURN = 0.10


def _validator_bid_anchor_mult() -> float:
    """Не платить за ANT выше last_price/(1+r), иначе недельная цель по марже недостижима."""
    return 1.0 / (1.0 + VALIDATOR_TARGET_WEEKLY_WRT_RETURN)


def _blocks_until_next_epoch(current_height: int) -> int:
    """Блоки до следующего §5.5 сброса ANT у Поставщиков (граница эпохи)."""
    h = max(0, int(current_height))
    r = h % BLOCKS_PER_EPOCH
    if r == 0:
        return BLOCKS_PER_EPOCH
    return BLOCKS_PER_EPOCH - r


def _wallet_delta_last_block(state: StateManager, address: str) -> tuple[float, float]:
    """Δ WRT и Δ ANT кошелька за последний применённый блок (после produce_block)."""
    raw = state.last_block_wallet_delta.get(address) or {}
    return float(raw.get("wrt", 0.0)), float(raw.get("ant", 0.0))


def _provider_open_sell_ant_pending(state: StateManager, address: str) -> float:
    """ANT в эскроу открытых лимитных SELL (неисполненный остаток) — при §5.5 сгорит, если не исполнить."""
    total = 0.0
    for o in state.orders.values():
        if o.owner == address and o.order_type == OrderType.SELL:
            total += max(0.0, float(o.amount) - float(o.filled))
    return total


def _epoch_tension_for_provider(current_height: int) -> float:
    """
    0 — только что началась эпоха (ANT ещё «далеко» от сброса), 1 — вплотную к границе.
    Поставщик стремится к max WRT: в начале эпохи держит высокий ask, к концу снижает цену
    и активнее выставляет объём, чтобы не потерять ANT при wipe §5.5.
    """
    left = _blocks_until_next_epoch(current_height)
    return max(0.0, min(1.0, 1.0 - left / float(BLOCKS_PER_EPOCH)))


class BotEngine:
    def __init__(self, state_manager: StateManager):
        self.state = state_manager
        self.is_running = False
        self.tx_per_second = 1.0

    def set_intensity(self, tx_per_second: float):
        self.tx_per_second = max(0.1, min(tx_per_second, 100.0))

    def _queue_tx(self, tx: Transaction, action: str, detail: str) -> None:
        self.state.mempool.append(tx)
        log_bot_queue(self.state, action, detail, tx.tx_hash)

    async def start(self):
        self.is_running = True
        print(f"Bot Engine started. Intensity: {self.tx_per_second} tx/s")
        while self.is_running:
            try:
                self.generate_traffic()
            except Exception as e:
                print(f"Bot error: {e}")
            # Sleep based on intensity
            await asyncio.sleep(1.0 / self.tx_per_second)

    def stop(self):
        self.is_running = False
        print("Bot Engine stopped.")

    def generate_traffic(self):
        all_accounts = list(self.state.accounts.values())
        accounts = [a for a in all_accounts if is_bot_created_address(a.address)]
        bot_count = len(accounts)

        # 1. Разгон только кошельков bot_*; казна — лишь источник первого минта WRT
        if bot_count < 10 or (bot_count < 100 and random.random() < 0.05):
            new_addr = f"{BOT_ADDRESS_PREFIX}{uuid.uuid4().hex[:8]}"
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.MINT,
                sender=SIM_TREASURY_ADDR,
                receiver=new_addr,
                amount=round(random.uniform(50, 200), 2),
                asset_type="wrt",
                timestamp=time.time()
            )
            treasury = self.state.accounts.get(SIM_TREASURY_ADDR)
            if treasury and treasury.wrt_balance >= tx.amount:
                self._queue_tx(tx, "mint_wrt_new_bot", f"WRT казна → новый адрес {new_addr} (разгон книги)")
            return

        # 2. Сценарии нагрузки + редкие проверки канона (ожидаемые отклонения в блоке)
        action = random.choices(
            [
                "transfer",
                "trade",
                "cancel_order",
                "mint",
                "set_role",
                "declare",
                "zkp_verify",
                "canon_probe_mint_ant_citizen",
                "canon_probe_transfer_ant",
            ],
            weights=[0.24, 0.19, 0.09, 0.12, 0.07, 0.09, 0.06, 0.08, 0.06],
        )[0]

        if action == "declare":
            validators = [
                acc for acc in accounts
                if acc.role == Role.VALIDATOR
                and acc.lzn_frozen_mining > 0
                and acc.ant_balance > 0.01
            ]
            if validators:
                val = random.choice(validators)
                cap = float(val.lzn_frozen_mining)
                ant = float(val.ant_balance)
                mx = min(cap, ant)
                if mx >= 0.02:
                    total = round(random.uniform(0.01, mx), 4)
                    b = round(random.uniform(0.01, max(0.01, total * 0.85)), 4)
                    s = round(min(total - b, cap - b, ant - b), 4)
                    if s < 0:
                        s = 0.0
                    if b + s > cap + 1e-9 or b + s > ant + 1e-9:
                        s = max(0.0, min(cap - b, ant - b))
                    if b + s <= ant + 1e-9 and b + s <= cap + 1e-9 and b + s >= 0.01:
                        tx = Transaction(
                            tx_hash=uuid.uuid4().hex,
                            tx_type=TransactionType.DECLARE_PARTICIPATION,
                            sender=val.address,
                            amount=b,
                            stake_amount=s,
                            asset_type="ant",
                            timestamp=time.time(),
                        )
                        self._queue_tx(
                            tx,
                            "declare",
                            f"§5.4 b={b} s={s} (валидатор {val.address[:14]}…)",
                        )
            return

        elif action == "zkp_verify":
            candidates = [a for a in accounts if not a.zkp_verified]
            if candidates:
                t = random.choice(candidates)
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.ZKP_VERIFY,
                    sender=t.address,
                    receiver=t.address,
                    timestamp=time.time(),
                )
                self._queue_tx(tx, "zkp_verify", f"§3.1 симуляция → {t.address[:16]}…")
            return

        elif action == "set_role":
            pool = list(accounts)
            if not pool:
                return
            target = random.choice(pool)
            order = [Role.CITIZEN, Role.PROVIDER, Role.VALIDATOR]
            random.shuffle(order)
            new_role = None
            for nr in order:
                if nr == target.role:
                    continue
                if nr == Role.VALIDATOR and not eligible_for_validator_role(target.address, target):
                    continue
                if nr == Role.PROVIDER:
                    if not eligible_for_provider_role(target.address, target):
                        continue
                new_role = nr
                break
            if new_role is None:
                return
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.SET_ROLE,
                sender=target.address,
                receiver=target.address,
                role=new_role,
                timestamp=time.time(),
            )
            self._queue_tx(tx, "set_role", f"§4.2 → {new_role.value} для {target.address[:14]}…")
            return

        elif action == "mint":
            if not accounts:
                return
            target = random.choice(accounts)
            possible_assets = ["wrt", "lzn"]
            if target.role in (Role.PROVIDER, Role.VALIDATOR):
                possible_assets.append("ant")
            
            asset = random.choice(possible_assets)
            amount = round(random.uniform(10, 100), 2)
            
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.MINT,
                sender=SIM_TREASURY_ADDR,
                receiver=target.address,
                amount=amount,
                asset_type=asset,
                timestamp=time.time()
            )
            
            treasury = self.state.accounts.get(SIM_TREASURY_ADDR)
            if treasury:
                if asset == "wrt" and treasury.wrt_balance >= amount:
                    self._queue_tx(tx, "mint", f"§4.1 WRT {amount} → {target.address[:14]}…")
                elif asset == "lzn" and treasury.lzn_balance >= amount:
                    self._queue_tx(tx, "mint", f"§4.1 LZN {amount} → {target.address[:14]}…")
                elif asset == "ant" and treasury.ant_balance >= amount:
                    self._queue_tx(tx, "mint", f"§4.1 ANT {amount} → {target.role.value} {target.address[:12]}…")
            return

        elif action == "transfer":
            valid_senders = [acc for acc in accounts if acc.wrt_balance > 0.1]
            if not valid_senders:
                return
            # Валидаторы экономят WRT: реже и мельче переводы; остальные — прежняя нагрузка
            val_senders = [a for a in valid_senders if a.role == Role.VALIDATOR]
            non_val = [a for a in valid_senders if a.role != Role.VALIDATOR]
            if non_val and (not val_senders or random.random() < 0.72):
                sender = random.choice(non_val)
            else:
                sender = random.choice(val_senders if val_senders else valid_senders)
            
            possible_receivers = [acc for acc in accounts if acc.address != sender.address]
            if not possible_receivers:
                return
            receiver = random.choice(possible_receivers)

            if sender.role == Role.VALIDATOR:
                frac_hi = 0.05
                cap = 25.0
            else:
                frac_hi = 0.2
                cap = sender.wrt_balance
            amount = round(random.uniform(0.1, min(sender.wrt_balance * frac_hi, cap)), 2)
            if amount < 0.1:
                amount = min(sender.wrt_balance, 0.11)
                
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.TRANSFER,
                sender=sender.address,
                receiver=receiver.address,
                amount=amount,
                timestamp=time.time()
            )
            self._queue_tx(tx, "transfer", f"§4.1 WRT {amount}: {sender.address[:10]}… → {receiver.address[:10]}…")
            return

        elif action == "cancel_order":
            bot_orders = [
                o for o in self.state.orders.values() if is_bot_created_address(o.owner)
            ]
            if not bot_orders:
                return
            order_to_cancel = random.choice(bot_orders)
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.CANCEL_ORDER,
                sender=order_to_cancel.owner,
                order_id=order_to_cancel.id,
                timestamp=time.time()
            )
            self._queue_tx(tx, "cancel_order", f"§5.2 отмена ордера {order_to_cancel.id[:10]}…")
            return

        elif action == "trade":
            validators = [acc for acc in accounts if acc.role == Role.VALIDATOR]
            providers = [acc for acc in accounts if acc.role == Role.PROVIDER]
            base_price = self.state.last_price if self.state.last_price > 0 else 10.0
            tension = _epoch_tension_for_provider(self.state.current_height)

            bids_book = [
                o
                for o in self.state.orders.values()
                if o.order_type == OrderType.BUY and is_bot_created_address(o.owner)
            ]
            asks_book = [
                o
                for o in self.state.orders.values()
                if o.order_type == OrderType.SELL and is_bot_created_address(o.owner)
            ]
            best_bid = max((o.price for o in bids_book), default=None)
            best_ask = min((o.price for o in asks_book), default=None)
            # Иначе накапливаются односторонние лимиты (bid << ask по спреду бота) — график без новых тиков.
            cross_book = random.random() < 0.28

            # Оба типа ордеров полезны для книги; чуть чаще SELL у поставщиков при высокой «угрозе» wipe
            sell_bias = 0.5 + 0.35 * tension
            order_type = (
                OrderType.SELL if random.random() < sell_bias else OrderType.BUY
            )

            if order_type == OrderType.BUY:
                if not validators:
                    return
                trader = random.choice(validators)
                anchor = _validator_bid_anchor_mult()
                # Цель ~10% за неделю: базовый коридор вокруг last/(1+r); корректировка по факту последнего блока
                dw, da = _wallet_delta_last_block(self.state, trader.address)
                bid_lo = max(0.70, anchor - 0.06)
                bid_hi = min(0.93, anchor + 0.04)
                if da > 1e-9 and dw < -1e-9 and base_price > 1e-12:
                    paid_per_ant = (-dw) / da
                    stress = paid_per_ant / base_price - 1.0
                    stress = max(-0.28, min(0.28, stress))
                    center = anchor - 0.14 * max(0.0, stress) + 0.05 * max(0.0, -stress)
                    bid_lo = max(0.70, center - 0.07)
                    bid_hi = min(0.92, center + 0.05)
                elif dw > 1e-9 and da <= 1e-9:
                    # Приток WRT (награда/комиссии) — чуть шире, но не ломаем недельный порог маржи
                    bid_lo = max(0.76, anchor - 0.035)
                    bid_hi = min(0.91, anchor + 0.025)
                bid_mult = random.uniform(bid_lo, bid_hi)
                price = round(max(0.01, base_price * bid_mult), 2)
                # Пересечение спреда только если ask всё ещё укладывается в целевую маржу ~10%
                hard_cap = base_price * min(0.94, anchor + 0.025)
                if cross_book and best_ask is not None and best_ask <= hard_cap + 1e-9:
                    price = round(max(0.01, best_ask * random.uniform(0.97, 1.0)), 2)
                elif cross_book and best_ask is not None:
                    price = round(max(0.01, base_price * bid_mult), 2)
                # Не разом выкидывать WRT: скромный notional
                max_wrt = min(trader.wrt_balance * 0.12, trader.wrt_balance)
                if max_wrt < price * 0.5:
                    return
                shares_amount = round(random.uniform(0.5, max(0.51, max_wrt / price)), 2)
                if trader.wrt_balance < price * shares_amount:
                    shares_amount = round(trader.wrt_balance / price * 0.95, 2)
                if shares_amount < 0.01:
                    return
            else:
                if not providers:
                    return
                trader = random.choice(providers)
                liquid_ant = float(trader.ant_balance)
                escrow_sell_ant = _provider_open_sell_ant_pending(self.state, trader.address)
                denom = max(1e-9, liquid_ant + escrow_sell_ant)
                escrow_pressure = min(1.0, escrow_sell_ant / denom)
                # К границе эпохи + «висит» много ANT в лимитах → чаще и крупнее рыночная продажа в bid
                urgency = max(tension, 0.55 * escrow_pressure)
                p_market = 0.0
                if bids_book and liquid_ant >= 0.02:
                    # Ближе к §5.5 и при «зависшем» эскроу SELL — заметно чаще MARKET SELL
                    p_market = min(
                        0.93,
                        0.04 + 0.86 * (tension**1.12) + 0.32 * escrow_pressure,
                    )
                if bids_book and liquid_ant >= 0.02 and random.random() < p_market:
                    frac = min(0.98, 0.18 + 0.72 * urgency)
                    shares_amount = round(max(0.02, liquid_ant * frac), 2)
                    if shares_amount > liquid_ant:
                        shares_amount = round(liquid_ant * 0.98, 2)
                    if shares_amount >= 0.02:
                        tx = Transaction(
                            tx_hash=uuid.uuid4().hex,
                            tx_type=TransactionType.CREATE_ORDER,
                            sender=trader.address,
                            order_type=OrderType.SELL,
                            price=0.0,
                            amount=shares_amount,
                            market=True,
                            timestamp=time.time(),
                        )
                        self._queue_tx(
                            tx,
                            "create_order",
                            f"§5.2 MARKET SELL {shares_amount} ANT (urgency={urgency:.2f} tension={tension:.2f} escrow_SELL={escrow_sell_ant:.2f})",
                        )
                    return

                # Поставщик: max выгода в WRT с учётом риска потери ANT на границе эпохи §5.5
                # tension↓ → дороже продаём; tension↑ → снижаем ask и увеличиваем объём, чтобы успеть в WRT
                ask_lo, ask_hi = 0.82 + 0.08 * (1.0 - tension), 1.28 - 0.22 * tension
                dw, da = _wallet_delta_last_block(self.state, trader.address)
                if da < -1e-9 and dw > 1e-9:
                    unit_wrt = dw / (-da)
                    if base_price > 1e-12:
                        edge = unit_wrt / base_price - 1.0
                        edge = max(-0.4, min(0.4, edge))
                        skew = 1.0 + 0.14 * edge
                        ask_lo *= skew
                        ask_hi *= skew
                elif da < -1e-9 and dw <= 1e-9:
                    # ANT ушли (эскроу/сжигание) без WRT — сильнее дисконт к ask
                    dump = 0.88 - 0.07 * tension
                    ask_lo *= dump
                    ask_hi *= dump
                elif dw > 1e-9 and da >= -1e-9:
                    # Чистый WRT без отдачи ANT — можно поднять ask
                    ask_lo *= 1.04
                    ask_hi *= 1.05
                ask_lo = max(0.5, ask_lo)
                ask_hi = max(ask_lo + 0.02, ask_hi)
                price = round(base_price * random.uniform(ask_lo, ask_hi), 2)
                if cross_book and best_bid is not None:
                    price = round(max(0.01, best_bid * random.uniform(0.96, 1.0)), 2)
                min_sh, max_sh = 1.0, 22.0
                size_boost = tension * 18.0
                cap_ant = min(float(trader.ant_balance), max_sh + size_boost)
                shares_amount = round(random.uniform(min_sh, max(min_sh + 0.01, cap_ant)), 2)
                if shares_amount > trader.ant_balance:
                    shares_amount = round(trader.ant_balance * 0.98, 2)
                if shares_amount < 0.01:
                    return

            if order_type == OrderType.BUY and trader.wrt_balance >= (price * shares_amount):
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.CREATE_ORDER,
                    sender=trader.address,
                    order_type=order_type,
                    price=price,
                    amount=shares_amount,
                    timestamp=time.time()
                )
                self._queue_tx(
                    tx,
                    "create_order",
                    f"§5.2 BUY {shares_amount} ANT @ {price} WRT (цель ~{VALIDATOR_TARGET_WEEKLY_WRT_RETURN:.0%}/нед, bid≈{bid_mult:.2f}× last≈{anchor:.3f}/(1+r), Δблок WRT={dw:+.4f} ANT={da:+.4f})",
                )
            elif order_type == OrderType.SELL and trader.role == Role.PROVIDER and trader.ant_balance >= shares_amount:
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.CREATE_ORDER,
                    sender=trader.address,
                    order_type=order_type,
                    price=price,
                    amount=shares_amount,
                    timestamp=time.time()
                )
                self._queue_tx(
                    tx,
                    "create_order",
                    f"§5.2 SELL {shares_amount} ANT @ {price} WRT (tension={tension:.2f}, Δблок WRT={dw:+.4f} ANT={da:+.4f})",
                )
            return

        elif action == "canon_probe_mint_ant_citizen":
            citizens = [a for a in accounts if a.role == Role.CITIZEN]
            if not citizens:
                addr = f"{BOT_ADDRESS_PREFIX}{uuid.uuid4().hex[:8]}"
                self.state.create_account(addr)
                citizens = [self.state.accounts[addr]]
            victim = random.choice(citizens)
            amt = round(random.uniform(1, 5), 2)
            treasury = self.state.accounts.get(SIM_TREASURY_ADDR)
            if treasury and treasury.ant_balance >= amt:
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.MINT,
                    sender=SIM_TREASURY_ADDR,
                    receiver=victim.address,
                    amount=amt,
                    asset_type="ant",
                    timestamp=time.time(),
                )
                self._queue_tx(
                    tx,
                    "canon_probe",
                    f"Тест канона: mint ANT Гражданину {victim.address[:14]}… — в блоке должно быть отклонение §4.2",
                )
            return

        elif action == "canon_probe_transfer_ant":
            vals = [a for a in accounts if a.role == Role.VALIDATOR and a.ant_balance > 1]
            if not vals:
                return
            v = random.choice(vals)
            recv_pool = [a for a in accounts if a.address != v.address]
            if not recv_pool:
                return
            recv = random.choice(recv_pool)
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.TRANSFER,
                sender=v.address,
                receiver=recv.address,
                amount=1.0,
                asset_type="ant",
                timestamp=time.time(),
            )
            self._queue_tx(
                tx,
                "canon_probe",
                "Тест канона: прямой перевод ANT — в блоке отклонение §4.1 (только внутренний рынок)",
            )
            return
