import asyncio
import random
import time
import uuid
from typing import Optional

from core.canon_audit import log_bot_queue, log_wallet_rejection
from core.models import OrderType, Role, Transaction, TransactionType
from core.profit_strategy import ProfitStrategy, top_switch_candidates
from core.state import (
    BLOCKS_PER_EPOCH,
    CANONICAL_BLOCK_INTERVAL_SEC,
    LZN_MAX_FROZEN_PER_ADDRESS,
    SIM_TREASURY_ADDR,
    StateManager,
    account_total_lzn,
    lzn_mint_headroom,
)

# Плотность трафика привязана к сим. времени: intensity = действий / сим. секунду.
# При intensity=1 → ~60 действий на блок (CANONICAL_BLOCK_INTERVAL_SEC).
# На высоких sim_speed реальный CPU ограничиваем, иначе 60×блоков/с.
MAX_BOT_ACTIONS_PER_BLOCK = 60
MAX_BOT_ACTIONS_PER_REAL_SEC = 250
MAX_BOT_ACTION_CARRY = MAX_BOT_ACTIONS_PER_BLOCK * 2
from core.wallet_validate import validate_and_build_tx, validate_treasury_mint

# Бот создаёт только адреса с префиксом bot_<hex>; не трогает genesis / казну / чужие кошельки.
BOT_ADDRESS_PREFIX = "bot_"

# Разрешение цены на рынке ANT (см. _round_ant_price).
ANT_PRICE_DECIMALS = 6
MIN_ANT_PRICE = 10.0 ** -ANT_PRICE_DECIMALS


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


def _round_ant_price(price: float) -> float:
    """Цена ANT живёт около 0.02 WRT — двух знаков не хватает даже на тик."""
    return round(max(MIN_ANT_PRICE, float(price)), ANT_PRICE_DECIMALS)


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


# Смена роли — на середине эпохи ANT и на границе §5.5 (два раза за цикл).
ROLE_REEVAL_INTERVAL_BLOCKS = BLOCKS_PER_EPOCH // 2  # 5040


