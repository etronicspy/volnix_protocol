import asyncio
import time
from typing import Callable, Dict, List, Optional, Tuple

from fastapi import WebSocket


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

import uuid

from core.canon_audit import audit_block_ledger, log_engine_skip
from core.consensus import (
    Decision,
    Evidence,
    FaultModel,
    run_consensus,
    slashing_amount,
)
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
    BLOCKS_PER_EPOCH,
    BURN_CAP_LAMBDA,
    CANONICAL_BLOCK_INTERVAL_SEC,
    GENESIS_VALIDATOR_ADDR,
    LZN_MAX_EPOCH,
    LZN_MAX_FROZEN_PER_ADDRESS,
    MAX_SIM_BLOCK_INTERVAL_SEC,
    MIN_SIM_BLOCK_INTERVAL_SEC,
    SIM_BOOTSTRAP_INJECT_HEIGHT,
    SIM_SPEED_MAX,
    SIM_SPEED_MIN,
    SIM_TREASURY_ADDR,
    burn_ceiling,
    burn_floor,
    consensus_validator_set_from_participation,
    eligible_for_provider_role,
    eligible_for_validator_role,
    lzn_mint_headroom,
    order_asset,
    select_proposer_for_height,
    validator_set_from_activated_lzn,
)

# Batch-режим скорости: если интервал блока меньше тика, движок производит
# пачку блоков за тик и шлёт в WS одну агрегированную дельту на пачку.
BATCH_TICK_SEC = 0.1

COEFF_MIN = 0.75
COEFF_MAX = 1.5
# Ruleset v2 (§5.5): сглаживание коэффициента эмиссии — EMA-вес нового целевого значения.
# coeff_new = α·(coeff/ratio) + (1−α)·coeff, затем клиппинг [COEFF_MIN, COEFF_MAX].
EMISSION_COEFF_ALPHA = 0.5
# Ruleset v2 (§5.5): нижний порог знаменателя ratio — при sold_prev ниже порога
# коэффициент не пересчитывается (защита от деления на ноль на genesis / мёртвой эпохе).
SOLD_PREV_FLOOR = 1e-9

# §5.4 / §7.2: коридор Σb_i ∈ [λ·L_total, (1−λ)·L_total]; потолок подписантов K.
# BURN_CAP_LAMBDA / burn_floor / burn_ceiling — в core.state; реэкспорт для тестов.
MAX_ACTIVE_VALIDATORS_K = 150


def empty_povb_competition(L_total: float = 0.0, *, kind: str = "povb") -> dict:
    """Снимок соревнования §5.4, который пишется в блок (genesis / пустой пакет)."""
    L = float(L_total)
    return {
        "kind": kind,
        "lambda": float(BURN_CAP_LAMBDA),
        "K": int(MAX_ACTIVE_VALIDATORS_K),
        "L_total": L,
        "floor": float(burn_floor(L)),
        "cap": float(burn_ceiling(L)),
        "B_candidates": 0.0,
        "B_selected": 0.0,
        "candidates_count": 0,
        "selected_count": 0,
        "culled_lambda_count": 0,
        "culled_k_count": 0,
        "deferred_count": 0,
        "entries": [],
    }


def build_povb_competition(
    *,
    L_total: float,
    candidates: List[dict],
    selected: List[dict],
    culled_lambda: List[dict],
    culled_k: List[dict],
    B_selected: float,
    deferred_count: int = 0,
    applied_hashes: Optional[set] = None,
) -> dict:
    """Ранжирование по убыванию w_i (tie-break: адрес). Статус — исход λ/K."""
    selected_set = {c["tx"].tx_hash for c in selected}
    lambda_set = {c["tx"].tx_hash for c in culled_lambda}
    k_set = {c["tx"].tx_hash for c in culled_k}
    applied = applied_hashes if applied_hashes is not None else selected_set
    ranked = sorted(candidates, key=lambda x: (-float(x["w_i"]), str(x["sender"])))
    entries: List[dict] = []
    for i, c in enumerate(ranked, start=1):
        h = c["tx"].tx_hash
        if h in lambda_set:
            status = "culled_lambda"
        elif h in k_set:
            status = "culled_k"
        elif h in selected_set and h not in applied:
            status = "deferred"
        elif h in selected_set:
            status = "selected"
        else:
            status = "deferred"
        entries.append(
            {
                "rank": i,
                "address": c["sender"],
                "tx_hash": h,
                "b": float(c["b"]),
                "s": float(c["s"]),
                "L_i": float(c["L_i"]),
                "w_i": float(c["w_i"]),
                "status": status,
            }
        )
    B_cand = float(sum(float(c["b"]) for c in candidates))
    snap = empty_povb_competition(L_total)
    snap["B_candidates"] = B_cand
    snap["B_selected"] = float(B_selected)
    snap["candidates_count"] = len(candidates)
    snap["selected_count"] = sum(1 for e in entries if e["status"] == "selected")
    snap["culled_lambda_count"] = len(culled_lambda)
    snap["culled_k_count"] = len(culled_k)
    snap["deferred_count"] = int(deferred_count) + sum(
        1 for e in entries if e["status"] == "deferred"
    )
    snap["entries"] = entries
    return snap
# §5.2/§5.4: условный пул комиссий блока (симуляция); дележ F·(b_i/B), B=Σb_i
SIM_FEE_WRT_PER_LEDGER_TX = 0.02

