import asyncio
import time
from fastapi import WebSocket
from typing import Dict, List, Optional, Tuple

class ConnectionManager:
    def __init__(self):
        self.active_connections: List[WebSocket] = []

    async def connect(self, websocket: WebSocket):
        await websocket.accept()
        self.active_connections.append(websocket)

    def disconnect(self, websocket: WebSocket):
        self.active_connections.remove(websocket)

    async def broadcast(self, message: dict):
        for connection in self.active_connections:
            try:
                await connection.send_json(message)
            except:
                pass

from core.models import Role, Transaction, TransactionType, Order, OrderType
from core.canon_audit import audit_block_ledger, log_engine_skip
from core.state import (
    BLOCKS_PER_EPOCH,
    CANONICAL_BLOCK_INTERVAL_SEC,
    LZN_MAX_FROZEN_PER_ADDRESS,
    SIM_TREASURY_ADDR,
    eligible_for_provider_role,
    eligible_for_validator_role,
)
import uuid

COEFF_MIN = 0.75
COEFF_MAX = 1.5

# §5.4 / §7.2: глобальный лимит сжигания и потолок подписантов (genesis-эталон)
BURN_CAP_LAMBDA = 1.0 / 3.0
MAX_ACTIVE_VALIDATORS_K = 150
# §5.2/§5.4: условный пул комиссий блока (симуляция); дележ F·(b_i/B), B=Σb_i
SIM_FEE_WRT_PER_LEDGER_TX = 0.02
# Допуск: Σb_i должна попасть в [λ·L_total − tol, λ·L_total + tol] для §5.1 и дележа комиссий
BURN_TARGET_ATOL = 0.5