class BotEngine:
    def __init__(self, state_manager: StateManager):
        self.state = state_manager
        self.is_running = False
        # intensity: действий бота на одну сим. секунду (не wall-clock).
        self.tx_per_second = 1.0
        self.profit_strategy = ProfitStrategy()
        self._last_reeval_period = 0
        self._action_carry = 0.0
        self._real_sec_window_start = 0.0
        self._actions_this_real_sec = 0
        self.enable_probes = True
        self.probe_ratio = 0.15
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
        """Подача tx через StateManager.submit_tx (replace-by-sender для declare)."""
        self.state.submit_tx(tx)

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

    def step_once(self) -> int:
        """Один тик перед блоком: трафик ∝ intensity × сим. длительность блока.

        Вызывается из `engine.pre_block_hook`, чтобы рынок/переводы шли в ногу
        с `sim_speed`, а не с wall-clock sleep(1/intensity).
        """
        if not self.is_running:
            return 0

        self._action_carry += float(self.tx_per_second) * CANONICAL_BLOCK_INTERVAL_SEC
        if self._action_carry > MAX_BOT_ACTION_CARRY:
            self._action_carry = float(MAX_BOT_ACTION_CARRY)

        n = int(self._action_carry)
        if n <= 0:
            return 0
        n = min(n, MAX_BOT_ACTIONS_PER_BLOCK)

        now = time.monotonic()
        if self._real_sec_window_start <= 0.0 or now - self._real_sec_window_start >= 1.0:
            self._real_sec_window_start = now
            self._actions_this_real_sec = 0
        budget = MAX_BOT_ACTIONS_PER_REAL_SEC - self._actions_this_real_sec
        if budget <= 0:
            return 0
        n = min(n, budget)

        self._action_carry -= float(n)
        done = 0
        for _ in range(n):
            try:
                self.generate_traffic()
                done += 1
            except Exception as e:
                print(f"Bot error: {e}")
        self._actions_this_real_sec += done
        return done

    async def start(self):
        """Keepalive: флажок is_running; трафик только из pre_block_hook.step_once."""
        self.is_running = True
        print(
            f"Bot Engine started. Intensity: {self.tx_per_second} actions/sim-sec "
            f"(~{self.tx_per_second * CANONICAL_BLOCK_INTERVAL_SEC:.0f}/block, via pre_block)"
        )
        while self.is_running:
            await asyncio.sleep(1.0)

    def stop(self):
        self.is_running = False
        self._action_carry = 0.0
        print("Bot Engine stopped.")

    def _periodic_role_reeval(self, accounts: list) -> None:
        """Смена роли раз в полэпохи ANT — на mid-epoch и на границе §5.5.

        Сравниваем номер полуэпохи, а не точное `h % 5040 == 0`: бот ходит по
        реальному времени, и на высоких скоростях (сотни блоков за тик) он почти
        никогда не наблюдает высоту, кратную интервалу, — переоценка не срабатывала
        вообще и боты навсегда оставались Гражданами.
        """
        h = int(self.state.current_height)
        if h <= 0:
            return
        period = h // ROLE_REEVAL_INTERVAL_BLOCKS
        if period <= self._last_reeval_period:
            return
        self._last_reeval_period = period

        switched = 0
        for acc, new_role, delta in top_switch_candidates(
            self.profit_strategy, accounts, self.state, limit=len(accounts)
        ):
            roi_info = self.profit_strategy.get_cached_roi(acc.address) or {}
            cur_roi = roi_info.get(acc.role.value)
            new_roi = roi_info.get(new_role.value)
            detail = (
                f"reeval §4.2 → {new_role.value} "
                f"(cur {acc.role.value} net={cur_roi.net:.1f}, best {new_role.value} net={new_roi.net:.1f}, Δ={delta:.1f})"
                if cur_roi and new_roi
                else f"reeval §4.2 → {new_role.value}"
            )
            tx = self._submit_op(
                op="set_role",
                address=acc.address,
                action="set_role_reeval",
                detail=detail,
                role=new_role,
            )
            if tx:
                switched += 1
        if switched:
            print(f"[ProfitStrategy] height {h}: {switched} bots switched role")

    def _trade_lzn_equipment(self, accounts: list) -> None:
        """§5.2 5.0-sim: покупка LZN («оборудования») на внутреннем рынке.

        Поставщик выставляет/доливает SELL LZN; валидатор — market BUY за WRT.
        Прямые MsgSend LZN запрещены.
        """
        sellers = [
            a
            for a in self.state.accounts.values()
            if a.role == Role.PROVIDER
            and a.address != SIM_TREASURY_ADDR
            and a.lzn_balance > 1.0
        ]
        buyers = [
            a
            for a in accounts
            if a.role == Role.VALIDATOR
            and a.zkp_verified
            and a.wrt_balance > 1.0
            and account_total_lzn(a) < LZN_MAX_FROZEN_PER_ADDRESS
        ]
        if not sellers or not buyers:
            return
        seller = max(sellers, key=lambda a: a.lzn_balance)
        buyer = random.choice([b for b in buyers if b.address != seller.address] or buyers)

        headroom = LZN_MAX_FROZEN_PER_ADDRESS - account_total_lzn(buyer)
        lot = round(min(seller.lzn_balance, headroom, random.uniform(5.0, 60.0)), 4)
        if lot < 1.0:
            return
        unit = max(
            0.01,
            float(getattr(self.state, "last_lzn_price", 0.0) or 0.0)
            or float(self.state.last_price)
            or 1.0,
        )
        ask_price = round(unit * random.uniform(0.95, 1.1), 4)
        self._submit_op(
            op="create_order",
            address=seller.address,
            action="lzn_market",
            detail=f"§5.2 SELL {lot} LZN @ {ask_price} WRT",
            side="sell",
            price=ask_price,
            amount=lot,
            asset="lzn",
        )
        max_wrt = round(min(buyer.wrt_balance * 0.5, lot * ask_price * 1.05), 4)
        if max_wrt < 0.01:
            return
        self._submit_op(
            op="create_order",
            address=buyer.address,
            action="lzn_market",
            detail=f"§5.2 market BUY до {lot} LZN (max WRT {max_wrt})",
            side="buy",
            amount=lot,
            market=True,
            max_wrt=max_wrt,
            asset="lzn",
        )

    def _try_profit_prep(self, acc) -> bool:
        """Unlock a more profitable role: ZKP, then activate LZN once validator."""
        desired = self.profit_strategy.desired_role(acc.address, acc, self.state)
        if desired == acc.role:
            if (
                acc.role == Role.VALIDATOR
                and acc.lzn_balance > 0.01
                and acc.lzn_frozen_mining <= 1e-12
            ):
                amt = round(acc.lzn_balance, 4)
                self._submit_op(
                    op="activate_lzn",
                    address=acc.address,
                    action="activate_lzn",
                    detail=f"profit-prep activate_lzn {amt} (валидатор {acc.address[:14]}…)",
                    amount=amt,
                )
                return True
            return False
        if not acc.zkp_verified and desired in (Role.PROVIDER, Role.VALIDATOR):
            self._submit_op(
                op="verify_zkp",
                address=acc.address,
                action="zkp_verify",
                detail=f"profit-prep ZKP → цель {desired.value} для {acc.address[:16]}…",
            )
            return True
        if (
            acc.role == Role.VALIDATOR
            and desired == Role.VALIDATOR
            and acc.lzn_balance > 0.01
        ):
            amt = round(min(acc.lzn_balance, max(0.01, acc.lzn_balance)), 4)
            self._submit_op(
                op="activate_lzn",
                address=acc.address,
                action="activate_lzn",
                detail=f"profit-prep activate_lzn {amt} (валидатор {acc.address[:14]}…)",
                amount=amt,
            )
            return True
        return False

    def generate_traffic(self):
        all_accounts = list(self.state.accounts.values())
        accounts = [a for a in all_accounts if is_bot_created_address(a.address)]
        bot_count = len(accounts)

        self._periodic_role_reeval(accounts)

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
            "lzn_market",
        ]

        if probes and random.random() < self.probe_ratio:
            action = random.choice(probes)
        else:
            action = random.choices(
                base_actions,
                weights=[0.24, 0.19, 0.09, 0.12, 0.07, 0.09, 0.06, 0.08],
            )[0]

        if action == "lzn_market":
            self._trade_lzn_equipment(accounts)
            return

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
            # Саму роль меняем только на half-epoch reeval; здесь — ZKP / activate_lzn.
            pool = list(accounts)
            if pool:
                self._try_profit_prep(random.choice(pool))
            return

        elif action == "mint":
            if not accounts:
                return
            target = random.choice(accounts)
            possible_assets = ["wrt"]
            # §4.1: LZN — фиксированная эмиссия; минтим только в пределах остатка.
            headroom = lzn_mint_headroom(self.state.accounts)
            if headroom >= 10.0:
                possible_assets.append("lzn")
            if target.role in (Role.PROVIDER, Role.VALIDATOR):
                possible_assets.append("ant")

            asset = random.choice(possible_assets)
            amount = round(random.uniform(10, 100), 2)
            if asset == "lzn":
                amount = round(min(amount, headroom), 2)

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
                price = _round_ant_price(max(MIN_ANT_PRICE, base_price * bid_mult))
                # Пересечение спреда только если ask всё ещё укладывается в целевую маржу ~10%
                hard_cap = base_price * min(0.94, anchor + 0.025)
                if cross_book and best_ask is not None and best_ask <= hard_cap + 1e-9:
                    price = _round_ant_price(max(MIN_ANT_PRICE, best_ask * random.uniform(0.97, 1.0)))
                elif cross_book and best_ask is not None:
                    price = _round_ant_price(max(MIN_ANT_PRICE, base_price * bid_mult))
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
                # Округление до 2 знаков схлопывало всю книгу: равновесная цена ANT
                # порядка 0.02 WRT, и любой ask превращался в 0.02/0.03 — то есть
                # выше безубыточности майнера, из-за чего заявки не исполнялись.
                price = _round_ant_price(base_price * random.uniform(ask_lo, ask_hi))
                if cross_book and best_bid is not None:
                    price = _round_ant_price(max(MIN_ANT_PRICE, best_bid * random.uniform(0.96, 1.0)))
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
