import json
import os
import time
from typing import Dict, List, Optional

from core.canon_log import CanonLogBuffer
from core.ledger import BlockLedger, CanonLogLedger, index_block_txs
from core.models import Account, Order, OrderType, Role, Transaction, TransactionType
from core.settings import get_settings, resolve_data_dir

# Эпоха ANT: технические блоки (§5.5, §7.2 п.11) — эталон 1 блок/мин × 7 суток
BLOCKS_PER_EPOCH = 7 * 24 * 60  # 10080
# Интервал между блоками в эталонной цепи (1 тех. блок ≈ 1 мин). В симуляции хранится в state и может быть иным для тестов.
CANONICAL_BLOCK_INTERVAL_SEC = 60.0
# Скорость симуляции: симулированных секунд за 1 реальную секунду
# (1 блок = CANONICAL_BLOCK_INTERVAL_SEC симулированного времени).
# 1.0 = реальное время (блок раз в 60 с); максимум — 1 реальная секунда = 1 неделя
# симуляции (604800×, т.е. 10080 блоков/с, batch-режим движка).
SIM_SPEED_MIN = CANONICAL_BLOCK_INTERVAL_SEC / 300.0  # 0.2× — блок раз в 300 с
SIM_SPEED_MAX = 604_800.0  # 1 с = 1 неделя
MIN_SIM_BLOCK_INTERVAL_SEC = CANONICAL_BLOCK_INTERVAL_SEC / SIM_SPEED_MAX  # ≈ 9.92e-5 с
MAX_SIM_BLOCK_INTERVAL_SEC = 300.0
# §5.4 / §7.2 п.9 (ruleset v2): λ — только ВЕРХНИЙ предел сжигания за блок (Σb_i ≤ λ·L_total).
# Минимальный порог Σb_i ≥ λ·L_total из канона v4.20 в v2 удалён: liveness цепи
# не зависит от рынка ANT (см. docs/CANON_PROBLEMS.md §1 и simulation/docs/V2_RULESET.md).
BURN_CAP_LAMBDA = 1.0 / 3.0
# §4.2: не более ⌊эталон/3⌋ активированных LZN на адрес (целые токены; эталон 10_000 → 3333).
# Genesis-валидатор: 6667 активированных (остаток ликвидности = 10_000 − 6667).
LZN_TOTAL_SUPPLY_REF = 10_000
LZN_GENESIS_ACTIVATED = 6_667
LZN_MAX_FROZEN_PER_ADDRESS = LZN_TOTAL_SUPPLY_REF // 3

# §6.3: два фиксированных genesis-адреса, без ZKP, без роли «Супервизор»
GENESIS_VALIDATOR_ADDR = "volnix1gval0validator0genesis0"
# Ruleset v2 (решение проблемы «курица-яйцо» §6.3, CANON_PROBLEMS §2): genesis-валидатор
# получает стартовый ANT на одну полную эпоху сжигания по верхнему пределу λ:
# ANT_val = EpochBlocks × λ × L_genesis. Дальше ANT покупается на внутреннем рынке.
GENESIS_VALIDATOR_ANT_BALANCE = float(BLOCKS_PER_EPOCH) * BURN_CAP_LAMBDA * LZN_GENESIS_ACTIVATED
GENESIS_PROVIDER_ADDR = "volnix1gprov0provider00genesis0"
# Вне цепочки: резерв симулятора для минта оператором (не в genesis-блоке)
SIM_TREASURY_ADDR = "sim_treasury_reserve"
# Исторический ключ казны в старых state.json — миграция в load_state
SIM_TREASURY_ADDR_LEGACY = "sim_treasury_reserve_godmode"

# Сколько последних тиков отдаём в WebSocket/API (полная история остаётся в памяти и price_history.jsonl).
MARKET_HISTORY_WS_MAX = 30_000
# Верхняя граница tx мемпула в state.json (защита от раздувания файла)
MEMPOOL_PERSIST_MAX = 1_000


def account_total_lzn(acc: Account) -> float:
    return acc.lzn_balance + acc.lzn_frozen_mining


def eligible_for_validator_role(address: str, acc: Account) -> bool:
    """§3.1 + §6.3: genesis-валидатор без ZKP в цепочке; остальные — ZKP и LZN."""
    if address == GENESIS_VALIDATOR_ADDR:
        return True
    return acc.zkp_verified and account_total_lzn(acc) > 0


