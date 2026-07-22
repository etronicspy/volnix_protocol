import asyncio
import random
import time
import uuid
from typing import Optional

from core.canon_audit import log_bot_queue, log_wallet_rejection
from core.models import OrderType, Role, Transaction, TransactionType
from core.state import (
    BLOCKS_PER_EPOCH,
    SIM_TREASURY_ADDR,
    StateManager,
    eligible_for_provider_role,
    eligible_for_validator_role,
)
from core.wallet_validate import validate_and_build_tx, validate_treasury_mint

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
        # Настройки "наблюдаемости": генерировать канон-пробы (ожидаемые отклонения в блоке)
        self.enable_probes = True
        # Доля probе-tx среди всех действий бота (0..1)
        self.probe_ratio = 0.15
        # Тогглы конкретных проб
        self.probe_transfer_ant = True
        self.probe_mint_ant_citizen = True
        self.probe_wrong_role_declare = True
        self.probe_wrong_role_activate_lzn = True
        self.probe_wrong_role_order = True
        self.probe_cancel_not_owned = True

    def set_intensity(self, tx_per_second: float):
        self.tx_per_second = max(0.1, min(tx_per_second, 100.0))

    def set_probe_settings(
        self,
        *,
        enable=None,
        ratio=None,
        transfer_ant=None,
        mint_ant_citizen=None,
        wrong_role_declare=None,
        wrong_role_activate_lzn=None,
        wrong_role_order=None,
        cancel_not_owned=None,
    ) -> None:
        if enable is not None:
            self.enable_probes = bool(enable)
        if ratio is not None:
            try:
                r = float(ratio)
            except (TypeError, ValueError):
                r = self.probe_ratio
            self.probe_ratio = max(0.0, min(1.0, r))
        if transfer_ant is not None:
            self.probe_transfer_ant = bool(transfer_ant)
        if mint_ant_citizen is not None:
            self.probe_mint_ant_citizen = bool(mint_ant_citizen)
        if wrong_role_declare is not None:
            self.probe_wrong_role_declare = bool(wrong_role_declare)
        if wrong_role_activate_lzn is not None:
            self.probe_wrong_role_activate_lzn = bool(wrong_role_activate_lzn)
        if wrong_role_order is not None:
            self.probe_wrong_role_order = bool(wrong_role_order)
        if cancel_not_owned is not None:
            self.probe_cancel_not_owned = bool(cancel_not_owned)

    def _mempool_push(self, tx: Transaction, sender: str = "") -> None:
        """Подача tx: NetworkSim (если attached) или общий мемпул."""
        net = getattr(self.state, "network", None)
        if net is not None:
            addr = sender or getattr(tx, "sender", "") or ""
            try:
                if addr:
                    net.submit_from_addr(addr, tx)
                else:
                    net.submit_to("node_0", tx)
                return
            except Exception:
                pass
        self.state.mempool.append(tx)

    def _queue_tx(self, tx: Transaction, action: str, detail: str) -> None:
        """Прямая постановка готовой Transaction в мемпул (legacy: для рукотворных tx бота).

        Все новые сценарии должны идти через _submit_op / _submit_mint, которые проходят через
        validate_and_build_tx (единые правила admission с кошельком и Sim Operator).
        """
        self._mempool_push(tx, getattr(tx, "sender", "") or "")
        log_bot_queue(self.state, action, detail, tx.tx_hash)

    def _submit_op(
        self,
        *,
        op: str,
        address: str,
        action: str,
        detail: str,
        **kwargs,
    ) -> Optional[Transaction]:
        """Сценарии бота → validate_and_build_tx (как кошелёк / Sim Operator).

        Логирует rejection в канон-аудит, если правила admission отвергают tx до мемпула.
        """
        ok, msg, tx = validate_and_build_tx(self.state, op, address, **kwargs)
        if not ok or tx is None:
            log_wallet_rejection(self.state, op, f"bot: {msg}", address)
            return None
        self._mempool_push(tx, address)
        log_bot_queue(self.state, action, detail, tx.tx_hash)
        return tx

    def _submit_mint(
        self,
        *,
        receiver: str,
        amount: float,
        asset: str,
        action: str,
        detail: str,
    ) -> Optional[Transaction]:
        """Бот-минт из казны → validate_treasury_mint (как Sim Operator)."""
        ok, msg, tx = validate_treasury_mint(self.state, receiver, amount, asset)
        if not ok or tx is None:
            log_wallet_rejection(self.state, "mint", f"bot: {msg}", receiver)
            return None
        # Sim Operator → treasury node (node_0)
        self._mempool_push(tx, getattr(tx, "sender", "") or "")
        log_bot_queue(self.state, action, detail, tx.tx_hash)
        return tx

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
            self.state.create_account(new_addr)
            amount = round(random.uniform(50, 200), 2)
            self._submit_mint(
                receiver=new_addr,
                amount=amount,
                asset="wrt",
                action="mint_wrt_new_bot",
                detail=f"WRT казна → новый адрес {new_addr} (разгон книги)",
            )
            return

        # 2. Сценарии нагрузки + настраиваемые проверки канона (ожидаемые отклонения в блоке)
        probes: list[str] = []
        if self.enable_probes:
            if self.probe_mint_ant_citizen:
                probes.append("canon_probe_mint_ant_citizen")
            if self.probe_transfer_ant:
                probes.append("canon_probe_transfer_ant")
            if self.probe_wrong_role_declare:
                probes.append("canon_probe_wrong_role_declare")
            if self.probe_wrong_role_activate_lzn:
                probes.append("canon_probe_wrong_role_activate_lzn")
            if self.probe_wrong_role_order:
                probes.append("canon_probe_wrong_role_order")
            if self.probe_cancel_not_owned:
                probes.append("canon_probe_cancel_not_owned")

        base_actions = [
            "transfer",
            "trade",
            "cancel_order",
            "mint",
            "set_role",
            "declare",
            "zkp_verify",
        ]

        if probes and random.random() < self.probe_ratio:
            action = random.choice(probes)
        else:
            action = random.choices(
                base_actions,
                weights=[0.24, 0.19, 0.09, 0.12, 0.07, 0.09, 0.06],
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
                        self._submit_op(
                            op="declare",
                            address=val.address,
                            action="declare",
                            detail=f"§5.4 b={b} s={s} (валидатор {val.address[:14]}…)",
                            burn_b=b,
                            stake_s=s,
                        )
            return

        elif action == "zkp_verify":
            candidates = [a for a in accounts if not a.zkp_verified]
            if candidates:
                t = random.choice(candidates)
                self._submit_op(
                    op="verify_zkp",
                    address=t.address,
                    action="zkp_verify",
                    detail=f"§3.1 симуляция → {t.address[:16]}…",
                )
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
            self._submit_op(
                op="set_role",
                address=target.address,
                action="set_role",
                detail=f"§4.2 → {new_role.value} для {target.address[:14]}…",
                role=new_role,
            )
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

            self._submit_mint(
                receiver=target.address,
                amount=amount,
                asset=asset,
                action="mint",
                detail=f"§4.1 {asset.upper()} {amount} → {target.role.value} {target.address[:14]}…",
            )
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

            self._submit_op(
                op="transfer",
                address=sender.address,
                action="transfer",
                detail=f"§4.1 WRT {amount}: {sender.address[:10]}… → {receiver.address[:10]}…",
                to_address=receiver.address,
                amount=amount,
                asset="wrt",
            )
            return

        elif action == "cancel_order":
            bot_orders = [
                o for o in self.state.orders.values() if is_bot_created_address(o.owner)
            ]
            if not bot_orders:
                return
            order_to_cancel = random.choice(bot_orders)
            self._submit_op(
                op="cancel_order",
                address=order_to_cancel.owner,
                action="cancel_order",
                detail=f"§5.2 отмена ордера {order_to_cancel.id[:10]}…",
                order_id=order_to_cancel.id,
            )
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
                        self._submit_op(
                            op="create_order",
                            address=trader.address,
                            action="create_order",
                            detail=(
                                f"§5.2 MARKET SELL {shares_amount} ANT "
                                f"(urgency={urgency:.2f} tension={tension:.2f} escrow_SELL={escrow_sell_ant:.2f})"
                            ),
                            side="sell",
                            amount=shares_amount,
                            market=True,
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
                self._submit_op(
                    op="create_order",
                    address=trader.address,
                    action="create_order",
                    detail=(
                        f"§5.2 BUY {shares_amount} ANT @ {price} WRT "
                        f"(цель ~{VALIDATOR_TARGET_WEEKLY_WRT_RETURN:.0%}/нед, "
                        f"bid≈{bid_mult:.2f}× last≈{anchor:.3f}/(1+r), Δблок WRT={dw:+.4f} ANT={da:+.4f})"
                    ),
                    side="buy",
                    price=price,
                    amount=shares_amount,
                )
            elif order_type == OrderType.SELL and trader.role == Role.PROVIDER and trader.ant_balance >= shares_amount:
                self._submit_op(
                    op="create_order",
                    address=trader.address,
                    action="create_order",
                    detail=(
                        f"§5.2 SELL {shares_amount} ANT @ {price} WRT "
                        f"(tension={tension:.2f}, Δблок WRT={dw:+.4f} ANT={da:+.4f})"
                    ),
                    side="sell",
                    price=price,
                    amount=shares_amount,
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

        elif action == "canon_probe_wrong_role_declare":
            # Подать declare не-валидатором — должно отклониться при финализации §5.4
            pool = [a for a in accounts if a.role != Role.VALIDATOR and a.zkp_verified]
            if not pool:
                pool = [a for a in accounts if a.role != Role.VALIDATOR]
            if not pool:
                return
            t = random.choice(pool)
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.DECLARE_PARTICIPATION,
                sender=t.address,
                amount=1.0,
                stake_amount=0.0,
                asset_type="ant",
                timestamp=time.time(),
            )
            self._queue_tx(
                tx,
                "canon_probe",
                "Тест канона: declare от не-валидатора — в блоке отклонение §5.4/§4.2",
            )
            return

        elif action == "canon_probe_wrong_role_activate_lzn":
            # Подать activate_lzn от не-валидатора — должно отклониться в DeliverTx
            pool = [a for a in accounts if a.role != Role.VALIDATOR and a.lzn_balance > 0.01]
            if not pool:
                pool = [a for a in accounts if a.role != Role.VALIDATOR]
            if not pool:
                return
            t = random.choice(pool)
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.ACTIVATE_LZN,
                sender=t.address,
                amount=1.0,
                asset_type="lzn",
                timestamp=time.time(),
            )
            self._queue_tx(
                tx,
                "canon_probe",
                "Тест канона: activate_lzn от не-валидатора — в блоке отклонение §4.2",
            )
            return

        elif action == "canon_probe_wrong_role_order":
            # Поставить ордер от Гражданина — должно отклониться в DeliverTx (рынок §5.2)
            citizens = [a for a in accounts if a.role == Role.CITIZEN]
            if not citizens:
                return
            c = random.choice(citizens)
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.CREATE_ORDER,
                sender=c.address,
                order_type=random.choice([OrderType.BUY, OrderType.SELL]),
                price=10.0,
                amount=1.0,
                timestamp=time.time(),
            )
            self._queue_tx(
                tx,
                "canon_probe",
                "Тест канона: create_order от Гражданина — в блоке отклонение §4.2/§5.2",
            )
            return

        elif action == "canon_probe_cancel_not_owned":
            # Попробовать отменить ордер чужим адресом — должно отклониться в DeliverTx
            any_orders = list(self.state.orders.values())
            if not any_orders:
                return
            o = random.choice(any_orders)
            impostors = [a for a in accounts if a.address != o.owner]
            if not impostors:
                return
            imp = random.choice(impostors)
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.CANCEL_ORDER,
                sender=imp.address,
                order_id=o.id,
                timestamp=time.time(),
            )
            self._queue_tx(
                tx,
                "canon_probe",
                "Тест канона: cancel_order не владельцем — в блоке отклонение §5.2",
            )
            return