class SimulationEngine:
    def __init__(self, state_manager):
        self.state = state_manager
        # Дефолт = эталон цепи (60 с); после load_state startup выставит из state.json
        self.block_time = float(
            getattr(state_manager, "sim_block_interval_sec", CANONICAL_BLOCK_INTERVAL_SEC)
        )
        self.is_running = False
        self.ws_manager = ConnectionManager()
        # Сброс состояния и produce_block не должны пересекаться — иначе старый блок перезаписывает сброс.
        self._state_lock = asyncio.Lock()

    def set_block_time(self, time_sec: float):
        self.block_time = max(0.1, min(time_sec, 300.0))
        self.state.sim_block_interval_sec = self.block_time

    def _release_orders_for_owner(self, owner_addr: str, acc):
        """Вернуть эскроу ордеров владельцу и удалить ордера из книги."""
        for oid, order in list(self.state.orders.items()):
            if order.owner != owner_addr:
                continue
            if order.order_type == OrderType.BUY:
                acc.wrt_balance += order.price * (order.amount - order.filled)
            else:
                acc.ant_balance += (order.amount - order.filled)
            del self.state.orders[oid]

    def _sanitize_ant_ineligible_roles(self, txs_in_block: list):
        """
        §4.1–4.2: ANT только у Поставщика и Валидатора; Гражданин (тип 1) — без ANT и без ордеров на рынке.
        Казначейство симуляции исключено.
        """
        bad_order_owners: set = set()
        for _oid, order in self.state.orders.items():
            owner = self.state.accounts.get(order.owner)
            if not owner or owner.address == SIM_TREASURY_ADDR:
                continue
            if owner.role == Role.CITIZEN:
                bad_order_owners.add(owner.address)
        for addr in bad_order_owners:
            acc = self.state.accounts.get(addr)
            if acc:
                self._release_orders_for_owner(addr, acc)
                txs_in_block.append({
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "protocol_order_cancel",
                    "receiver": addr,
                    "details": "Снятие ордеров: Гражданин (тип 1 §4.2) не участвует в рынке ANT",
                    "timestamp": time.time(),
                })
        for acc in self.state.accounts.values():
            if acc.address == SIM_TREASURY_ADDR:
                continue
            if acc.role != Role.CITIZEN:
                continue
            if acc.ant_balance > 1e-12:
                burnt = acc.ant_balance
                acc.ant_balance = 0.0
                txs_in_block.append({
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "protocol_ant_burn",
                    "receiver": acc.address,
                    "amount": burnt,
                    "asset_type": "ant",
                    "details": "§4.1: ANT сжёжен — тип кошелька Гражданин (§4.2 тип 1) не держит ANT",
                    "timestamp": time.time(),
                })

    async def start(self):
        self.is_running = True
        print(f"Simulation Engine started. Block time: {self.block_time}s")
        while self.is_running:
            await self.produce_block()
            await asyncio.sleep(self.block_time)

    def stop(self):
        self.is_running = False
        print("Simulation Engine stopped.")

    def _match_orders(self):
        # Simple matching engine
        bids = [o for o in self.state.orders.values() if o.order_type == OrderType.BUY]
        asks = [o for o in self.state.orders.values() if o.order_type == OrderType.SELL]
        
        bids.sort(key=lambda x: (-x.price, x.timestamp))
        asks.sort(key=lambda x: (x.price, x.timestamp))
        
        matched_txs = []
        
        while bids and asks:
            highest_bid = bids[0]
            lowest_ask = asks[0]
            
            if highest_bid.price >= lowest_ask.price:
                # Match found
                match_price = lowest_ask.price # price of the maker
                match_amount = min(highest_bid.amount - highest_bid.filled, lowest_ask.amount - lowest_ask.filled)
                
                # Execute trade
                buyer = self.state.accounts.get(highest_bid.owner)
                seller = self.state.accounts.get(lowest_ask.owner)
                
                total_cost = match_price * match_amount
                # BUY: WRT полностью зарезервированы при create_order (price * amount).
                # Исполнение: продавцу — match_price * qty; покупателю — скидка к лимиту (bid−maker)*qty.
                # SELL: ANT уже списаны с баланса при create_order — повторно не снимаем.
                if (
                    buyer
                    and seller
                    and buyer.role == Role.VALIDATOR
                    and seller.role == Role.PROVIDER
                ):
                    buyer.wrt_balance += (highest_bid.price - match_price) * match_amount
                    seller.wrt_balance += total_cost
                    buyer.ant_balance += match_amount
                    if seller.role == Role.PROVIDER:
                        self.state.epoch_ant_sold_volume += match_amount

                    highest_bid.filled += match_amount
                    lowest_ask.filled += match_amount
                    self.state.record_trade_price(match_price)

                    matched_txs.append(
                        {
                            "tx_hash": uuid.uuid4().hex,
                            "tx_type": "trade",
                            "buyer": buyer.address,
                            "seller": seller.address,
                            "price": match_price,
                            "amount": match_amount,
                            "timestamp": time.time(),
                        }
                    )
                else:
                    # Иначе вечный цикл: цена пересекается, но сделка не проходит (несогласованность книги).
                    break

                if highest_bid.filled >= highest_bid.amount:
                    bids.pop(0)
                    del self.state.orders[highest_bid.id]
                if lowest_ask.filled >= lowest_ask.amount:
                    asks.pop(0)
                    del self.state.orders[lowest_ask.id]
            else:
                break # No more matches possible
                
        return matched_txs

    def _push_trade_price(self, match_price: float) -> None:
        self.state.record_trade_price(match_price)

    def _execute_market_buy(self, tx: Transaction) -> List[dict]:
        """Покупка до amount ANT по лучшим ask, не более max_wrt WRT (списание с баланса по мере сделок)."""
        buyer = self.state.accounts.get(tx.sender)
        if not buyer or buyer.role != Role.VALIDATOR:
            return []
        target = float(tx.amount or 0)
        if target <= 1e-12:
            return []
        max_spend = float(tx.max_wrt) if tx.max_wrt is not None else buyer.wrt_balance
        max_spend = min(max_spend, buyer.wrt_balance)
        if max_spend <= 1e-12:
            return []

        bought = 0.0
        spent = 0.0
        matched_txs: List[dict] = []

        while bought + 1e-12 < target and spent + 1e-12 < max_spend:
            asks = [o for o in self.state.orders.values() if o.order_type == OrderType.SELL]
            if not asks:
                break
            asks.sort(key=lambda x: (x.price, x.timestamp))
            ask = asks[0]
            seller = self.state.accounts.get(ask.owner)
            if not seller or seller.role != Role.PROVIDER:
                del self.state.orders[ask.id]
                continue
            avail = ask.amount - ask.filled
            match_price = ask.price
            want = target - bought
            budget_left = max_spend - spent
            max_qty_by_budget = budget_left / match_price if match_price > 1e-18 else want
            match_amount = min(avail, want, max_qty_by_budget)
            if match_amount <= 1e-12:
                break
            cost = match_amount * match_price
            if buyer.wrt_balance + 1e-12 < cost:
                break
            buyer.wrt_balance -= cost
            buyer.ant_balance += match_amount
            seller.wrt_balance += cost
            self.state.epoch_ant_sold_volume += match_amount
            ask.filled += match_amount
            bought += match_amount
            spent += cost
            self._push_trade_price(match_price)
            matched_txs.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "trade",
                    "buyer": buyer.address,
                    "seller": seller.address,
                    "price": match_price,
                    "amount": match_amount,
                    "timestamp": time.time(),
                }
            )
            if ask.filled >= ask.amount - 1e-12:
                del self.state.orders[ask.id]

        return matched_txs

    def _execute_market_sell(self, tx: Transaction) -> List[dict]:
        """Продажа до amount ANT по лучшим bid; непроданный остаток возвращается на баланс."""
        seller = self.state.accounts.get(tx.sender)
        if not seller or seller.role != Role.PROVIDER:
            return []
        total_ant = float(tx.amount or 0)
        if total_ant <= 1e-12 or seller.ant_balance + 1e-12 < total_ant:
            return []

        seller.ant_balance -= total_ant
        rem = total_ant
        matched_txs: List[dict] = []

        while rem > 1e-12:
            bids = [o for o in self.state.orders.values() if o.order_type == OrderType.BUY]
            if not bids:
                break
            bids.sort(key=lambda x: (-x.price, x.timestamp))
            bid = bids[0]
            buyer = self.state.accounts.get(bid.owner)
            if not buyer or buyer.role != Role.VALIDATOR:
                del self.state.orders[bid.id]
                continue
            avail = bid.amount - bid.filled
            match_amount = min(avail, rem)
            if match_amount <= 1e-12:
                break
            match_price = bid.price
            total_cost = match_price * match_amount
            buyer.wrt_balance += (bid.price - match_price) * match_amount
            seller.wrt_balance += total_cost
            buyer.ant_balance += match_amount
            self.state.epoch_ant_sold_volume += match_amount
            bid.filled += match_amount
            rem -= match_amount
            self._push_trade_price(match_price)
            matched_txs.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "trade",
                    "buyer": buyer.address,
                    "seller": seller.address,
                    "price": match_price,
                    "amount": match_amount,
                    "timestamp": time.time(),
                }
            )
            if bid.filled >= bid.amount - 1e-12:
                del self.state.orders[bid.id]

        if rem > 1e-12:
            seller.ant_balance += rem
        return matched_txs

    def _network_lzn_total_validators(self) -> float:
        """§5.4: L_total = Σ L_i по всем валидаторам (активированный LZN)."""
        return sum(
            float(a.lzn_frozen_mining)
            for a in self.state.accounts.values()
            if a.role == Role.VALIDATOR
        )

    def _dry_declare_candidate(self, tx: Transaction) -> Optional[dict]:
        """Проверка declare/burn без списания ANT (состояние — на момент вызова)."""
        sender = tx.sender
        if not sender:
            return None
        acc = self.state.accounts.get(sender)
        if not acc or acc.role != Role.VALIDATOR:
            return None
        if tx.tx_type == TransactionType.BURN:
            b, s = float(tx.amount or 0), 0.0
        elif tx.tx_type == TransactionType.DECLARE_PARTICIPATION:
            b, s = float(tx.amount or 0), float(tx.stake_amount or 0)
        else:
            return None
        L_i = float(acc.lzn_frozen_mining)
        if L_i <= 0 or b < 0 or s < 0 or b + s > L_i + 1e-9:
            return None
        if acc.ant_balance + 1e-9 < b + s:
            return None
        w_i = s / L_i if L_i > 1e-18 else 0.0
        return {"tx": tx, "sender": sender, "b": b, "s": s, "L_i": L_i, "w_i": w_i}

    def _finalize_declares_batch(
        self,
        declare_txs: List[Transaction],
        txs_in_block: list,
        participation: Dict[str, dict],
        requeue: List[Transaction],
    ) -> Tuple[float, int]:
        """
        §5.4: после локальных проверок — отсев по λ·L_total (снять наименьшие w_i=s/L_i),
        затем потолок K с наибольшими w_i; исполнить выбранные (оба числа сжигаются).
        Не прошедшие — в requeue. Возвращает (Σ b_i по исполненным, число исполненных declare).
        """
        ordered_unique: List[Transaction] = []
        seen_sid: set = set()
        for tx in declare_txs:
            sid = tx.sender
            if not sid:
                requeue.append(tx)
                continue
            if sid in seen_sid:
                requeue.append(tx)
                continue
            seen_sid.add(sid)
            ordered_unique.append(tx)

        candidates: List[dict] = []
        for tx in ordered_unique:
            c = self._dry_declare_candidate(tx)
            if c:
                candidates.append(c)

        L_total = self._network_lzn_total_validators()
        cap = BURN_CAP_LAMBDA * L_total
        if L_total <= 1e-18:
            requeue.extend(ordered_unique)
            return 0.0, 0

        work = list(candidates)
        work.sort(key=lambda x: (x["w_i"], x["sender"]))
        while work and sum(c["b"] for c in work) > cap + 1e-9:
            work.pop(0)

        if len(work) > MAX_ACTIVE_VALIDATORS_K:
            work.sort(key=lambda x: (-x["w_i"], x["sender"]))
            work = work[:MAX_ACTIVE_VALIDATORS_K]

        selected_hashes = {c["tx"].tx_hash for c in work}
        by_hash = {c["tx"].tx_hash: c for c in work}
        B_applied = 0.0
        applied_count = 0

        for tx in ordered_unique:
            if tx.tx_hash not in selected_hashes:
                requeue.append(tx)
                continue
            c = by_hash.get(tx.tx_hash)
            if not c:
                requeue.append(tx)
                continue
            acc = self.state.accounts.get(c["sender"])
            if not acc or acc.ant_balance + 1e-9 < c["b"] + c["s"]:
                requeue.append(tx)
                continue
            total = c["b"] + c["s"]
            acc.ant_balance -= total
            self.state.current_epoch_burn += total
            d = c["tx"].model_dump(mode="json")
            d["amount"] = c["b"]
            d["stake_amount"] = c["s"]
            txs_in_block.append(d)
            participation[c["sender"]] = {
                "b": c["b"],
                "s": c["s"],
                "L_i": c["L_i"],
                "w_i": c["w_i"],
            }
            B_applied += c["b"]
            applied_count += 1

        return B_applied, applied_count

    def _epoch_boundary(self, block_height: int, txs_in_block: list):
        """§5.5: сброс ANT у Поставщиков; эмиссия = sold×coeff; обновление coeff."""
        if block_height <= 0 or block_height % BLOCKS_PER_EPOCH != 0:
            return

        for oid, order in list(self.state.orders.items()):
            owner = self.state.accounts.get(order.owner)
            if not owner:
                del self.state.orders[oid]
                continue
            rem = order.amount - order.filled
            if order.order_type == OrderType.BUY:
                # Эскроу WRT при лимитном BUY — вернуть неисполненный остаток
                owner.wrt_balance += order.price * rem
            # SELL: ANT уже списаны с баланса в эскроу ордера; на границе эпохи §5.5 остаток
            # сгорает бесследно вместе со снятием книги (без возврата на кошелёк)
            del self.state.orders[oid]

        providers = [a for a in self.state.accounts.values() if a.role == Role.PROVIDER]

        for p in providers:
            wiped = float(p.ant_balance)
            p.ant_balance = 0.0
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.EPOCH_ANT_WIPE.value,
                "receiver": p.address,
                "amount": wiped,
                "asset_type": "ant",
                "details": f"§5.5 step 1: сжигание ANT Поставщика на границе эпохи (height {block_height})",
                "timestamp": time.time(),
            })

        sold_epoch = self.state.epoch_ant_sold_volume
        coeff = self.state.epoch_emission_coefficient
        sold_prev = self.state.epoch_ant_sold_last
        emission = sold_epoch * coeff

        if sold_prev > 1e-12:
            ratio = sold_epoch / sold_prev
            new_coeff = coeff / max(ratio, 1e-12)
        else:
            new_coeff = coeff
        self.state.epoch_emission_coefficient = max(COEFF_MIN, min(COEFF_MAX, new_coeff))
        self.state.epoch_ant_sold_last = sold_epoch
        self.state.epoch_ant_sold_volume = 0.0

        per = (emission / len(providers)) if providers else 0.0
        for p in providers:
            p.ant_balance += per
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.EPOCH_ANT_CREDIT.value,
                "receiver": p.address,
                "amount": per,
                "asset_type": "ant",
                "details": (
                    f"§5.5 step 2: эпохальная эмиссия ANT Поставщику "
                    f"(sold×coeff / число поставщиков; всего эмиссия {emission:.4f})"
                ),
                "timestamp": time.time(),
            })

        txs_in_block.append({
            "tx_hash": uuid.uuid4().hex,
            "tx_type": TransactionType.EPOCH_EMISSION.value,
            "amount": emission,
            "asset_type": "ant",
            "details": (
                f"§5.5 epoch summary height {block_height}: providers={len(providers)}; "
                f"sold_epoch={sold_epoch:.4f}; coeff {coeff:.4f}→{self.state.epoch_emission_coefficient:.4f}; "
                f"total_emit={emission:.4f} ANT (только кошельки Поставщиков, не валидаторы)"
            ),
            "timestamp": time.time(),
        })
        self.state.current_epoch_burn = 0.0

    def _begin_block(self, height: int, txs_in_block: list) -> None:
        """
        Предблок — аналог Cosmos SDK BeginBlock (до DeliverTx по пользовательским сообщениям).
        В ленте блока: маркер фазы, затем протокольная санитизация §4.1–4.2.
        """
        txs_in_block.append(
            {
                "tx_hash": uuid.uuid4().hex,
                "tx_type": "begin_block",
                "sender": "",
                "receiver": "",
                "details": (
                    f"BeginBlock (предблок) height={height}: до исполнения tx из мемпула; "
                    "в полной цепи — evidence, слэшинг, выплаты за прошлый блок и т.п."
                ),
                "timestamp": time.time(),
            }
        )
        self._sanitize_ant_ineligible_roles(txs_in_block)

    def _end_block(
        self,
        block_height: int,
        txs_in_block: list,
        participation: Dict[str, dict],
        B_applied: float,
        cap: float,
        burn_band_ok: bool,
        fee_tx_count: int,
    ) -> None:
        """
        Постблок — аналог Cosmos SDK EndBlock (после всех DeliverTx и внутриблочного матчинга).
        §5.1 базовая WRT, §5.4 делёж комиссий, §5.5 эпоха ANT; на полной цепи — ValidatorUpdates (§5.4).
        """
        reward_amount = 50.0
        eligible = [
            (addr, data["L_i"])
            for addr, data in participation.items()
            if data["b"] > 0 and data["L_i"] > 0
        ]
        if burn_band_ok and eligible:
            total_L = sum(L for _, L in eligible)
            for addr, L_i in eligible:
                v = self.state.accounts.get(addr)
                if v:
                    share = (L_i / total_L) * reward_amount
                    v.wrt_balance += share
            txs_in_block.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": TransactionType.BLOCK_REWARD.value,
                    "receiver": "validators",
                    "amount": reward_amount,
                    "asset_type": "wrt",
                    "details": (
                        f"§5.1 EndBlock: WRT block reward; Σb_i≈λ·L_total (B={B_applied:.4f}, cap={cap:.4f}); "
                        "доля ∝ L_i среди валидаторов с b_i>0"
                    ),
                    "timestamp": time.time(),
                }
            )
        elif eligible and not burn_band_ok:
            txs_in_block.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "block_reward_skipped",
                    "receiver": "validators",
                    "amount": 0.0,
                    "asset_type": "wrt",
                    "details": (
                        f"§5.4 EndBlock: нет базовой WRT — Σb_i={B_applied:.4f} вне цели λ·L_total={cap:.4f} "
                        f"(±{BURN_TARGET_ATOL})"
                    ),
                    "timestamp": time.time(),
                }
            )

        F = SIM_FEE_WRT_PER_LEDGER_TX * max(0, fee_tx_count)
        if burn_band_ok and B_applied > 1e-18 and F > 0:
            for addr, data in participation.items():
                bi = float(data["b"])
                if bi <= 0:
                    continue
                share_f = F * (bi / B_applied)
                v = self.state.accounts.get(addr)
                if v:
                    v.wrt_balance += share_f
            txs_in_block.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "fee_distribution",
                    "receiver": "validators",
                    "amount": F,
                    "asset_type": "wrt",
                    "details": f"§5.4 EndBlock: комиссии блока ∝ b_i/B (F={F:.6f}, B={B_applied:.4f})",
                    "timestamp": time.time(),
                }
            )

        self._epoch_boundary(block_height, txs_in_block)

        txs_in_block.append(
            {
                "tx_hash": uuid.uuid4().hex,
                "tx_type": "end_block",
                "sender": "",
                "receiver": "",
                "details": (
                    f"EndBlock (постблок) height={block_height}: эпоха §5.5 применена на границе; "
                    "на полной цепи CometBFT здесь же ValidatorUpdates по declare §5.4 для h+1"
                ),
                "timestamp": time.time(),
            }
        )

    async def produce_block(self):
        """
        Цикл блока по порядку, близкому к CometBFT + Cosmos SDK (каноническая цепь):
        BeginBlock → DeliverTx (мемпул по сообщениям) → пакет declare §5.4 →
        внутриблочный матчинг §5.2 → EndBlock (награды, комиссии, эпоха §5.5) →
        запись блока и персистентность.
        Интервал между блоками в эталоне — CANONICAL_BLOCK_INTERVAL_SEC (60 с);
        в симуляции задаётся sim_block_interval_sec (тесты 0.1–300 с).
        """
        async with self._state_lock:
            await self._produce_block_body()

    async def _produce_block_body(self):
        balance_snap = {
            addr: (float(acc.wrt_balance), float(acc.ant_balance))
            for addr, acc in self.state.accounts.items()
        }
        txs_in_block = []
        block_height = self.state.current_height + 1
        self._begin_block(block_height, txs_in_block)
        declare_buffer: List[Transaction] = []
        participation: Dict[str, dict] = {}
        fee_ledger_ops = 0  # только успешные пользовательские tx (не begin_block / protocol_*)
    
        # Process mempool (declare/burn — пакетно после §5.4 λ/K)
        for tx in self.state.mempool:
            if tx.tx_type in (TransactionType.BURN, TransactionType.DECLARE_PARTICIPATION):
                declare_buffer.append(tx)
                continue
            if tx.tx_type == TransactionType.TRANSFER:
                sender = self.state.accounts.get(tx.sender)
                receiver_addr = tx.receiver
                if sender and receiver_addr:
                    recv = self.state.accounts.get(receiver_addr)
                    if not recv:
                        recv = self.state.create_account(receiver_addr)
                    asset = (getattr(tx, "asset_type", None) or "wrt").lower()
                    amt = float(tx.amount or 0)
                    if amt > 0:
                        if asset == "wrt" and sender.wrt_balance >= amt:
                            sender.wrt_balance -= amt
                            recv.wrt_balance += amt
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
                        elif asset == "lzn" and sender.lzn_balance >= amt:
                            sender.lzn_balance -= amt
                            recv.lzn_balance += amt
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
                        else:
                            if asset not in ("wrt", "lzn"):
                                log_engine_skip(
                                    self.state,
                                    block_height=block_height,
                                    tx_hash=tx.tx_hash,
                                    tx_type="transfer",
                                    canon="§4.1",
                                    title="Перевод отклонён: по канону только WRT и LZN (не ANT MsgSend)",
                                    detail=f"asset={asset}",
                                )
                            elif asset == "wrt":
                                log_engine_skip(
                                    self.state,
                                    block_height=block_height,
                                    tx_hash=tx.tx_hash,
                                    tx_type="transfer",
                                    canon="§4.1",
                                    title="Перевод WRT не исполнен",
                                    detail="Недостаточно баланса отправителя",
                                )
                            elif asset == "lzn":
                                log_engine_skip(
                                    self.state,
                                    block_height=block_height,
                                    tx_hash=tx.tx_hash,
                                    tx_type="transfer",
                                    canon="§4.1",
                                    title="Перевод LZN не исполнен",
                                    detail="Недостаточно ликвидного LZN",
                                )
            elif tx.tx_type == TransactionType.MINT:
                if tx.sender != SIM_TREASURY_ADDR:
                    continue
                treasury = self.state.accounts.get(SIM_TREASURY_ADDR)
                receiver_addr = tx.receiver
                if not treasury or not receiver_addr:
                    continue
                receiver = self.state.accounts.get(receiver_addr)
                if not receiver:
                    receiver = self.state.create_account(receiver_addr)
                asset = (getattr(tx, "asset_type", None) or "wrt").lower()
                amt = float(tx.amount or 0)
                if amt <= 0:
                    continue
                if asset == "ant" and receiver.role not in (Role.PROVIDER, Role.VALIDATOR):
                    log_engine_skip(
                        self.state,
                        block_height=block_height,
                        tx_hash=tx.tx_hash,
                        tx_type="mint",
                        canon="§4.1–4.2",
                        title="Mint ANT отклонён — соответствует канону",
                        detail=f"Получатель role={receiver.role.value}; ANT на кошельке только у Поставщика и Валидатора.",
                    )
                    continue
                ok = False
                if asset == "wrt" and treasury.wrt_balance + 1e-12 >= amt:
                    treasury.wrt_balance -= amt
                    receiver.wrt_balance += amt
                    ok = True
                elif asset == "lzn" and treasury.lzn_balance + 1e-12 >= amt:
                    treasury.lzn_balance -= amt
                    receiver.lzn_balance += amt
                    ok = True
                elif asset == "ant" and treasury.ant_balance + 1e-12 >= amt:
                    treasury.ant_balance -= amt
                    receiver.ant_balance += amt
                    ok = True
                if ok:
                    txs_in_block.append(tx.model_dump(mode="json"))
                    fee_ledger_ops += 1
            elif tx.tx_type == TransactionType.SET_ROLE:
                addr = tx.sender or tx.receiver
                role_applied = False
                if addr and tx.role:
                    rcv = self.state.accounts.get(addr)
                    if not rcv:
                        rcv = self.state.create_account(addr)
                    new_role = tx.role
                    if new_role == Role.CITIZEN:
                        self._release_orders_for_owner(addr, rcv)
                        rcv.ant_balance = 0.0
                        rcv.role = Role.CITIZEN
                        txs_in_block.append(tx.model_dump(mode="json"))
                        fee_ledger_ops += 1
                        role_applied = True
                    elif new_role == Role.VALIDATOR and eligible_for_validator_role(addr, rcv):
                        rcv.role = Role.VALIDATOR
                        txs_in_block.append(tx.model_dump(mode="json"))
                        fee_ledger_ops += 1
                        role_applied = True
                    elif new_role == Role.PROVIDER and eligible_for_provider_role(addr, rcv):
                        rcv.role = Role.PROVIDER
                        txs_in_block.append(tx.model_dump(mode="json"))
                        fee_ledger_ops += 1
                        role_applied = True
                    if not role_applied:
                        log_engine_skip(
                            self.state,
                            block_height=block_height,
                            tx_hash=tx.tx_hash,
                            tx_type="set_role",
                            canon="§4.2 / §3.1",
                            title="set_role не применён",
                            detail=f"Запрошено {tx.role.value}: не выполнены ZKP/LZN или правила §4.2.",
                        )
            elif tx.tx_type == TransactionType.ZKP_VERIFY:
                zaddr = tx.sender or tx.receiver
                if zaddr:
                    zacc = self.state.accounts.get(zaddr)
                    if zacc and not zacc.zkp_verified:
                        zacc.zkp_verified = True
                        txs_in_block.append(tx.model_dump(mode="json"))
                        fee_ledger_ops += 1
            elif tx.tx_type == TransactionType.ACTIVATE_LZN:
                act = self.state.accounts.get(tx.sender)
                if act and act.role == Role.VALIDATOR:
                    amt = float(tx.amount or 0)
                    if amt > 0 and act.lzn_balance + 1e-12 >= amt:
                        if act.lzn_frozen_mining + amt <= float(LZN_MAX_FROZEN_PER_ADDRESS) + 1e-9:
                            act.lzn_balance -= amt
                            act.lzn_frozen_mining += amt
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
            elif tx.tx_type == TransactionType.CREATE_ORDER:
                owner = self.state.accounts.get(tx.sender)
                if owner and owner.role in (Role.PROVIDER, Role.VALIDATOR):
                    if getattr(tx, "market", False):
                        mtr: List[dict] = []
                        if tx.order_type == OrderType.BUY:
                            mtr = self._execute_market_buy(tx)
                        elif tx.order_type == OrderType.SELL:
                            mtr = self._execute_market_sell(tx)
                        if mtr:
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
                            txs_in_block.extend(mtr)
                            fee_ledger_ops += len(mtr)
                        else:
                            log_engine_skip(
                                self.state,
                                block_height=block_height,
                                tx_hash=tx.tx_hash,
                                tx_type="create_order",
                                canon="§5.2",
                                title="Рыночная заявка без исполнения",
                                detail="Нет подходящей ликвидности в книге или исчерпан лимит WRT/ANT.",
                            )
                    elif (
                        tx.order_type == OrderType.BUY
                        and owner.role == Role.VALIDATOR
                        and owner.wrt_balance >= (tx.price * tx.amount)
                    ):
                        owner.wrt_balance -= (tx.price * tx.amount)
                        order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                        self.state.orders[tx.tx_hash] = order
                        txs_in_block.append(tx.model_dump(mode="json"))
                        fee_ledger_ops += 1
                    elif tx.order_type == OrderType.SELL:
                        if owner.role != Role.PROVIDER:
                            pass
                        elif owner.ant_balance >= tx.amount:
                            owner.ant_balance -= tx.amount
                            order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                            self.state.orders[tx.tx_hash] = order
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
            elif tx.tx_type == TransactionType.CANCEL_ORDER:
                order = self.state.orders.get(tx.order_id)
                if order and order.owner == tx.sender:
                    owner = self.state.accounts.get(tx.sender)
                    if owner:
                        if order.order_type == OrderType.BUY:
                            owner.wrt_balance += (order.price * (order.amount - order.filled))
                        else:
                            owner.ant_balance += (order.amount - order.filled)
                        del self.state.orders[tx.order_id]
                        txs_in_block.append(tx.model_dump(mode="json"))
                        fee_ledger_ops += 1
            else:
                txs_in_block.append(
                    {
                        "tx_hash": getattr(tx, "tx_hash", uuid.uuid4().hex),
                        "tx_type": "mempool_tx_dropped",
                        "sender": getattr(tx, "sender", None) or "",
                        "details": f"Тип {tx.tx_type!r} не обрабатывается в DeliverTx симулятора",
                        "timestamp": time.time(),
                    }
                )
    
        # DeliverTx: мемпул (кроме declare), затем пакет declare §5.4, затем матчинг ордеров
        requeue_declare: List[Transaction] = []
        B_applied, _ndecl = self._finalize_declares_batch(
            declare_buffer, txs_in_block, participation, requeue_declare
        )
        fee_tx_count = fee_ledger_ops + _ndecl
    
        trades = self._match_orders()
        txs_in_block.extend(trades)
        fee_tx_count += len(trades)
    
        L_total = self._network_lzn_total_validators()
        cap = BURN_CAP_LAMBDA * L_total
        burn_band_ok = L_total <= 1e-18 or (
            B_applied > 0
            and B_applied >= cap - BURN_TARGET_ATOL
            and B_applied <= cap + BURN_TARGET_ATOL
        )
    
        self._end_block(block_height, txs_in_block, participation, B_applied, cap, burn_band_ok, fee_tx_count)
    
        audit_block_ledger(self.state, txs_in_block, block_height)
    
        delta_map: Dict[str, Dict[str, float]] = {}
        for addr, acc in self.state.accounts.items():
            w0, a0 = balance_snap.get(addr, (0.0, 0.0))
            dw = float(acc.wrt_balance) - w0
            da = float(acc.ant_balance) - a0
            if abs(dw) > 1e-9 or abs(da) > 1e-9:
                delta_map[addr] = {"wrt": dw, "ant": da}
        self.state.last_block_wallet_delta = delta_map
    
        import hashlib
        block_hash = hashlib.sha256(f"{self.state.current_height + 1}{time.time()}".encode()).hexdigest()[:16]
        
        block = {
            "height": self.state.current_height + 1,
            "hash": block_hash,
            "timestamp": time.time(),
            "transactions": txs_in_block,
            "tx_count": len(txs_in_block)
        }
        
        # Calculate TPS
        tps = len(txs_in_block) / self.block_time
        self.state.tps_history.append({
            "time": time.strftime("%H:%M:%S"),
            "tps": round(tps, 2)
        })
        if len(self.state.tps_history) > 50:
            self.state.tps_history.pop(0)
        
        self.state.mempool = requeue_declare
        self.state.add_block(block)
    
        try:
            self.state.save_state()
        except OSError as e:
            print(f"Warning: save_state failed after block {block['height']}: {e}")
    
        # Broadcast new state to all connected clients
        await self.ws_manager.broadcast({
            "type": "new_block",
            "data": {
                "block": block,
                "state": self.state.get_full_state(),
                "block_time": self.block_time
            }
        })
        
        print(f"Produced block {block['height']} with {len(txs_in_block)} txs")