def _deserialize_mempool(raw: object) -> List[Transaction]:
    if not isinstance(raw, list):
        return []
    out: List[Transaction] = []
    for item in raw[:MEMPOOL_PERSIST_MAX]:
        if not isinstance(item, dict):
            continue
        try:
            out.append(Transaction.model_validate(item))
        except Exception:
            continue
    return out


def eligible_for_provider_role(address: str, acc: Account) -> bool:
    """Поставщик: требует ZKP; LZN не требуется (genesis-поставщик — исключение §6.3)."""
    if address == GENESIS_PROVIDER_ADDR:
        return True
    return bool(acc.zkp_verified)


def default_consensus_validator_set(accounts: Dict[str, Account]) -> List[dict]:
    """§6.3(5): до первого EndBlock по declare — единственный участник ValidatorSet = genesis-валидатор; power ≈ L_i."""
    gv = accounts.get(GENESIS_VALIDATOR_ADDR)
    if gv and gv.role == Role.VALIDATOR:
        p = max(1e-12, float(gv.lzn_frozen_mining))
        return [{"address": GENESIS_VALIDATOR_ADDR, "power": p}]
    return []


def select_proposer_for_height(height: int, validator_set: List[dict], fallback_addr: str) -> str:
    """
    Пропозер блока height по текущему ValidatorSet (после предыдущего блока / genesis).
    Вес power — аналог доли в CometBFT (в каноне — из s_i / L_i после §5.4); порядок адресов детерминирован.
    При одном валидаторе (первый блок после genesis) — он же и пропозер, как в §6.3.
    """
    if not validator_set:
        return fallback_addr
    items = sorted(
        (
            {"address": str(x["address"]), "power": max(0.0, float(x.get("power", 0.0)))}
            for x in validator_set
            if isinstance(x, dict) and x.get("address")
        ),
        key=lambda x: x["address"],
    )
    if not items:
        return fallback_addr
    if len(items) == 1:
        return items[0]["address"]
    weights = [max(1, int(round(it["power"] * 1_000_000))) for it in items]
    total = sum(weights)
    if total <= 0:
        return items[0]["address"]
    r = (height - 1) % total
    c = 0
    for it, w in zip(items, weights):
        c += w
        if r < c:
            return it["address"]
    return items[-1]["address"]


def consensus_validator_set_from_participation(participation: Dict[str, dict]) -> List[dict]:
    """После успешного блока: ValidatorSet для следующих высот из исполненных declare (§5.4 EndBlocker)."""
    out: List[dict] = []
    for addr in sorted(participation.keys()):
        d = participation[addr]
        s = float(d.get("s", 0) or 0)
        w = float(d.get("w_i", 0) or 0)
        L = float(d.get("L_i", 0) or 0)
        power = max(1e-12, s, w * L)
        out.append({"address": addr, "power": power})
    return out