class SimulationEngine:
    def __init__(self, state_manager, security_config: Optional[SecurityConfig] = None):
        self.state = state_manager
        # Дефолт = эталон цепи (60 с); после load_state startup выставит из state.json
        self.block_time = float(
            getattr(state_manager, "sim_block_interval_sec", CANONICAL_BLOCK_INTERVAL_SEC)
        )
        # Скорость симуляции: сим. секунд за реальную (1 блок = 60 сим. секунд).
        self.speed = float(getattr(state_manager, "sim_speed", 1.0))
        # Фактическая скорость (EMA по batch-тикам) — для UI при высоких скоростях.
        self.effective_speed = self.speed
        self.is_running = False
        self.ws_manager = ConnectionManager()
        # Хук перед блоком: AutoMarket / BotEngine / AutoDeclare (см. main lifespan).
        # На высоких sim_speed wall-clock демоны не успевают — только pre_block.
        self.pre_block_hook: Optional[Callable[[], None]] = None
        # Сброс состояния и produce_block не должны пересекаются — иначе старый блок перезаписывает сброс.
        self._state_lock = asyncio.Lock()
        # Этап 3: модель сбоев консенсуса (PreVote/PreCommit/double_sign). По умолчанию выключена.
        self.consensus_fault_model: FaultModel = FaultModel()
        self._evidence_log: List[Evidence] = []
        # Параметры slashing — доля от lzn_frozen_mining: часть → burn, часть → казне.
        self.slashing_fraction = 0.10
        self.slashing_burn_share = 0.5
        # Security subsystems (§3.1, §3.3, §5.3, §5.4, §6.1, §7.2)
        self.security_config = security_config or SecurityConfig()
        self.moa_tracker = MOATracker(config=self.security_config)
        self.zkp_registry = ZKPRegistry(config=self.security_config)
        # Последний снимок λ/K — пишется в committed-блок как ``competition``.
        self.last_povb_competition: dict = empty_povb_competition()

    def set_speed(self, speed: float) -> None:
        """Задать скорость симуляции: сим. секунд за 1 реальную секунду.

        1.0 → блок раз в 60 с (реальное время); 604800 → 1 с = 1 неделя
        (10080 блоков/с, batch-режим, фактический темп ограничен CPU).
        """
        s = max(SIM_SPEED_MIN, min(float(speed), SIM_SPEED_MAX))
        self.speed = s
        self.block_time = max(
            MIN_SIM_BLOCK_INTERVAL_SEC,
            min(CANONICAL_BLOCK_INTERVAL_SEC / s, MAX_SIM_BLOCK_INTERVAL_SEC),
        )
        self.effective_speed = s
        self.state.sim_speed = s
        self.state.sim_block_interval_sec = self.block_time

    def set_block_time(self, time_sec: float):
        """Совместимость: прямое задание интервала блока (0.1–300 с) пересчитывается в speed."""
        t = max(0.1, min(float(time_sec), 300.0))
        self.set_speed(CANONICAL_BLOCK_INTERVAL_SEC / t)

    def set_consensus_fault_model(
        self,
        *,
        p_absent: float = 0.0,
        p_nil: float = 0.0,
        p_double_sign: float = 0.0,
        seed: Optional[int] = None,
    ) -> None:
        """Включить многовалидаторный режим (по умолчанию все нули — однопропозерный fallback)."""
        self.consensus_fault_model = FaultModel(
            p_absent=max(0.0, min(1.0, float(p_absent))),
            p_nil=max(0.0, min(1.0, float(p_nil))),
            p_double_sign=max(0.0, min(1.0, float(p_double_sign))),
            seed=seed,
        )

    def _has_active_faults(self) -> bool:
        fm = self.consensus_fault_model
        return fm.p_absent > 0 or fm.p_nil > 0 or fm.p_double_sign > 0

    @property
    def is_consensus_enabled(self) -> bool:
        """Multi-round PreVote/PreCommit включается в двух случаях:

        - в ValidatorSet >= 2 узлов (естественная многоузловость симуляции);
        - явно включён FaultModel (стресс-тест single-validator).
        """
        vs = getattr(self.state, "consensus_validator_set", []) or []
        return len(vs) >= 2 or self._has_active_faults()

    def _apply_slashing(self, evidence_list: List[Evidence], txs_in_block: list) -> None:
        """§5.4: за double_sign снимаем долю lzn_frozen_mining; часть → burn, часть → казне."""
        if not evidence_list:
            return
        treasury = self.state.accounts.get(SIM_TREASURY_ADDR)
        burn_share = max(0.0, min(1.0, float(self.slashing_burn_share)))
        for ev in evidence_list:
            acc = self.state.accounts.get(ev.address)
            if not acc or acc.lzn_frozen_mining <= 0:
                continue
            amt = slashing_amount(0.0, float(acc.lzn_frozen_mining), self.slashing_fraction)
            if amt <= 0:
                continue
            burn = round(amt * burn_share, 6)
            to_treasury = round(amt - burn, 6)
            acc.lzn_frozen_mining = max(0.0, acc.lzn_frozen_mining - amt)
            if treasury is not None:
                treasury.lzn_balance += to_treasury
            slash_tx = {
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.SLASH_EVIDENCE.value,
                "sender": ev.address,
                "receiver": SIM_TREASURY_ADDR,
                "amount": amt,
                "asset_type": "lzn",
                "details": (
                    f"§5.4 slashing {ev.kind}: amt={amt:.4f} LZN "
                    f"(burn={burn:.4f}, treasury={to_treasury:.4f}); reason={ev.detail}"
                ),
                "timestamp": time.time(),
            }
            txs_in_block.append(slash_tx)
            try:
                self.state.canon_log.push(
                    source="consensus",
                    status="warn",
                    category="slashing",
                    canon="§5.4",
                    title=f"Slashing {ev.kind}: {ev.address[:14]}…",
                    detail=slash_tx["details"],
                    tx_hash=slash_tx["tx_hash"],
                    block_height=self.state.current_height + 1,
                )
            except Exception:
                pass

    def _rollback_to(self, snapshot: dict) -> None:
        """Откат состояния к снимку до начала блока (блок не принят консенсусом / инвариантами)."""
        self.state.current_height = snapshot["current_height"]
        self.state.accounts = snapshot["accounts"]
        self.state.orders = snapshot["orders"]
        self.state.mempool = snapshot["mempool"]
        self.state.blocks = snapshot["blocks"]
        self.state.tps_history = snapshot["tps_history"]
        self.state.last_price = snapshot["last_price"]
        self.state.price_history = snapshot["price_history"]
        self.state.current_epoch_burn = snapshot["current_epoch_burn"]
        self.state.epoch_ant_sold_volume = snapshot["epoch_ant_sold_volume"]
        self.state.epoch_ant_sold_last = snapshot["epoch_ant_sold_last"]
        self.state.epoch_emission_coefficient = snapshot["epoch_emission_coefficient"]
        self.state.epoch_lzn_sold_volume = snapshot.get("epoch_lzn_sold_volume", 0.0)
        self.state.epoch_lzn_sold_last = snapshot.get("epoch_lzn_sold_last", 0.0)
        self.state.epoch_lzn_emission_coefficient = snapshot.get(
            "epoch_lzn_emission_coefficient", 1.0
        )
        self.state.last_lzn_price = snapshot.get("last_lzn_price", 0.0)
        self.state.lzn_price_history = snapshot.get("lzn_price_history", [])
        self.state.last_block_wallet_delta = snapshot["last_block_wallet_delta"]
        self.state.consensus_validator_set = snapshot["consensus_validator_set"]
        self.state.sim_bootstrap_injected = bool(snapshot.get("sim_bootstrap_injected", False))

    def _release_orders_for_owner(self, owner_addr: str, acc):
        """Вернуть эскроу ордеров владельцу и удалить ордера из книги."""
        for oid, order in list(self.state.orders.items()):
            if order.owner != owner_addr:
                continue
            rem = order.amount - order.filled
            if order.order_type == OrderType.BUY:
                acc.wrt_balance += order.price * rem
            elif order_asset(order) == "lzn":
                acc.lzn_balance += rem
            else:
                acc.ant_balance += rem
            del self.state.orders[oid]

    def _cancel_orders_with_stale_side(self, txs_in_block: list) -> None:
        """§4.2/§5.2: BUY — только Валидатор, SELL — только Поставщик.

        Роль меняется после выставления ордера (§4.2), и заявка остаётся в книге
        с «чужой» стороной. Такие ордера не исполнимы и, стоя на вершине книги,
        блокируют матчинг. Снимаем их с возвратом эскроу.
        """
        for oid, order in list(self.state.orders.items()):
            owner = self.state.accounts.get(order.owner)
            if not owner or owner.address == SIM_TREASURY_ADDR:
                continue
            if order.order_type == OrderType.BUY:
                ok = owner.role == Role.VALIDATOR
            else:
                ok = owner.role == Role.PROVIDER
            if ok:
                continue
            rem = order.amount - order.filled
            if order.order_type == OrderType.BUY:
                owner.wrt_balance += order.price * rem
                asset, back = "wrt", order.price * rem
            elif order_asset(order) == "lzn":
                owner.lzn_balance += rem
                asset, back = "lzn", rem
            else:
                owner.ant_balance += rem
                asset, back = "ant", rem
            del self.state.orders[oid]
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": "protocol_order_cancel",
                "receiver": owner.address,
                "amount": back,
                "asset_type": asset,
                "details": (
                    f"§4.2/§5.2: сторона ордера не соответствует роли "
                    f"({order.order_type.value} при role={owner.role.value}); эскроу возвращён"
                ),
                "timestamp": time.time(),
            })

    def _sanitize_ant_ineligible_roles(self, txs_in_block: list):
        """
        §4.1–4.2: ANT только у Поставщика и Валидатора; Гражданин (тип 1) — без ANT и без ордеров на рынке.
        Казначейство симуляции исключено.
        """
        self._cancel_orders_with_stale_side(txs_in_block)
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
        print(f"Simulation Engine started. Block time: {self.block_time}s (speed ×{self.speed:g})")
        while self.is_running:
            if self.block_time >= BATCH_TICK_SEC:
                # Обычный режим: один блок за цикл, WS-дельта на каждый блок.
                self.effective_speed = self.speed
                self._run_pre_block_hook()
                try:
                    await self.produce_block()
                except RuntimeError:
                    # Блок отклонён (min-burn/кворум/инварианты) — состояние откатано,
                    # высота не растёт, цикл продолжается до следующей попытки.
                    pass
                await asyncio.sleep(self.block_time)
                continue
            await self._produce_batch_tick()

    def _run_pre_block_hook(self) -> None:
        """§5.4: declare подаётся перед каждой высотой, независимо от скорости."""
        if self.pre_block_hook is None:
            return
        try:
            self.pre_block_hook()
        except Exception as e:  # pragma: no cover
            print(f"pre_block_hook error: {e}")

    async def _produce_batch_tick(self) -> None:
        """Batch-режим высоких скоростей: пачка блоков за тик + одна WS-дельта.

        Целевое число блоков = tick / block_time; при нехватке CPU пачка
        обрезается по бюджету времени — фактическая скорость видна в
        effective_speed (EMA), UI не деградирует (одно сообщение на тик).
        """
        t0 = time.monotonic()
        target = max(1, int(round(BATCH_TICK_SEC / self.block_time)))
        accounts_before = self.state.snapshot_accounts()
        orders_before = self.state.snapshot_orders()
        produced = 0
        while produced < target and self.is_running:
            self._run_pre_block_hook()
            try:
                await self.produce_block(quiet=True)
            except RuntimeError:
                # Блок отклонён (кворум/инварианты) — состояние откатано, продолжаем.
                pass
            produced += 1
            if time.monotonic() - t0 >= BATCH_TICK_SEC:
                break
        elapsed = max(1e-6, time.monotonic() - t0)
        # EMA фактической скорости: produced блоков × 60 сим.с за elapsed реальных секунд
        inst = produced * CANONICAL_BLOCK_INTERVAL_SEC / elapsed
        self.effective_speed = 0.7 * self.effective_speed + 0.3 * inst

        if produced > 0 and self.state.blocks:
            delta = self.state.compute_delta(accounts_before, orders_before)
            block = self.state.blocks[-1]
            await self.ws_manager.broadcast(
                {
                    "type": "block_delta",
                    "data": {
                        "block": block,
                        "delta": delta,
                        "blocks_in_batch": produced,
                        "market": self.state.get_orderbook(),
                        "consensus_validators": list(self.state.consensus_validator_set),
                        "next_proposer": select_proposer_for_height(
                            self.state.current_height + 1,
                            self.state.consensus_validator_set,
                            GENESIS_VALIDATOR_ADDR,
                        ),
                        "current_epoch_burn": self.state.current_epoch_burn,
                        "epoch_ant_sold_volume": self.state.epoch_ant_sold_volume,
                        "epoch_emission_coefficient": self.state.epoch_emission_coefficient,
                        "last_block_wallet_delta": self.state.last_block_wallet_delta,
                        "mempool_size": len(self.state.mempool),
                        "tps_history_tail": self.state.tps_history[-30:],
                        "block_time": self.block_time,
                        "sim_speed": self.speed,
                        "effective_speed": self.effective_speed,
                    },
                }
            )
        await asyncio.sleep(max(0.0, BATCH_TICK_SEC - elapsed) + 1e-4)

    def stop(self):
        self.is_running = False
        print("Simulation Engine stopped.")

    def _match_orders(self):
        matched: List[dict] = []
        for asset in ("ant", "lzn"):
            matched.extend(self._match_orders_for_asset(asset))
        return matched

    def _match_orders_for_asset(self, asset: str) -> List[dict]:
        bids = [
            o
            for o in self.state.orders.values()
            if o.order_type == OrderType.BUY and order_asset(o) == asset
        ]
        asks = [
            o
            for o in self.state.orders.values()
            if o.order_type == OrderType.SELL and order_asset(o) == asset
        ]
        bids.sort(key=lambda x: (-x.price, x.timestamp))
        asks.sort(key=lambda x: (x.price, x.timestamp))
        matched_txs: List[dict] = []

        while bids and asks:
            highest_bid = bids[0]
            lowest_ask = asks[0]
            if highest_bid.price < lowest_ask.price:
                break
            match_price = lowest_ask.price
            match_amount = min(
                highest_bid.amount - highest_bid.filled,
                lowest_ask.amount - lowest_ask.filled,
            )
            buyer = self.state.accounts.get(highest_bid.owner)
            seller = self.state.accounts.get(lowest_ask.owner)
            total_cost = match_price * match_amount
            if (
                buyer
                and seller
                and buyer.role == Role.VALIDATOR
                and seller.role == Role.PROVIDER
            ):
                buyer.wrt_balance += (highest_bid.price - match_price) * match_amount
                seller.wrt_balance += total_cost
                if asset == "lzn":
                    buyer.lzn_balance += match_amount
                    self.state.epoch_lzn_sold_volume += match_amount
                else:
                    buyer.ant_balance += match_amount
                    self.state.epoch_ant_sold_volume += match_amount
                highest_bid.filled += match_amount
                lowest_ask.filled += match_amount
                self.state.record_trade_price(match_price, asset=asset)
                matched_txs.append(
                    {
                        "tx_hash": uuid.uuid4().hex,
                        "tx_type": "trade",
                        "buyer": buyer.address,
                        "seller": seller.address,
                        "price": match_price,
                        "amount": match_amount,
                        "asset_type": asset,
                        "timestamp": time.time(),
                    }
                )
            else:
                if not buyer or buyer.role != Role.VALIDATOR:
                    bids.pop(0)
                if not seller or seller.role != Role.PROVIDER:
                    asks.pop(0)
                continue
            if highest_bid.filled >= highest_bid.amount:
                bids.pop(0)
                del self.state.orders[highest_bid.id]
            if lowest_ask.filled >= lowest_ask.amount:
                asks.pop(0)
                del self.state.orders[lowest_ask.id]
        return matched_txs

    def _push_trade_price(self, match_price: float, *, asset: str = "ant") -> None:
        self.state.record_trade_price(match_price, asset=asset)

    def _execute_market_buy(self, tx: Transaction) -> List[dict]:
        """Покупка до amount актива по лучшим ask, не более max_wrt WRT."""
        buyer = self.state.accounts.get(tx.sender)
        if not buyer or buyer.role != Role.VALIDATOR:
            return []
        asset = (getattr(tx, "asset_type", None) or "ant").lower()
        if asset not in ("ant", "lzn"):
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
            asks = [
                o
                for o in self.state.orders.values()
                if o.order_type == OrderType.SELL and order_asset(o) == asset
            ]
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
            seller.wrt_balance += cost
            if asset == "lzn":
                buyer.lzn_balance += match_amount
                self.state.epoch_lzn_sold_volume += match_amount
            else:
                buyer.ant_balance += match_amount
                self.state.epoch_ant_sold_volume += match_amount
            ask.filled += match_amount
            bought += match_amount
            spent += cost
            self._push_trade_price(match_price, asset=asset)
            matched_txs.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "trade",
                    "buyer": buyer.address,
                    "seller": seller.address,
                    "price": match_price,
                    "amount": match_amount,
                    "asset_type": asset,
                    "timestamp": time.time(),
                }
            )
            if ask.filled >= ask.amount - 1e-12:
                del self.state.orders[ask.id]

        return matched_txs

    def _execute_market_sell(self, tx: Transaction) -> List[dict]:
        """Продажа до amount актива по лучшим bid; непроданный остаток возвращается."""
        seller = self.state.accounts.get(tx.sender)
        if not seller or seller.role != Role.PROVIDER:
            return []
        asset = (getattr(tx, "asset_type", None) or "ant").lower()
        if asset not in ("ant", "lzn"):
            return []
        total = float(tx.amount or 0)
        if total <= 1e-12:
            return []
        if asset == "lzn":
            if seller.lzn_balance + 1e-12 < total:
                return []
            seller.lzn_balance -= total
        else:
            if seller.ant_balance + 1e-12 < total:
                return []
            seller.ant_balance -= total

        rem = total
        matched_txs: List[dict] = []

        while rem > 1e-12:
            bids = [
                o
                for o in self.state.orders.values()
                if o.order_type == OrderType.BUY and order_asset(o) == asset
            ]
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
            if asset == "lzn":
                buyer.lzn_balance += match_amount
                self.state.epoch_lzn_sold_volume += match_amount
            else:
                buyer.ant_balance += match_amount
                self.state.epoch_ant_sold_volume += match_amount
            bid.filled += match_amount
            rem -= match_amount
            self._push_trade_price(match_price, asset=asset)
            matched_txs.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "trade",
                    "buyer": buyer.address,
                    "seller": seller.address,
                    "price": match_price,
                    "amount": match_amount,
                    "asset_type": asset,
                    "timestamp": time.time(),
                }
            )
            if bid.filled >= bid.amount - 1e-12:
                del self.state.orders[bid.id]

        if rem > 1e-12:
            if asset == "lzn":
                seller.lzn_balance += rem
            else:
                seller.ant_balance += rem
        return matched_txs

    def _network_lzn_total_validators(self) -> float:
        """§5.4: L_total = Σ L_i по всем валидаторам (активированный LZN)."""
        return sum(
            float(a.lzn_frozen_mining)
            for a in self.state.accounts.values()
            if a.role == Role.VALIDATOR
        )

    def _classify_declare(
        self, tx: Transaction
    ) -> Tuple[str, Optional[dict], str]:
        """Классификация declare/burn без списания ANT.

        Возвращает (action, candidate|None, reason):
        - ``ok`` — кандидат в пакет (dict с b/s/L_i/w_i);
        - ``wait`` — временно неисполним (мало ANT) → requeue в мемпул;
        - ``drop`` — структурно невалиден (роль, L_i=0, b+s>L_i, …) →
          выкинуть из мемпула с записью в блок/canon (не копить вечно).
        """
        sender = tx.sender
        if not sender:
            return "drop", None, "empty sender"
        acc = self.state.accounts.get(sender)
        if not acc:
            return "drop", None, "account not found"
        if acc.role != Role.VALIDATOR:
            return "drop", None, f"role={acc.role.value} (need validator)"
        if tx.tx_type == TransactionType.BURN:
            b, s = float(tx.amount or 0), 0.0
        elif tx.tx_type == TransactionType.DECLARE_PARTICIPATION:
            b, s = float(tx.amount or 0), float(tx.stake_amount or 0)
        else:
            return "drop", None, f"unsupported tx_type={tx.tx_type}"
        L_i = float(acc.lzn_frozen_mining)
        if L_i <= 0:
            return "drop", None, "L_i=0 (no activated LZN)"
        if b < 0 or s < 0:
            return "drop", None, "negative b or s"
        if b + s > L_i + 1e-9:
            return "drop", None, f"b+s={b + s} > L_i={L_i}"
        if acc.ant_balance + 1e-9 < b + s:
            return "wait", None, "insufficient ANT"
        w_i = s / L_i if L_i > 1e-18 else 0.0
        return (
            "ok",
            {"tx": tx, "sender": sender, "b": b, "s": s, "L_i": L_i, "w_i": w_i},
            "",
        )

    def _dry_declare_candidate(self, tx: Transaction) -> Optional[dict]:
        """Проверка declare/burn без списания ANT (состояние — на момент вызова)."""
        action, cand, _reason = self._classify_declare(tx)
        return cand if action == "ok" else None

    @staticmethod
    def _append_declare_dropped(
        txs_in_block: list, tx: Transaction, reason: str
    ) -> None:
        """Запись в ленту блока: declare отброшен из мемпула (не requeue)."""
        txs_in_block.append(
            {
                "tx_hash": getattr(tx, "tx_hash", "") or "",
                "tx_type": "declare_dropped",
                "sender": getattr(tx, "sender", None) or "",
                "details": reason,
                "timestamp": time.time(),
            }
        )

    def _finalize_declares_batch(
        self,
        declare_txs: List[Transaction],
        txs_in_block: list,
        participation: Dict[str, dict],
        requeue: List[Transaction],
    ) -> Tuple[float, int, float]:
        """
        §5.4: после локальных проверок — отсев по верхнему пределу (1−λ)·L_total
        (снять наименьшие w_i=s/L_i), затем потолок K с наибольшими w_i.
        Tie-breaker при равных w_i — лексикографически по адресу (нормативно, fork-safe).
        Исполнение: сжигается ТОЛЬКО b_i; s_i — ставка на высоту, проверяется по балансу
        (ANT ≥ b_i + s_i), но не списывается и возвращается валидатору после блока.

        Структурно невалидные declare (роль, L_i=0, b+s>L_i) — drop из мемпула
        с ``declare_dropped`` в блоке. Дубликаты sender: тихо оставляем **последний**
        (replace-by-sender / финальная воля на высоту) — без спама canon_log.
        Временно неисполнимые (мало ANT) и отсеянные λ/K — в requeue.
        Снимок ранжирования пишется в ``self.last_povb_competition`` (затем в блок).
        Возвращает (Σ b_i, число исполненных, L_total на момент отбора).
        """
        # Last-wins per sender (согласовано с mempool replace-by-sender).
        last_by_sender: Dict[str, Transaction] = {}
        sender_order: List[str] = []
        for tx in declare_txs:
            sid = tx.sender
            if not sid:
                self._append_declare_dropped(txs_in_block, tx, "empty sender")
                continue
            if sid not in last_by_sender:
                sender_order.append(sid)
            last_by_sender[sid] = tx
        ordered_unique: List[Transaction] = [last_by_sender[s] for s in sender_order]

        candidates: List[dict] = []
        waiting: List[Transaction] = []
        for tx in ordered_unique:
            action, cand, reason = self._classify_declare(tx)
            if action == "ok" and cand is not None:
                candidates.append(cand)
            elif action == "wait":
                waiting.append(tx)
            else:
                self._append_declare_dropped(txs_in_block, tx, reason)

        L_total = self._network_lzn_total_validators()
        cap = burn_ceiling(L_total)
        if L_total <= 1e-18:
            # Нет активации LZN в сети — wait + оставшиеся кандидаты ждут;
            # структурно битые уже drop'нуты выше.
            requeue.extend(waiting)
            requeue.extend(c["tx"] for c in candidates)
            snap = empty_povb_competition(L_total)
            snap["deferred_count"] = len(waiting) + len(candidates)
            snap["candidates_count"] = len(candidates)
            self.last_povb_competition = snap
            return 0.0, 0, L_total

        # λ-отсев по верхнему пределу (1−λ)·L_total: сортировка по возрастанию w_i,
        # при равенстве — лексикографически по адресу (детерминированный tie-breaker).
        work = list(candidates)
        work.sort(key=lambda x: (x["w_i"], x["sender"]))
        culled_lambda: List[dict] = []
        while work and sum(c["b"] for c in work) > cap + 1e-9:
            culled_lambda.append(work.pop(0))

        # Потолок K: наибольшие w_i, tie-breaker — адрес по возрастанию.
        culled_k: List[dict] = []
        if len(work) > MAX_ACTIVE_VALIDATORS_K:
            work.sort(key=lambda x: (-x["w_i"], x["sender"]))
            culled_k = work[MAX_ACTIVE_VALIDATORS_K:]
            work = work[:MAX_ACTIVE_VALIDATORS_K]

        selected_hashes = {c["tx"].tx_hash for c in work}
        B_applied = 0.0
        applied_count = 0

        # Не прошедшие λ/K — requeue (могут пройти в следующем блоке).
        for c in candidates:
            if c["tx"].tx_hash not in selected_hashes:
                requeue.append(c["tx"])

        requeue.extend(waiting)

        for c in work:
            tx = c["tx"]
            acc = self.state.accounts.get(c["sender"])
            if not acc or acc.ant_balance + 1e-9 < c["b"] + c["s"]:
                # Гонка балансов между классификацией и исполнением — wait.
                requeue.append(tx)
                continue
            # v2: сжигается только b_i; s_i остаётся на балансе (возвращаемая ставка на высоту).
            acc.ant_balance -= c["b"]
            self.state.current_epoch_burn += c["b"]
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

        applied_tx_hashes = {
            c["tx"].tx_hash for c in work if c["sender"] in participation
        }
        self.last_povb_competition = build_povb_competition(
            L_total=L_total,
            candidates=candidates,
            selected=work,
            culled_lambda=culled_lambda,
            culled_k=culled_k,
            B_selected=B_applied,
            deferred_count=len(waiting),
            applied_hashes=applied_tx_hashes,
        )
        return B_applied, applied_count, L_total

    def _epoch_boundary(self, block_height: int, txs_in_block: list):
        """§5.5 (5.0-sim / ruleset v2) — граница эпохи:

        1. Отменить открытые SELL-ордера ANT Поставщиков (эскроу → баланс) перед wipe.
           SELL LZN переживают границу (LZN не wipe). BUY-ордера (WRT) переживают.
        2. Сжечь весь ANT на счетах Поставщиков.
        3. Зачислить ANT_emit = sold×coeff (коридор спроса/потолка).
        4. Зачислить LZN_emit = sold_lzn×coeff_lzn, потолок LZN_MAX_EPOCH и headroom.
        """
        if block_height <= 0 or block_height % BLOCKS_PER_EPOCH != 0:
            return

        # Шаг 1: отмена только ANT SELL (эскроу вернуть перед wipe).
        for oid, order in sorted(self.state.orders.items()):
            if order.order_type != OrderType.SELL:
                continue
            if order_asset(order) != "ant":
                continue
            owner = self.state.accounts.get(order.owner)
            if owner is None:
                del self.state.orders[oid]
                continue
            rem = order.amount - order.filled
            owner.ant_balance += rem
            del self.state.orders[oid]
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": "epoch_order_cancel",
                "receiver": owner.address,
                "amount": rem,
                "asset_type": "ant",
                "details": (
                    f"§5.5 v2 step 1: отмена SELL-ордера ANT на границе эпохи, эскроу {rem:.4f} ANT "
                    f"возвращён перед wipe (height {block_height})"
                ),
                "timestamp": time.time(),
            })

        providers = [a for a in self.state.accounts.values() if a.role == Role.PROVIDER]
        providers.sort(key=lambda a: a.address)

        # Шаг 2: сжигание всего ANT Поставщиков.
        for p in providers:
            wiped = float(p.ant_balance)
            p.ant_balance = 0.0
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.EPOCH_ANT_WIPE.value,
                "receiver": p.address,
                "amount": wiped,
                "asset_type": "ant",
                "details": f"§5.5 v2 step 2: сжигание ANT Поставщика на границе эпохи (height {block_height})",
                "timestamp": time.time(),
            })

        sold_epoch = self.state.epoch_ant_sold_volume
        coeff = self.state.epoch_emission_coefficient
        sold_prev = self.state.epoch_ant_sold_last
        emission = sold_epoch * coeff

        L_total_epoch = self._network_lzn_total_validators()
        ant_max_epoch = float(BLOCKS_PER_EPOCH) * L_total_epoch
        epoch_burn_demand = ant_max_epoch * BURN_CAP_LAMBDA
        emission_market = emission
        emission = min(max(emission, epoch_burn_demand), ant_max_epoch)

        if sold_prev > SOLD_PREV_FLOOR:
            ratio = max(sold_epoch / sold_prev, 1e-6)
            target = coeff / ratio
            new_coeff = EMISSION_COEFF_ALPHA * target + (1.0 - EMISSION_COEFF_ALPHA) * coeff
        else:
            new_coeff = coeff
        self.state.epoch_emission_coefficient = max(COEFF_MIN, min(COEFF_MAX, new_coeff))
        self.state.epoch_ant_sold_last = sold_epoch
        self.state.epoch_ant_sold_volume = 0.0

        # Шаг 3: зачисление ANT.
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
                    f"§5.5 v2 step 3: эпохальная эмиссия ANT Поставщику "
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
                f"total_emit={emission:.4f} ANT (только кошельки Поставщиков, не валидаторы); "
                f"рынок sold×coeff={emission_market:.4f}; "
                f"спрос эпохи EpochBlocks×λ×L_total={epoch_burn_demand:.4f}; "
                f"ANT_max_epoch=EpochBlocks×L_total={ant_max_epoch:.4f}"
            ),
            "timestamp": time.time(),
        })

        # Шаг 4: производство LZN (§5.5 5.0-sim).
        lzn_sold = self.state.epoch_lzn_sold_volume
        lzn_coeff = self.state.epoch_lzn_emission_coefficient
        lzn_sold_prev = self.state.epoch_lzn_sold_last
        lzn_emission = lzn_sold * lzn_coeff
        headroom = lzn_mint_headroom(self.state.accounts)
        lzn_emission = min(max(lzn_emission, 0.0), float(LZN_MAX_EPOCH), headroom)
        if lzn_sold_prev > SOLD_PREV_FLOOR:
            ratio_l = max(lzn_sold / lzn_sold_prev, 1e-6)
            target_l = lzn_coeff / ratio_l
            new_lzn_coeff = EMISSION_COEFF_ALPHA * target_l + (1.0 - EMISSION_COEFF_ALPHA) * lzn_coeff
        else:
            new_lzn_coeff = lzn_coeff
        self.state.epoch_lzn_emission_coefficient = max(COEFF_MIN, min(COEFF_MAX, new_lzn_coeff))
        self.state.epoch_lzn_sold_last = lzn_sold
        self.state.epoch_lzn_sold_volume = 0.0

        per_lzn = (lzn_emission / len(providers)) if providers else 0.0
        for p in providers:
            if per_lzn <= 0:
                break
            p.lzn_balance += per_lzn
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.EPOCH_LZN_CREDIT.value,
                "receiver": p.address,
                "amount": per_lzn,
                "asset_type": "lzn",
                "details": (
                    f"§5.5 5.0-sim: эпохальная продукция LZN Поставщику "
                    f"(всего {lzn_emission:.4f}, headroom до потолка обращения)"
                ),
                "timestamp": time.time(),
            })
        txs_in_block.append({
            "tx_hash": uuid.uuid4().hex,
            "tx_type": TransactionType.EPOCH_EMISSION.value,
            "amount": lzn_emission,
            "asset_type": "lzn",
            "details": (
                f"§5.5 LZN epoch summary height {block_height}: providers={len(providers)}; "
                f"sold_lzn={lzn_sold:.4f}; coeff {lzn_coeff:.4f}→{self.state.epoch_lzn_emission_coefficient:.4f}; "
                f"total_emit={lzn_emission:.4f} LZN; LZN_max_epoch={LZN_MAX_EPOCH}; "
                f"headroom={headroom:.4f}"
            ),
            "timestamp": time.time(),
        })
        self.state.current_epoch_burn = 0.0

    def _begin_block(self, height: int, txs_in_block: list) -> None:
        """
        Предблок — аналог Cosmos SDK BeginBlock (до DeliverTx по пользовательским сообщениям).
        В ленте блока: маркер фазы, затем протокольная санитизация §4.1–4.2, MOA §3.3/§5.3.
        """
        proposer = select_proposer_for_height(
            height,
            getattr(self.state, "consensus_validator_set", []) or [],
            GENESIS_VALIDATOR_ADDR,
        )
        txs_in_block.append(
            {
                "tx_hash": uuid.uuid4().hex,
                "tx_type": "begin_block",
                "sender": "",
                "receiver": "",
                "details": (
                    f"BeginBlock (предблок) height={height}: proposer (§6.1 из ValidatorSet после предыдущего блока) = {proposer}; "
                    "до исполнения tx из мемпула; в полной цепи — evidence, слэшинг, выплаты за прошлый блок и т.п."
                ),
                "timestamp": time.time(),
            }
        )
        if height == SIM_BOOTSTRAP_INJECT_HEIGHT:
            self.state.inject_sim_bootstrap_cohort(txs_in_block)
            # После скачка L_total добрать declare в мемпул этой же высоты
            # (pre_block видел только seed). Иначе низ коридора λ·L_total не набрать.
            try:
                from core import auto_declare

                auto_declare.step_once(self.state, self)
            except Exception as e:  # pragma: no cover
                print(f"bootstrap auto_declare error: {e}")
        self._sanitize_ant_ineligible_roles(txs_in_block)
        # MOA §3.3/§5.3: check inactivity and strip roles
        sanctions = self.moa_tracker.check_inactivity(self.state, height)
        for s in sanctions:
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": "moa_sanction",
                "sender": s["address"],
                "amount": s["burned_ant"],
                "asset_type": "ant",
                "details": (
                    f"§3.3/§5.3 MOA sanction: inactive for {s['window']} blocks "
                    f"(last_active={s['last_active']}); role→citizen, ANT burned={s['burned_ant']:.4f}"
                ),
                "timestamp": time.time(),
            })

    def _end_block(
        self,
        block_height: int,
        txs_in_block: list,
        participation: Dict[str, dict],
        B_applied: float,
        cap: float,
        fee_tx_count: int,
    ) -> None:
        """
        Постблок — аналог Cosmos SDK EndBlock (после всех DeliverTx и внутриблочного матчинга).
        §5.1 базовая WRT, §5.4 делёж комиссий, §5.5 эпоха ANT; на полной цепи — ValidatorUpdates (§5.4).
        §5.1 базовая WRT, §5.4 делёж комиссий, §5.5 эпоха ANT; на полной цепи — ValidatorUpdates (§5.4).
        Коридор: λ·L_total ≤ Σb_i ≤ (1−λ)·L_total; при b_i = 0 доля дохода = 0 (§5.1).
        """
        reward_amount = 50.0
        eligible = [
            (addr, data["L_i"])
            for addr, data in participation.items()
            if data["b"] > 0 and data["L_i"] > 0
        ]
        if eligible:
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
                        f"§5.1 EndBlock: WRT block reward; "
                        f"λ·L≤Σb_i≤(1−λ)·L (B={B_applied:.4f}, cap={cap:.4f}); "
                        "доля ∝ L_i среди валидаторов с b_i>0"
                    ),
                    "timestamp": time.time(),
                }
            )
        else:
            txs_in_block.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "block_reward_skipped",
                    "receiver": "validators",
                    "amount": 0.0,
                    "asset_type": "wrt",
                    "details": (
                        f"§5.1/§5.4 v2 EndBlock: на высоте нет валидаторов с b_i>0 — "
                        f"базовая эмиссия WRT не начисляется (Σb_i={B_applied:.4f}, cap={cap:.4f}); "
                        "блок при этом валиден"
                    ),
                    "timestamp": time.time(),
                }
            )

        F = SIM_FEE_WRT_PER_LEDGER_TX * max(0, fee_tx_count)
        if B_applied > 1e-18 and F > 0:
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

    async def produce_block(self, quiet: bool = False):
        """
        Цикл блока по порядку, близкому к CometBFT + Cosmos SDK (каноническая цепь):
        BeginBlock → DeliverTx (мемпул по сообщениям) → пакет declare §5.4 →
        внутриблочный матчинг §5.2 → EndBlock (награды, комиссии, эпоха §5.5) →
        запись блока и персистентность.
        Интервал между блоками в эталоне — CANONICAL_BLOCK_INTERVAL_SEC (60 с);
        в симуляции определяется скоростью (sim_speed) / sim_block_interval_sec.
        quiet=True (batch-режим): без WS-дельты и print на каждый блок —
        агрегированную дельту шлёт _produce_batch_tick.
        """
        async with self._state_lock:
            await self._produce_block_body(quiet=quiet)

    def _snapshot_state(self) -> dict:
        """Снимок для отката блока.

        Account/Order — плоские pydantic-модели (скаляры), достаточно model_copy();
        списки blocks/tps/price_history — append-only (записи не мутируются),
        достаточно копии списка. Это на порядок дешевле copy.deepcopy и делает
        batch-режим высоких скоростей практичным.
        """
        return {
            "current_height": self.state.current_height,
            "accounts": {a: acc.model_copy() for a, acc in self.state.accounts.items()},
            "orders": {oid: o.model_copy() for oid, o in self.state.orders.items()},
            "mempool": list(self.state.mempool),
            "blocks": list(self.state.blocks),
            "tps_history": list(self.state.tps_history),
            "last_price": self.state.last_price,
            "price_history": list(self.state.price_history),
            "current_epoch_burn": self.state.current_epoch_burn,
            "epoch_ant_sold_volume": self.state.epoch_ant_sold_volume,
            "epoch_ant_sold_last": self.state.epoch_ant_sold_last,
            "epoch_emission_coefficient": self.state.epoch_emission_coefficient,
            "epoch_lzn_sold_volume": self.state.epoch_lzn_sold_volume,
            "epoch_lzn_sold_last": self.state.epoch_lzn_sold_last,
            "epoch_lzn_emission_coefficient": self.state.epoch_lzn_emission_coefficient,
            "last_lzn_price": self.state.last_lzn_price,
            "lzn_price_history": list(self.state.lzn_price_history),
            "last_block_wallet_delta": dict(self.state.last_block_wallet_delta),
            "consensus_validator_set": list(self.state.consensus_validator_set),
            "sim_bootstrap_injected": bool(self.state.sim_bootstrap_injected),
        }

    async def _produce_block_body(self, quiet: bool = False):
        # NetworkSim (Phase B): подтянуть готовые (≥quorum) tx из узлов в общий мемпул.
        # Если network не attached — режим fastpath, мемпул уже содержит всё.
        net = getattr(self.state, "network", None)
        if net is not None:
            try:
                ready = net.flush_to_global()
            except Exception as e:  # pragma: no cover
                ready = []
                print(f"NetworkSim flush failed: {e}")
            if ready:
                for tx in ready:
                    self.state.mempool_admit(tx)

        # §5.4: один declare/burn на sender — сжать хвост после сбоев/гонок.
        self.state.compact_declare_mempool()

        # Снэпшот состояния до блока: если блок отклонён кворумом консенсуса или
        # нарушены инварианты, откатываем все изменения и не коммитим высоту.
        # canon_log не откатываем — это локальная наблюдаемость причин отказа.
        snapshot = self._snapshot_state()
        # Snapshot для WS-дельты (JSON-форма аккаунтов/ордеров до блока);
        # в quiet-режиме не нужен — batch-тик считает дельту на всю пачку.
        accounts_before_json = {} if quiet else self.state.snapshot_accounts()
        orders_before_json = {} if quiet else self.state.snapshot_orders()
        balance_snap = {
            addr: (float(acc.wrt_balance), float(acc.ant_balance))
            for addr, acc in self.state.accounts.items()
        }
        txs_in_block = []
        block_height = self.state.current_height + 1
        self._begin_block(block_height, txs_in_block)
        # MEV: proposer reorders mempool (§5.2)
        if self.security_config.mev_reorder_enabled and self.state.mempool:
            proposer_addr = select_proposer_for_height(
                block_height,
                getattr(self.state, "consensus_validator_set", []) or [],
                GENESIS_VALIDATOR_ADDR,
            )
            self.state.mempool = reorder_mempool_mev(
                self.state.mempool, proposer_addr, self.security_config
            )
        declare_buffer: List[Transaction] = []
        participation: Dict[str, dict] = {}
        fee_ledger_ops = 0  # только успешные пользовательские tx (не begin_block / protocol_*)

        def reject_in_block(
            tx: Transaction,
            *,
            canon: str,
            title: str,
            detail: str,
            tx_type: str,
        ) -> None:
            """Записать отклонение и в канон-аудит, и в ленту блока (как DeliverTx reject)."""
            log_engine_skip(
                self.state,
                block_height=block_height,
                tx_hash=tx.tx_hash,
                tx_type=tx_type,
                canon=canon,
                title=title,
                detail=detail,
            )
            txs_in_block.append(
                {
                    "tx_hash": tx.tx_hash,
                    "tx_type": "deliver_tx_reject",
                    "orig_tx_type": (getattr(tx, "tx_type", None).value if getattr(tx, "tx_type", None) else str(tx_type)),
                    "sender": getattr(tx, "sender", None) or "",
                    "receiver": getattr(tx, "receiver", None) or "",
                    "canon": canon,
                    "title": title,
                    "details": detail,
                    "timestamp": time.time(),
                }
            )
    
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
                        elif asset == "lzn":
                            reject_in_block(
                                tx,
                                canon="§4.1 / §5.2",
                                title="Перевод LZN отклонён: только внутренний рынок",
                                detail="5.0-sim: LZN user-to-user — через книгу ордеров, не MsgSend",
                                tx_type="transfer",
                            )
                        else:
                            if asset not in ("wrt", "lzn"):
                                reject_in_block(
                                    tx,
                                    canon="§4.1",
                                    title="Перевод отклонён: по канону только WRT (не ANT/LZN MsgSend)",
                                    detail=f"asset={asset}",
                                    tx_type="transfer",
                                )
                            elif asset == "wrt":
                                reject_in_block(
                                    tx,
                                    canon="§4.1",
                                    title="Перевод WRT не исполнен",
                                    detail="Недостаточно баланса отправителя",
                                    tx_type="transfer",
                                )
            elif tx.tx_type == TransactionType.MINT:
                if tx.sender != SIM_TREASURY_ADDR:
                    reject_in_block(
                        tx,
                        canon="§4.1",
                        title="Mint отклонён: не из казны симуляции",
                        detail=f"sender={tx.sender}",
                        tx_type="mint",
                    )
                    continue
                treasury = self.state.accounts.get(SIM_TREASURY_ADDR)
                receiver_addr = tx.receiver
                if not treasury or not receiver_addr:
                    reject_in_block(
                        tx,
                        canon="§4.1",
                        title="Mint отклонён: нет казны или получателя",
                        detail=f"receiver={receiver_addr}",
                        tx_type="mint",
                    )
                    continue
                receiver = self.state.accounts.get(receiver_addr)
                if not receiver:
                    receiver = self.state.create_account(receiver_addr)
                asset = (getattr(tx, "asset_type", None) or "wrt").lower()
                amt = float(tx.amount or 0)
                if amt <= 0:
                    reject_in_block(
                        tx,
                        canon="§4.1",
                        title="Mint отклонён: amount ≤ 0",
                        detail=f"amount={tx.amount}",
                        tx_type="mint",
                    )
                    continue
                if asset == "ant" and receiver.role not in (Role.PROVIDER, Role.VALIDATOR):
                    reject_in_block(
                        tx,
                        canon="§4.1–4.2",
                        title="Mint ANT отклонён — соответствует канону",
                        detail=f"Получатель role={receiver.role.value}; ANT на кошельке только у Поставщика и Валидатора.",
                        tx_type="mint",
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
                else:
                    reject_in_block(
                        tx,
                        canon="§4.1",
                        title="Mint отклонён: недостаточно резерва казны",
                        detail=f"asset={asset} amount={amt}",
                        tx_type="mint",
                    )
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
                        if not check_supplier_cap(self.state, self.security_config):
                            reject_in_block(
                                tx,
                                canon="§7.2",
                                title="set_role→Provider отклонён: достигнут MaxActiveSuppliers",
                                detail=f"cap={self.security_config.max_active_suppliers}",
                                tx_type="set_role",
                            )
                        else:
                            rcv.role = Role.PROVIDER
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
                            role_applied = True
                    if not role_applied:
                        reject_in_block(
                            tx,
                            canon="§4.2 / §3.1",
                            title="set_role не применён",
                            detail=f"Запрошено {tx.role.value}: не выполнены ZKP/LZN или правила §4.2.",
                            tx_type="set_role",
                        )
            elif tx.tx_type == TransactionType.ZKP_VERIFY:
                zaddr = tx.sender or tx.receiver
                if not zaddr:
                    reject_in_block(
                        tx,
                        canon="§3.1",
                        title="ZKP verify отклонён: пустой адрес",
                        detail="sender/receiver empty",
                        tx_type="verify_zkp",
                    )
                else:
                    zacc = self.state.accounts.get(zaddr)
                    if not zacc:
                        reject_in_block(
                            tx,
                            canon="§3.1",
                            title="ZKP verify отклонён: неизвестный адрес",
                            detail=f"address={zaddr}",
                            tx_type="verify_zkp",
                        )
                    elif zacc.zkp_verified:
                        reject_in_block(
                            tx,
                            canon="§3.1",
                            title="ZKP verify пропущен: уже подтверждён",
                            detail=f"address={zaddr}",
                            tx_type="verify_zkp",
                        )
                    else:
                        # §3.1: nullifier uniqueness check
                        id_hash = getattr(tx, "details", None) or zaddr
                        if not self.zkp_registry.can_verify(id_hash):
                            reject_in_block(
                                tx,
                                canon="§3.1",
                                title="ZKP verify отклонён: identity nullifier уже зарегистрирован",
                                detail=f"identity_hash={id_hash}",
                                tx_type="verify_zkp",
                            )
                        else:
                            self.zkp_registry.register(id_hash)
                            zacc.zkp_verified = True
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
            elif tx.tx_type == TransactionType.ACTIVATE_LZN:
                act = self.state.accounts.get(tx.sender)
                if not act:
                    reject_in_block(
                        tx,
                        canon="§4.2",
                        title="activate_lzn отклонён: неизвестный адрес",
                        detail=f"sender={tx.sender}",
                        tx_type="activate_lzn",
                    )
                elif act.role != Role.VALIDATOR:
                    reject_in_block(
                        tx,
                        canon="§4.2",
                        title="activate_lzn отклонён: роль не Валидатор",
                        detail=f"role={act.role.value}",
                        tx_type="activate_lzn",
                    )
                else:
                    amt = float(tx.amount or 0)
                    if amt <= 0:
                        reject_in_block(
                            tx,
                            canon="§4.2",
                            title="activate_lzn отклонён: amount ≤ 0",
                            detail=f"amount={tx.amount}",
                            tx_type="activate_lzn",
                        )
                    elif act.lzn_balance + 1e-12 < amt:
                        reject_in_block(
                            tx,
                            canon="§4.2",
                            title="activate_lzn не исполнен: недостаточно ликвидного LZN",
                            detail=f"need={amt} have={act.lzn_balance}",
                            tx_type="activate_lzn",
                        )
                    elif act.lzn_frozen_mining + amt > float(LZN_MAX_FROZEN_PER_ADDRESS) + 1e-9:
                        reject_in_block(
                            tx,
                            canon="§4.2",
                            title="activate_lzn отклонён: превышен потолок активированных LZN",
                            detail=f"cap={LZN_MAX_FROZEN_PER_ADDRESS} current={act.lzn_frozen_mining} add={amt}",
                            tx_type="activate_lzn",
                        )
                    else:
                        act.lzn_balance -= amt
                        act.lzn_frozen_mining += amt
                        txs_in_block.append(tx.model_dump(mode="json"))
                        fee_ledger_ops += 1
            elif tx.tx_type == TransactionType.CREATE_ORDER:
                owner = self.state.accounts.get(tx.sender)
                if not owner:
                    reject_in_block(
                        tx,
                        canon="§5.2",
                        title="Ордер отклонён: неизвестный адрес",
                        detail=f"sender={tx.sender}",
                        tx_type="create_order",
                    )
                elif owner.role not in (Role.PROVIDER, Role.VALIDATOR):
                    reject_in_block(
                        tx,
                        canon="§4.2 / §5.2",
                        title="Ордер отклонён: роль не участвует в рынке",
                        detail=f"role={owner.role.value}",
                        tx_type="create_order",
                    )
                else:
                    amt = float(tx.amount or 0)
                    asset = (getattr(tx, "asset_type", None) or "ant").lower()
                    if asset not in ("ant", "lzn"):
                        reject_in_block(
                            tx,
                            canon="§5.2",
                            title="Ордер отклонён: asset должен быть ant или lzn",
                            detail=f"asset={asset}",
                            tx_type="create_order",
                        )
                        continue
                    if amt <= 0:
                        reject_in_block(
                            tx,
                            canon="§5.2",
                            title="Ордер отклонён: amount ≤ 0",
                            detail=f"amount={tx.amount}",
                            tx_type="create_order",
                        )
                        continue
                    if getattr(tx, "market", False):
                        mtr: List[dict] = []
                        if tx.order_type == OrderType.BUY:
                            mtr = self._execute_market_buy(tx)
                        elif tx.order_type == OrderType.SELL:
                            mtr = self._execute_market_sell(tx)
                        else:
                            reject_in_block(
                                tx,
                                canon="§5.2",
                                title="Рыночная заявка отклонена: неизвестная сторона",
                                detail=f"order_type={getattr(tx, 'order_type', None)}",
                                tx_type="create_order",
                            )
                        if mtr:
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
                            txs_in_block.extend(mtr)
                            fee_ledger_ops += len(mtr)
                        else:
                            reject_in_block(
                                tx,
                                canon="§5.2",
                                title="Рыночная заявка без исполнения",
                                detail="Нет подходящей ликвидности в книге или исчерпан лимит WRT/актива, или роль не соответствует стороне.",
                                tx_type="create_order",
                            )
                    else:
                        price = float(tx.price or 0)
                        if price <= 0:
                            reject_in_block(
                                tx,
                                canon="§5.2",
                                title="Лимитный ордер отклонён: price ≤ 0",
                                detail=f"price={tx.price}",
                                tx_type="create_order",
                            )
                            continue
                        if tx.order_type == OrderType.BUY:
                            if owner.role != Role.VALIDATOR:
                                reject_in_block(
                                    tx,
                                    canon="§5.2",
                                    title="BUY ордер отклонён: покупает только Валидатор",
                                    detail=f"role={owner.role.value}",
                                    tx_type="create_order",
                                )
                                continue
                            cost = price * amt
                            if owner.wrt_balance + 1e-12 < cost:
                                reject_in_block(
                                    tx,
                                    canon="§5.2",
                                    title="BUY ордер не принят: недостаточно WRT для эскроу",
                                    detail=f"need={cost} have={owner.wrt_balance}",
                                    tx_type="create_order",
                                )
                                continue
                            owner.wrt_balance -= cost
                            order = Order(
                                id=tx.tx_hash,
                                owner=tx.sender,
                                order_type=tx.order_type,
                                price=price,
                                amount=amt,
                                timestamp=tx.timestamp,
                                asset=asset,
                            )
                            self.state.orders[tx.tx_hash] = order
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
                        elif tx.order_type == OrderType.SELL:
                            if owner.role != Role.PROVIDER:
                                reject_in_block(
                                    tx,
                                    canon="§5.2",
                                    title="SELL ордер отклонён: продаёт только Поставщик",
                                    detail=f"role={owner.role.value}",
                                    tx_type="create_order",
                                )
                                continue
                            if asset == "lzn":
                                bal = owner.lzn_balance
                                label = "LZN"
                            else:
                                bal = owner.ant_balance
                                label = "ANT"
                            if bal + 1e-12 < amt:
                                reject_in_block(
                                    tx,
                                    canon="§5.2",
                                    title=f"SELL ордер не принят: недостаточно {label} для эскроу",
                                    detail=f"need={amt} have={bal}",
                                    tx_type="create_order",
                                )
                                continue
                            if asset == "lzn":
                                owner.lzn_balance -= amt
                            else:
                                owner.ant_balance -= amt
                            order = Order(
                                id=tx.tx_hash,
                                owner=tx.sender,
                                order_type=tx.order_type,
                                price=price,
                                amount=amt,
                                timestamp=tx.timestamp,
                                asset=asset,
                            )
                            self.state.orders[tx.tx_hash] = order
                            txs_in_block.append(tx.model_dump(mode="json"))
                            fee_ledger_ops += 1
                        else:
                            reject_in_block(
                                tx,
                                canon="§5.2",
                                title="Лимитный ордер отклонён: неизвестная сторона",
                                detail=f"order_type={getattr(tx, 'order_type', None)}",
                                tx_type="create_order",
                            )
            elif tx.tx_type == TransactionType.CANCEL_ORDER:
                order = self.state.orders.get(tx.order_id)
                if not order:
                    reject_in_block(
                        tx,
                        canon="§5.2",
                        title="Отмена ордера отклонена: ордер не найден",
                        detail=f"order_id={tx.order_id}",
                        tx_type="cancel_order",
                    )
                elif order.owner != tx.sender:
                    reject_in_block(
                        tx,
                        canon="§5.2",
                        title="Отмена ордера отклонена: не владелец",
                        detail=f"owner={order.owner} sender={tx.sender}",
                        tx_type="cancel_order",
                    )
                else:
                    owner = self.state.accounts.get(tx.sender)
                    if not owner:
                        reject_in_block(
                            tx,
                            canon="§5.2",
                            title="Отмена ордера отклонена: аккаунт владельца не найден",
                            detail=f"sender={tx.sender}",
                            tx_type="cancel_order",
                        )
                    else:
                        rem = order.amount - order.filled
                        if order.order_type == OrderType.BUY:
                            owner.wrt_balance += order.price * rem
                        elif order_asset(order) == "lzn":
                            owner.lzn_balance += rem
                        else:
                            owner.ant_balance += rem
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
        B_applied, _ndecl, L_at_declare = self._finalize_declares_batch(
            declare_buffer, txs_in_block, participation, requeue_declare
        )
        fee_tx_count = fee_ledger_ops + _ndecl
    
        trades = self._match_orders()
        txs_in_block.extend(trades)
        fee_tx_count += len(trades)
    
        floor = burn_floor(L_at_declare)
        cap = burn_ceiling(L_at_declare)

        # Верх (1−λ)·L уже применён в _finalize_declares_batch.
        # Низ λ·L — по L_total на момент declare (не после activate_lzn в том же блоке).
        if not check_min_burn_threshold(
            B_applied, L_at_declare, BURN_CAP_LAMBDA, self.security_config
        ):
            self.state.canon_log.push(
                source="engine",
                status="reject",
                category="consensus",
                canon="§5.4",
                title="Блок отклонён: сжигание ANT не набрано (min-burn)",
                detail=(
                    f"B={B_applied:.4f}, mode={self.security_config.min_burn_mode}, "
                    f"floor=λ·L={floor:.4f}, ceiling=(1−λ)·L={cap:.4f}"
                ),
                tx_hash="",
                block_height=block_height,
            )
            self._rollback_to(snapshot)
            raise RuntimeError("Block rejected: min-burn threshold not met (§5.4 corridor)")

        self._end_block(block_height, txs_in_block, participation, B_applied, cap, fee_tx_count)

        # §6.1 ValidatorSet update from declare participation (w_i = s_i/L_i)
        new_valset = recalculate_validator_set(participation, self.state, self.security_config)
        if new_valset:
            self.state.consensus_validator_set = new_valset

        # MOA: record activity for all tx senders in this block
        for tx_entry in txs_in_block:
            sender = tx_entry.get("sender") if isinstance(tx_entry, dict) else None
            if sender and sender != SIM_TREASURY_ADDR:
                self.moa_tracker.record_activity(sender, block_height)

        audit_block_ledger(self.state, txs_in_block, block_height)

        # --- ValidatorSet (§6.3, §5.4): все записи должны быть действующими валидаторами ---
        cvs = getattr(self.state, "consensus_validator_set", None) or []
        if not cvs:
            self.state.canon_log.push(
                source="engine",
                status="reject",
                category="consensus",
                canon="§6.1",
                title="Блок отклонён: пустой ValidatorSet",
                detail="consensus_validator_set пуст — нет базы для выбора пропозера (§6.1).",
                tx_hash="",
                block_height=block_height,
            )
            self._rollback_to(snapshot)
            raise RuntimeError("Block rejected: empty consensus validator set")
        for entry in cvs:
            addr = entry.get("address") if isinstance(entry, dict) else None
            if not addr:
                continue
            pacc = self.state.accounts.get(addr)
            if not pacc or pacc.role != Role.VALIDATOR:
                self.state.canon_log.push(
                    source="engine",
                    status="reject",
                    category="consensus",
                    canon="§6.1",
                    title="Блок отклонён: адрес в ValidatorSet не является валидатором",
                    detail=f"address={addr!r} отсутствует или роль ≠ validator.",
                    tx_hash="",
                    block_height=block_height,
                )
                self._rollback_to(snapshot)
                raise RuntimeError("Block rejected: consensus set member not validator")
        for acc in self.state.accounts.values():
            if acc.wrt_balance < -1e-9 or acc.lzn_balance < -1e-9 or acc.lzn_frozen_mining < -1e-9 or acc.ant_balance < -1e-9:
                self.state.canon_log.push(
                    source="engine",
                    status="reject",
                    category="consensus",
                    canon="инварианты",
                    title="Блок отклонён: отрицательный баланс",
                    detail=f"addr={acc.address} wrt={acc.wrt_balance} lzn={acc.lzn_balance} frozen={acc.lzn_frozen_mining} ant={acc.ant_balance}",
                    tx_hash="",
                    block_height=block_height,
                )
                self._rollback_to(snapshot)
                raise RuntimeError("Block rejected: negative balance invariant")
    
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
        
        validator_set = getattr(self.state, "consensus_validator_set", []) or []
        base_height = self.state.current_height + 1
        proposer = select_proposer_for_height(
            base_height,
            validator_set,
            GENESIS_VALIDATOR_ADDR,
        )

        consensus_log: List[dict] = []
        new_evidence: List[Evidence] = []
        if self.is_consensus_enabled and validator_set:
            def _proposer_for_round(r: int) -> str:
                return select_proposer_for_height(
                    base_height + r,
                    validator_set,
                    GENESIS_VALIDATOR_ADDR,
                )

            reports, new_evidence = run_consensus(
                height=base_height,
                validators=validator_set,
                proposer_for_round_fn=_proposer_for_round,
                fault_model=self.consensus_fault_model,
                max_rounds=4,
            )
            consensus_log = [r.as_block_log_entry() for r in reports]
            # Evidence фиксируем сразу: ушло в журнал нарушений, даже если блок
            # будет отклонён по кворуму (нарушители уже подписали double_sign).
            if new_evidence:
                self._evidence_log.extend(new_evidence)
            last = reports[-1] if reports else None
            if last is not None:
                proposer = last.proposer
                if last.decision != Decision.COMMIT:
                    self.state.canon_log.push(
                        source="consensus",
                        status="reject",
                        category="consensus",
                        canon="§6.1",
                        title=f"Блок h={base_height} отклонён кворумом",
                        detail=(
                            f"rounds={len(reports)} decision={last.decision.value} "
                            f"power_block_pc={last.power_block_pc:.2f}/{last.power_total:.2f} "
                            f"evidence={len(new_evidence)}"
                        ),
                        block_height=base_height,
                    )
                    # v2: отклонённый кворумом блок не оставляет следов в состоянии —
                    # все мутации DeliverTx/EndBlock откатываются (evidence сохранён выше).
                    self._rollback_to(snapshot)
                    raise RuntimeError("Block rejected by consensus quorum")
            if new_evidence:
                self._apply_slashing(new_evidence, txs_in_block)

        next_h = self.state.current_height + 1
        sim_ts = self.state.sim_timestamp(for_height=next_h)
        competition = getattr(self, "last_povb_competition", None) or empty_povb_competition(
            L_at_declare
        )
        block = {
            "height": next_h,
            "hash": block_hash,
            "timestamp": sim_ts,
            "proposer": proposer,
            "transactions": txs_in_block,
            "tx_count": len(txs_in_block),
            "competition": competition,
        }
        if consensus_log:
            block["consensus_round_log"] = consensus_log
        if new_evidence:
            block["evidence"] = [
                {"address": ev.address, "kind": ev.kind, "detail": ev.detail}
                for ev in new_evidence
            ]
        
        # Calculate TPS
        tps = len(txs_in_block) / self.block_time
        self.state.tps_history.append({
            "time": time.strftime("%H:%M:%S", time.localtime(sim_ts)),
            "tps": round(tps, 2)
        })
        if len(self.state.tps_history) > 50:
            self.state.tps_history.pop(0)
        
        self.state.mempool = requeue_declare
        self.state.add_block(block)
        self.state.note_block_committed(next_h)
        if participation:
            self.state.consensus_validator_set = consensus_validator_set_from_participation(participation)
        else:
            # §6.1: без declare набор нельзя оставлять замороженным — иначе пропозер
            # фиксируется навсегда. Пересобираем по активированному LZN («оборудованию»).
            self.state.consensus_validator_set = validator_set_from_activated_lzn(self.state.accounts)

        # Persistence — снимок state.json только каждые N блоков (settings.snapshot_every_n_blocks);
        # blocks.jsonl уже дописан в add_block().
        if self.state.should_snapshot_now():
            try:
                self.state.save_state()
                self.state.mark_snapshot_taken()
            except OSError as e:
                print(f"Warning: save_state failed after block {block['height']}: {e}")

        if quiet:
            # Batch-режим: WS-дельту на пачку и лог отправит _produce_batch_tick.
            return

        # Delta WS: только изменённые аккаунты/ордера (+ компактный summary блока).
        delta = self.state.compute_delta(accounts_before_json, orders_before_json)
        await self.ws_manager.broadcast(
            {
                "type": "block_delta",
                "data": {
                    "block": block,
                    "delta": delta,
                    "market": self.state.get_orderbook(),
                    "consensus_validators": list(self.state.consensus_validator_set),
                    "next_proposer": select_proposer_for_height(
                        self.state.current_height + 1,
                        self.state.consensus_validator_set,
                        GENESIS_VALIDATOR_ADDR,
                    ),
                    "current_epoch_burn": self.state.current_epoch_burn,
                    "epoch_ant_sold_volume": self.state.epoch_ant_sold_volume,
                    "epoch_emission_coefficient": self.state.epoch_emission_coefficient,
                    "last_block_wallet_delta": self.state.last_block_wallet_delta,
                    "mempool_size": len(self.state.mempool),
                    "tps_history_tail": self.state.tps_history[-30:],
                    "block_time": self.block_time,
                    "sim_speed": self.speed,
                    "effective_speed": self.effective_speed,
                },
            }
        )

        print(f"Produced block {block['height']} with {len(txs_in_block)} txs")