class StateManager:
    def __init__(self, data_dir: Optional[str] = None):
        settings = get_settings()
        self.data_dir = data_dir or resolve_data_dir()
        self._active_state_path = os.path.join(self.data_dir, "state.json")
        self.current_height = 0
        self.accounts: Dict[str, Account] = {}
        self.mempool: List[Transaction] = []
        self.blocks: List[dict] = []
        self.orders: Dict[str, Order] = {}
        self.last_price = 0.0
        self.price_history: List[dict] = []
        self.tps_history: List[dict] = []
        self.current_epoch_burn = 0.0
        self.epoch_ant_sold_volume = 0.0  # §5.5: объём ANT, проданного поставщиками за текущую эпоху
        self.epoch_ant_sold_last = 0.0  # продажи за предыдущую эпоху (для ratio коэффициента)
        self.epoch_emission_coefficient = 1.0  # §5.5: genesis = 1, границы 0.75–1.5
        self.canon_log = CanonLogBuffer(maxlen=settings.canon_log_capacity)
        # После каждого блока: изменение балансов за блок (для бота — цена vs фактические WRT/ANT)
        self.last_block_wallet_delta: Dict[str, Dict[str, float]] = {}
        self.sim_block_interval_sec: float = CANONICAL_BLOCK_INTERVAL_SEC
        # Скорость симуляции (сим. секунд за реальную): 1.0 = реальное время.
        self.sim_speed: float = 1.0
        # ValidatorSet для выбора пропозера (§6.1, §6.3): после genesis — из EndBlocker/declare §5.4
        self.consensus_validator_set: List[dict] = []

        # Append-only ledger (Этап 2): blocks.jsonl + canon_log.jsonl + tx-индекс.
        self._blocks_in_memory_max = max(50, int(settings.blocks_in_memory))
        self._snapshot_every_n = max(1, int(settings.snapshot_every_n_blocks))
        self._blocks_since_snapshot = 0
        # Троттлинг snapshot по времени: на высоких скоростях N блоков пролетают
        # за доли секунды — полный дамп state.json не чаще, чем раз в 2 с.
        self._last_snapshot_ts = 0.0
        self.block_ledger = BlockLedger(os.path.join(self.data_dir, "blocks.jsonl"))
        if settings.canon_log_persist:
            self._canon_ledger = CanonLogLedger(os.path.join(self.data_dir, "canon_log.jsonl"))
            self.canon_log.attach_ledger(self._canon_ledger)
        else:
            self._canon_ledger = None
        # tx_hash → {height, tx_idx, tx_type, sender, receiver}
        self.tx_index: Dict[str, dict] = {}
        # addr → [tx_hash, ...] (хронологически)
        self.account_tx_index: Dict[str, List[str]] = {}

        # NetworkSim (Phase B): per-node mempool + gossip. Attached in lifespan.
        # None → классический режим (одиночный self.mempool).
        self.network = None

    @staticmethod
    def _price_history_jsonl_path(state_json_path: str) -> str:
        d = os.path.dirname(os.path.abspath(state_json_path))
        return os.path.join(d if d else ".", "price_history.jsonl")

    @staticmethod
    def _read_price_history_jsonl(jsonl_path: str) -> List[dict]:
        if not os.path.exists(jsonl_path):
            return []
        out: List[dict] = []
        with open(jsonl_path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    out.append(json.loads(line))
                except json.JSONDecodeError:
                    continue
        return out

    @staticmethod
    def _append_price_history_jsonl(jsonl_path: str, row: dict) -> None:
        d = os.path.dirname(jsonl_path)
        if d:
            os.makedirs(d, exist_ok=True)
        with open(jsonl_path, "a", encoding="utf-8") as f:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")
            f.flush()

    @staticmethod
    def _rewrite_price_history_jsonl(jsonl_path: str, rows: List[dict]) -> None:
        d = os.path.dirname(jsonl_path)
        if d:
            os.makedirs(d, exist_ok=True)
        tmp = jsonl_path + ".tmp"
        with open(tmp, "w", encoding="utf-8") as f:
            for r in rows:
                f.write(json.dumps(r, ensure_ascii=False) + "\n")
        os.replace(tmp, jsonl_path)

    def init_genesis(self):
        import uuid

        ts = time.time()
        l_total_genesis = LZN_GENESIS_ACTIVATED
        ant_genesis = float(BLOCKS_PER_EPOCH) * l_total_genesis  # §5.5: ANT_genesis = EpochBlocks × L_total_genesis

        gv = self.create_account(GENESIS_VALIDATOR_ADDR)
        gv.role = Role.VALIDATOR
        gv.wrt_balance = 0.0
        gv.lzn_balance = float(LZN_TOTAL_SUPPLY_REF - LZN_GENESIS_ACTIVATED)
        gv.lzn_frozen_mining = float(LZN_GENESIS_ACTIVATED)
        gv.ant_balance = float(GENESIS_VALIDATOR_ANT_BALANCE)
        gv.zkp_verified = True  # §6.3: genesis без ончейн ZKP, в UI — «как верифицирован»

        gp = self.create_account(GENESIS_PROVIDER_ADDR)
        gp.role = Role.PROVIDER
        gp.wrt_balance = 0.0
        gp.lzn_balance = 0.0
        gp.ant_balance = ant_genesis
        gp.zkp_verified = True

        # Резерв симуляции (не отражать как genesis-актив в §6.3)
        tr = self.create_account(SIM_TREASURY_ADDR)
        tr.role = Role.CITIZEN
        tr.wrt_balance = 1_000_000.0
        tr.lzn_balance = 1_000_000.0
        tr.ant_balance = 1_000_000.0

        txs = [
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_MESSAGE,
                details=(
                    "Volnix Protocol §6.3 — genesis: два кошелька (Поставщик + Валидатор), без ZKP, без Супервизора. "
                    f"§6.3(5): ValidatorSet в genesis — genesis-Валидатор как единственный участник; пропозер блока 1 — "
                    f"по правилам §6.1 из этого набора. EpochBlocks={BLOCKS_PER_EPOCH}; ANT_genesis=EpochBlocks×L_total={ant_genesis:.0f}; "
                    f"симуляция: genesis-Валидатор — {GENESIS_VALIDATOR_ANT_BALANCE:.0f} ANT."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_VALIDATOR_LZN,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=float(LZN_TOTAL_SUPPLY_REF),
                asset_type="lzn",
                details="§6.3(3): 10 000 LZN на genesis-Валидатора (полная одноразовая эмиссия лицензий).",
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_LZN_ACTIVATE,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=float(LZN_GENESIS_ACTIVATED),
                asset_type="lzn",
                details=(
                    f"§6.3(3)+§4.2: {LZN_GENESIS_ACTIVATED} LZN активировано (генезис-исключение); "
                    f"{LZN_TOTAL_SUPPLY_REF - LZN_GENESIS_ACTIVATED} ликвидных; далее потолок ⌊{LZN_TOTAL_SUPPLY_REF}/3⌋ = {LZN_MAX_FROZEN_PER_ADDRESS} на адрес."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_VALIDATOR_ANT,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=float(GENESIS_VALIDATOR_ANT_BALANCE),
                asset_type="ant",
                details=(
                    f"Ruleset v2: стартовый ANT на genesis-Валидаторе = {GENESIS_VALIDATOR_ANT_BALANCE:.0f} "
                    f"(EpochBlocks × λ × L_genesis = {BLOCKS_PER_EPOCH} × {BURN_CAP_LAMBDA:.4f} × {LZN_GENESIS_ACTIVATED}; "
                    "бюджет сжигания §5.4 на первую эпоху — решение genesis «курица-яйцо»)."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_PROVIDER_ANT,
                receiver=GENESIS_PROVIDER_ADDR,
                amount=ant_genesis,
                asset_type="ant",
                details=(
                    f"§6.3(4)+§5.5: стартовая ANT на genesis-Поставщика = {ant_genesis:.0f} "
                    f"(EpochBlocks×L_total_genesis)."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
        ]

        genesis_block = {
            "height": 0,
            "hash": "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
            "timestamp": ts,
            "transactions": txs,
            "tx_count": len(txs),
        }
        # init_genesis вызывается на чистом StateManager (current_height=0);
        # append блока вручную + индексация, без инкремента высоты.
        self.blocks.append(genesis_block)
        try:
            self.block_ledger.append_block(genesis_block)
        except OSError:
            pass
        self._index_block(genesis_block)
        self.consensus_validator_set = default_consensus_validator_set(self.accounts)

    def create_account(self, address: str) -> Account:
        if address not in self.accounts:
            self.accounts[address] = Account(address=address)
            net = getattr(self, "network", None)
            if net is not None:
                try:
                    net.register_address(address)
                except Exception:
                    pass
        return self.accounts[address]

    def list_open_orders_for_address(self, address: str) -> List[dict]:
        return [
            o.model_dump(mode="json")
            for o in self.orders.values()
            if o.owner == address
        ]

    def get_orderbook(self):
        bids = [o.model_dump(mode="json") for o in self.orders.values() if o.order_type == OrderType.BUY]
        asks = [o.model_dump(mode="json") for o in self.orders.values() if o.order_type == OrderType.SELL]
        
        # Sort bids descending (highest price first), asks ascending (lowest price first)
        bids.sort(key=lambda x: (-x["price"], x["timestamp"]))
        asks.sort(key=lambda x: (x["price"], x["timestamp"]))
        
        hist = self.price_history
        if len(hist) > MARKET_HISTORY_WS_MAX:
            hist = hist[-MARKET_HISTORY_WS_MAX:]
        return {"bids": bids[:10], "asks": asks[:10], "last_price": self.last_price, "history": hist}

    def record_trade_price(self, match_price: float) -> None:
        """Добавить тик цены с Unix-временем; сразу дописывает на диск (восстановление после перезапуска/сбоя)."""
        t = time.time()
        self.last_price = match_price
        row = {
            "time": time.strftime("%H:%M:%S", time.localtime(t)),
            "price": match_price,
            "ts": t,
        }
        self.price_history.append(row)
        jsonl = self._price_history_jsonl_path(self._active_state_path)
        self._append_price_history_jsonl(jsonl, row)

    def get_full_state(self):
        return {
            "height": self.current_height,
            "mempool_size": len(self.mempool),
            "accounts_count": len(self.accounts),
            "accounts": {addr: acc.model_dump(mode="json") for addr, acc in self.accounts.items()},
            "market": self.get_orderbook(),
            "blocks": self.blocks[-10:], # Return last 10 blocks for the tape
            "tps_history": self.tps_history,
            "current_epoch_burn": self.current_epoch_burn,
            "epoch_ant_sold_volume": self.epoch_ant_sold_volume,
            "epoch_ant_sold_last": self.epoch_ant_sold_last,
            "epoch_emission_coefficient": self.epoch_emission_coefficient,
            "blocks_per_epoch": BLOCKS_PER_EPOCH,
            "genesis_validator": GENESIS_VALIDATOR_ADDR,
            "consensus_validators": list(self.consensus_validator_set),
            "next_proposer": select_proposer_for_height(
                self.current_height + 1,
                self.consensus_validator_set,
                GENESIS_VALIDATOR_ADDR,
            ),
            "genesis_provider": GENESIS_PROVIDER_ADDR,
            "sim_treasury": SIM_TREASURY_ADDR,
            "canon_log": self.canon_log.to_list_newest_first(),
            "last_block_wallet_delta": self.last_block_wallet_delta,
            "canonical_block_interval_sec": CANONICAL_BLOCK_INTERVAL_SEC,
            "sim_block_interval_sec": self.sim_block_interval_sec,
            "sim_speed": self.sim_speed,
            "sim_speed_min": SIM_SPEED_MIN,
            "sim_speed_max": SIM_SPEED_MAX,
        }
        
    def add_block(self, block: dict):
        self.blocks.append(block)
        self.current_height += 1
        try:
            self.block_ledger.append_block(block)
        except OSError as e:
            print(f"Warning: block_ledger.append failed for h={block.get('height')}: {e}")
        self._index_block(block)
        # Хвост в RAM: для UI достаточно последних N блоков; остальное в JSONL.
        if len(self.blocks) > self._blocks_in_memory_max:
            self.blocks = self.blocks[-self._blocks_in_memory_max:]
        self._blocks_since_snapshot += 1

    def _index_block(self, block: dict) -> None:
        """Обновить tx_index/account_tx_index по транзакциям блока."""
        for rec in index_block_txs(block):
            tx_hash = rec["tx_hash"]
            self.tx_index[tx_hash] = rec
            for who in (rec.get("sender"), rec.get("receiver")):
                if not who:
                    continue
                bucket = self.account_tx_index.setdefault(who, [])
                bucket.append(tx_hash)
                # ограничим хвост, чтобы не разрастаться бесконечно
                if len(bucket) > 5000:
                    del bucket[: len(bucket) - 5000]

    def should_snapshot_now(self) -> bool:
        """True, если пора писать полный snapshot state.json (по блокам и не чаще раза в 2 с)."""
        return (
            self._blocks_since_snapshot >= self._snapshot_every_n
            and (time.time() - self._last_snapshot_ts) >= 2.0
        )

    def mark_snapshot_taken(self) -> None:
        self._blocks_since_snapshot = 0
        self._last_snapshot_ts = time.time()

    # --- Доступ к ledger для API ---

    def get_block_by_height(self, height: int) -> Optional[dict]:
        for blk in self.blocks:
            if int(blk.get("height", -1)) == height:
                return blk
        return self.block_ledger.get_by_height(height)

    def get_blocks_range(self, from_h: int, to_h: int) -> List[dict]:
        if from_h > to_h:
            from_h, to_h = to_h, from_h
        return self.block_ledger.read_range(from_h, to_h)

    def get_tx_record(self, tx_hash: str) -> Optional[dict]:
        idx = self.tx_index.get(tx_hash)
        if not idx:
            return None
        block = self.get_block_by_height(int(idx["height"]))
        out = dict(idx)
        if block is not None:
            tx_idx = int(idx.get("tx_idx", -1))
            txs = block.get("transactions") or []
            if 0 <= tx_idx < len(txs):
                out["tx"] = txs[tx_idx]
            out["block_hash"] = block.get("hash")
            out["block_timestamp"] = block.get("timestamp")
        return out

    def get_account_tx_history(self, address: str, limit: int = 100) -> List[dict]:
        hashes = list(reversed(self.account_tx_index.get(address, [])))[:limit]
        out: List[dict] = []
        for h in hashes:
            rec = self.get_tx_record(h)
            if rec is not None:
                out.append(rec)
        return out

    def snapshot_accounts(self) -> Dict[str, dict]:
        """Снимок аккаунтов (для расчёта дельты после блока)."""
        return {addr: acc.model_dump(mode="json") for addr, acc in self.accounts.items()}

    def snapshot_orders(self) -> Dict[str, dict]:
        return {oid: o.model_dump(mode="json") for oid, o in self.orders.items()}

    def compute_delta(
        self,
        accounts_before: Dict[str, dict],
        orders_before: Dict[str, dict],
    ) -> dict:
        """Дельта аккаунтов/ордеров для WS (только изменённое и удалённое)."""
        cur_accounts = self.snapshot_accounts()
        cur_orders = self.snapshot_orders()

        accounts_changed: Dict[str, dict] = {}
        for addr, snap in cur_accounts.items():
            prev = accounts_before.get(addr)
            if prev != snap:
                accounts_changed[addr] = snap
        accounts_removed = [addr for addr in accounts_before if addr not in cur_accounts]

        orders_changed: Dict[str, dict] = {}
        for oid, snap in cur_orders.items():
            prev = orders_before.get(oid)
            if prev != snap:
                orders_changed[oid] = snap
        orders_removed = [oid for oid in orders_before if oid not in cur_orders]

        return {
            "accounts_changed": accounts_changed,
            "accounts_removed": accounts_removed,
            "orders_changed": orders_changed,
            "orders_removed": orders_removed,
        }

    def rebuild_indices_from_blocks(self) -> None:
        """Перестроить tx-индекс из in-memory blocks + ledger (для load_state)."""
        self.tx_index.clear()
        self.account_tx_index.clear()
        seen: set = set()
        # 1) хвост из RAM
        for blk in self.blocks:
            for rec in index_block_txs(blk):
                self.tx_index[rec["tx_hash"]] = rec
                seen.add(rec["tx_hash"])
                for who in (rec.get("sender"), rec.get("receiver")):
                    if who:
                        self.account_tx_index.setdefault(who, []).append(rec["tx_hash"])
        # 2) исторические из JSONL (для tx_index «всех времён», но без дублей)
        try:
            for blk in self.block_ledger._writer.iter_records():
                for rec in index_block_txs(blk):
                    if rec["tx_hash"] in seen:
                        continue
                    self.tx_index[rec["tx_hash"]] = rec
                    seen.add(rec["tx_hash"])
                    for who in (rec.get("sender"), rec.get("receiver")):
                        if who:
                            self.account_tx_index.setdefault(who, []).append(rec["tx_hash"])
        except (OSError, AttributeError):
            pass

    def try_save_state(self) -> None:
        """Сохранить state.json (включая мемпул). Ошибки диска — только в лог, без raise."""
        try:
            self.save_state()
        except OSError as e:
            print(f"Warning: try_save_state failed: {e}")

    def save_state(self, filepath: Optional[str] = None):
        filepath = filepath or os.path.join(self.data_dir, "state.json")
        self._active_state_path = filepath
        d = os.path.dirname(filepath)
        if d:
            os.makedirs(d, exist_ok=True)
        mp_cap = self.mempool[:MEMPOOL_PERSIST_MAX]
        if len(self.mempool) > MEMPOOL_PERSIST_MAX:
            print(
                f"Warning: mempool {len(self.mempool)} tx — в state.json сохранены первые {MEMPOOL_PERSIST_MAX}"
            )
        data = {
            "current_height": self.current_height,
            "last_price": self.last_price,
            "price_history": self.price_history,
            "tps_history": self.tps_history,
            "current_epoch_burn": self.current_epoch_burn,
            "epoch_ant_sold_volume": self.epoch_ant_sold_volume,
            "epoch_ant_sold_last": self.epoch_ant_sold_last,
            "epoch_emission_coefficient": self.epoch_emission_coefficient,
            "last_block_wallet_delta": self.last_block_wallet_delta,
            "sim_block_interval_sec": self.sim_block_interval_sec,
            "sim_speed": self.sim_speed,
            "consensus_validator_set": list(self.consensus_validator_set),
            "genesis_validator": GENESIS_VALIDATOR_ADDR,
            "genesis_provider": GENESIS_PROVIDER_ADDR,
            "sim_treasury": SIM_TREASURY_ADDR,
            "accounts": {addr: acc.model_dump(mode="json") for addr, acc in self.accounts.items()},
            "orders": {oid: o.model_dump(mode="json") for oid, o in self.orders.items()},
            "blocks": self.blocks,
            "mempool": [tx.model_dump(mode="json") for tx in mp_cap],
        }
        with open(filepath, "w", encoding="utf-8") as f:
            json.dump(data, f)
        self._rewrite_price_history_jsonl(self._price_history_jsonl_path(filepath), self.price_history)

    def load_state(self, filepath: Optional[str] = None):
        filepath = filepath or os.path.join(self.data_dir, "state.json")
        self._active_state_path = filepath
        if os.path.exists(filepath):
            data = None
            try:
                with open(filepath, encoding="utf-8") as f:
                    data = json.load(f)
            except json.JSONDecodeError:
                # Иногда state.json может оборваться (например, при kill -9 во время записи).
                # Пытаемся подняться из резервной копии; если её нет/она тоже битая — стартуем с genesis.
                bak = filepath + ".bak"
                try:
                    if os.path.exists(bak):
                        with open(bak, encoding="utf-8") as f:
                            data = json.load(f)
                        print(f"Warning: state.json corrupted; restored from {os.path.basename(bak)}")
                except json.JSONDecodeError:
                    data = None
            if not isinstance(data, dict):
                data = {}
            self.current_height = data.get("current_height", 0)
            self.last_price = data.get("last_price", 0.0)
            self.price_history = data.get("price_history", [])
            self.tps_history = data.get("tps_history", [])
            self.current_epoch_burn = data.get("current_epoch_burn", 0.0)
            self.epoch_ant_sold_volume = data.get("epoch_ant_sold_volume", 0.0)
            self.epoch_ant_sold_last = data.get("epoch_ant_sold_last", 0.0)
            self.epoch_emission_coefficient = data.get("epoch_emission_coefficient", 1.0)
            self.blocks = data.get("blocks", [])
            self.last_block_wallet_delta = data.get("last_block_wallet_delta", {})
            _iv = float(data.get("sim_block_interval_sec", CANONICAL_BLOCK_INTERVAL_SEC))
            self.sim_block_interval_sec = max(
                MIN_SIM_BLOCK_INTERVAL_SEC, min(MAX_SIM_BLOCK_INTERVAL_SEC, _iv)
            )
            _spd_raw = data.get("sim_speed")
            if _spd_raw is None:
                # Legacy state.json без sim_speed — вывести из интервала блока
                _spd = CANONICAL_BLOCK_INTERVAL_SEC / self.sim_block_interval_sec
            else:
                _spd = float(_spd_raw)
            self.sim_speed = max(SIM_SPEED_MIN, min(SIM_SPEED_MAX, _spd))

            accounts_data = data.get("accounts", {})
            merged_accounts = {}
            for addr, raw in accounts_data.items():
                acc_data = dict(raw)
                acc_data.setdefault("lzn_frozen_mining", 0.0)
                acc_data.setdefault("zkp_verified", False)
                # Канон v4.20: тип 1 §4.2 — «Гражданин» (в state/json — citizen); старые guest → citizen
                if acc_data.get("role") == "guest":
                    acc_data["role"] = Role.CITIZEN.value
                merged_accounts[addr] = Account(**acc_data)
            if SIM_TREASURY_ADDR_LEGACY in merged_accounts:
                legacy_acc = merged_accounts.pop(SIM_TREASURY_ADDR_LEGACY)
                if SIM_TREASURY_ADDR not in merged_accounts:
                    ld = legacy_acc.model_dump()
                    ld["address"] = SIM_TREASURY_ADDR
                    merged_accounts[SIM_TREASURY_ADDR] = Account(**ld)
            self.accounts = merged_accounts
            
            orders_data = data.get("orders", {})
            self.orders = {oid: Order(**o_data) for oid, o_data in orders_data.items()}

            self.mempool = _deserialize_mempool(data.get("mempool", []))

            # blocks-в-памяти: если snapshot отстал, наверстаем хвост из JSONL
            try:
                ledger_tail = self.block_ledger.read_tail(self._blocks_in_memory_max)
                if ledger_tail:
                    # совместить snapshot и хвост по height (snapshot мог быть устаревшим)
                    by_h = {int(b.get("height", -1)): b for b in self.blocks}
                    for blk in ledger_tail:
                        by_h[int(blk.get("height", -1))] = blk
                    self.blocks = [by_h[h] for h in sorted(by_h.keys())]
                    if self.blocks:
                        max_h = int(self.blocks[-1].get("height", self.current_height))
                        if max_h > self.current_height:
                            self.current_height = max_h
            except OSError:
                pass

            _cvs = data.get("consensus_validator_set")
            if isinstance(_cvs, list) and _cvs:
                parsed: List[dict] = []
                for x in _cvs:
                    if not isinstance(x, dict) or not x.get("address"):
                        continue
                    parsed.append(
                        {
                            "address": str(x["address"]),
                            "power": float(x.get("power", 0.0) or 0.0),
                        }
                    )
                self.consensus_validator_set = parsed
            else:
                self.consensus_validator_set = default_consensus_validator_set(self.accounts)

        jsonl_path = self._price_history_jsonl_path(filepath)
        jsonl_rows = self._read_price_history_jsonl(jsonl_path)
        if jsonl_rows:
            self.price_history = jsonl_rows
        elif self.price_history:
            self._rewrite_price_history_jsonl(jsonl_path, self.price_history)

        # jsonl — источник истины для тиков; last_price в state.json мог устареть (0 при ненулевой истории).
        if self.price_history:
            try:
                p = float(self.price_history[-1].get("price", 0) or 0)
                if p >= 0:
                    self.last_price = p
            except (TypeError, ValueError, AttributeError, KeyError):
                pass

        # перестроить tx-индексы из имеющихся блоков (RAM + JSONL)
        self.rebuild_indices_from_blocks()

    def reset_state(self, filepath: Optional[str] = None):
        filepath = filepath or os.path.join(self.data_dir, "state.json")
        self._active_state_path = filepath
        self.current_height = 0
        self.accounts = {}
        self.mempool = []
        self.blocks = []
        self.orders = {}
        self.last_price = 0.0
        self.price_history = []
        self.tps_history = []
        self.current_epoch_burn = 0.0
        self.epoch_ant_sold_volume = 0.0
        self.epoch_ant_sold_last = 0.0
        self.epoch_emission_coefficient = 1.0
        self.last_block_wallet_delta = {}
        self.sim_block_interval_sec = CANONICAL_BLOCK_INTERVAL_SEC
        self.sim_speed = 1.0
        self.tx_index.clear()
        self.account_tx_index.clear()
        self._blocks_since_snapshot = 0
        self.canon_log.clear()
        # Truncate JSONL ledgers
        try:
            self.block_ledger.truncate()
        except OSError:
            pass
        if self._canon_ledger is not None:
            try:
                self._canon_ledger.truncate()
            except OSError:
                pass
        self.init_genesis()
        if os.path.exists(filepath):
            os.remove(filepath)
        jpath = self._price_history_jsonl_path(filepath)
        if os.path.exists(jpath):
            os.remove(jpath)
        tmp = jpath + ".tmp"
        if os.path.exists(tmp):
            os.remove(tmp)
        try:
            self.save_state(filepath)
        except OSError as e:
            print(f"Warning: save_state after reset failed: {e}")
