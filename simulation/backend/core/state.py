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
# §5.4 / §7.2 п.9: коридор сжигания за блок —
#   λ·L_total ≤ Σ b_i ≤ (1−λ)·L_total
# (исправление точки-коридора, где верх и низ были оба λ·L_total).
# Ограничение DAO: λ ≤ BURN_CAP_LAMBDA_MAX (= 5/12) → ширина коридора ≥ 1/6.
BURN_CAP_LAMBDA = 1.0 / 3.0
BURN_CAP_LAMBDA_MAX = 5.0 / 12.0


def burn_floor(L_total: float, lam: float = BURN_CAP_LAMBDA) -> float:
    """Нижняя граница Σ b_i за блок: λ · L_total."""
    return float(lam) * float(L_total)


def burn_ceiling(L_total: float, lam: float = BURN_CAP_LAMBDA) -> float:
    """Верхняя граница Σ b_i за блок: (1 − λ) · L_total."""
    return (1.0 - float(lam)) * float(L_total)


# §4.2: не более ⌊эталон/3⌋ активированных LZN на адрес (целые токены; эталон 10_000 → 3333).
# §4.1 5.0-sim: LZN_TOTAL_SUPPLY_REF — потолок обращения (продукция Поставщиков), не разовая эмиссия seed.
LZN_TOTAL_SUPPLY_REF = 10_000
LZN_MAX_FROZEN_PER_ADDRESS = LZN_TOTAL_SUPPLY_REF // 3
# §5.5 / §7.2 п.13: потолок производства LZN за эпоху (DAO; эталон стенда).
LZN_MAX_EPOCH = 1_000.0

# §6.3 (sim 5.0): единственный seed на высоте 0; когорта 5+5 — на высоте 2.
GENESIS_VALIDATOR_ADDR = "volnix1gval0validator0genesis0"
LZN_GENESIS_ACTIVATED = 1
GENESIS_VALIDATOR_ANT_BALANCE = 1.0

GENESIS_BOOTSTRAP_VALIDATOR_COUNT = 5
GENESIS_BOOTSTRAP_PROVIDER_COUNT = 5
GENESIS_BOOTSTRAP_VALIDATOR_LZN = 1_000
GENESIS_BOOTSTRAP_VALIDATORS: tuple[str, ...] = tuple(
    f"volnix1val{i:02d}bootstrap0zkp00" for i in range(GENESIS_BOOTSTRAP_VALIDATOR_COUNT)
)
GENESIS_BOOTSTRAP_PROVIDERS: tuple[str, ...] = tuple(
    f"volnix1prov{i:02d}bootstrap0zkp00" for i in range(GENESIS_BOOTSTRAP_PROVIDER_COUNT)
)
# Совместимость UI/API: «первый» поставщик bootstrap-когорты (появляется на высоте 2).
GENESIS_PROVIDER_ADDR = GENESIS_BOOTSTRAP_PROVIDERS[0]

# Первая продукция LZN Поставщикам (= спрос bootstrap-валидаторов на мощность)
# + остаток на открытую книгу после sim-fill.
LZN_BOOTSTRAP_TOTAL = float(
    GENESIS_BOOTSTRAP_VALIDATOR_COUNT * GENESIS_BOOTSTRAP_VALIDATOR_LZN
)
LZN_BOOTSTRAP_MARKET_EXTRA = 200.0  # непроданный остаток на книгу после fill
LZN_BOOTSTRAP_PER_PROVIDER = (
    LZN_BOOTSTRAP_TOTAL / float(GENESIS_BOOTSTRAP_PROVIDER_COUNT)
    + LZN_BOOTSTRAP_MARKET_EXTRA
)

# Стартовая ANT на поставщиков = спрос первой эпохи (EpochBlocks × λ × L_total),
# L_total после sim-fill LZN на bootstrap-валидаторов + seed.
L_TOTAL_GENESIS = float(
    GENESIS_BOOTSTRAP_VALIDATOR_COUNT * GENESIS_BOOTSTRAP_VALIDATOR_LZN + LZN_GENESIS_ACTIVATED
)
ANT_GENESIS_TOTAL = float(BLOCKS_PER_EPOCH) * BURN_CAP_LAMBDA * L_TOTAL_GENESIS
ANT_GENESIS_PER_PROVIDER = ANT_GENESIS_TOTAL / float(GENESIS_BOOTSTRAP_PROVIDER_COUNT)

# Симуляционный алгоритм: когорта 5+5 вводится в BeginBlock высоты 2 (не в genesis).
SIM_BOOTSTRAP_INJECT_HEIGHT = 2

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


def circulating_lzn(accounts: Dict[str, Account]) -> float:
    """§4.1: LZN в обращении — всё, кроме нераспределённого резерва симуляции."""
    return sum(
        account_total_lzn(a)
        for addr, a in accounts.items()
        if addr != SIM_TREASURY_ADDR
    )


def lzn_mint_headroom(accounts: Dict[str, Account]) -> float:
    """Сколько LZN ещё можно произвести, не превысив потолок обращения §4.1 5.0-sim."""
    return max(0.0, float(LZN_TOTAL_SUPPLY_REF) - circulating_lzn(accounts))


def order_asset(order: Order) -> str:
    """Книга ордера: ant (по умолчанию для старых state) или lzn."""
    a = (getattr(order, "asset", None) or "ant").lower()
    return a if a in ("ant", "lzn") else "ant"


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
    """Поставщик: требует ZKP; LZN не требуется (§3.1 / §4.2)."""
    _ = address
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


def validator_set_from_activated_lzn(accounts: Dict[str, Account]) -> List[dict]:
    """§6.1 fallback: набор из всех валидаторов с активированным LZN, power ∝ L_i.

    Нужен, когда на высоте не было ни одного declare: иначе `consensus_validator_set`
    остаётся замороженным с прошлого обновления и цепь бесконечно штампует блоки
    одним и тем же пропозером (вырождение консенсуса).
    """
    out: List[dict] = []
    for addr in sorted(accounts):
        acc = accounts[addr]
        if acc.role != Role.VALIDATOR:
            continue
        L_i = float(acc.lzn_frozen_mining)
        if L_i <= 0:
            continue
        out.append({"address": addr, "power": L_i})
    return out


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
        self.epoch_lzn_sold_volume = 0.0  # §5.5 5.0-sim: объём LZN, проданного за эпоху
        self.epoch_lzn_sold_last = 0.0
        self.epoch_lzn_emission_coefficient = 1.0
        self.last_lzn_price = 0.0
        self.lzn_price_history: List[dict] = []
        self.canon_log = CanonLogBuffer(maxlen=settings.canon_log_capacity)
        # После каждого блока: изменение балансов за блок (для бота — цена vs фактические WRT/ANT)
        self.last_block_wallet_delta: Dict[str, Dict[str, float]] = {}
        self.sim_block_interval_sec: float = CANONICAL_BLOCK_INTERVAL_SEC
        # Скорость симуляции (сим. секунд за реальную): 1.0 = реальное время.
        self.sim_speed: float = 1.0
        # Якорь «unix» для графиков: genesis_unix + height×60. Фиксируется при genesis/load.
        self.genesis_unix: float = time.time()
        # Сколько сделок уже записано на текущей высоте (субсекундный сдвиг ts).
        self._trades_on_height: int = 0
        # ValidatorSet для выбора пропозера (§6.1, §6.3): после genesis — из EndBlocker/declare §5.4
        self.consensus_validator_set: List[dict] = []
        # Симуляция: когорта 5 валидаторов + 5 поставщиков уже введена (обычно на высоте 2).
        self.sim_bootstrap_injected: bool = False

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
        self.genesis_unix = ts
        self._trades_on_height = 0
        self.sim_bootstrap_injected = False

        # Единственный протокольный seed-адрес (+ служебная казна симуляции).
        gv = self.create_account(GENESIS_VALIDATOR_ADDR)
        gv.role = Role.VALIDATOR
        gv.wrt_balance = 0.0
        gv.lzn_balance = 0.0
        gv.lzn_frozen_mining = float(LZN_GENESIS_ACTIVATED)
        gv.ant_balance = float(GENESIS_VALIDATOR_ANT_BALANCE)
        gv.zkp_verified = True  # seed без отдельной ZKP-записи; UI — «верифицирован»

        tr = self.create_account(SIM_TREASURY_ADDR)
        tr.role = Role.CITIZEN
        tr.wrt_balance = 1_000_000.0
        tr.lzn_balance = 1_000_000.0
        tr.ant_balance = 1_000_000.0

        txs: list = [
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_MESSAGE,
                details=(
                    "Volnix Protocol §6.3 (sim) — genesis: один seed-кошелёк "
                    f"({GENESIS_VALIDATOR_ADDR}) с {LZN_GENESIS_ACTIVATED} LZN + "
                    f"{GENESIS_VALIDATOR_ANT_BALANCE:.0f} ANT. "
                    f"Когорта {GENESIS_BOOTSTRAP_VALIDATOR_COUNT}+{GENESIS_BOOTSTRAP_PROVIDER_COUNT} "
                    f"вводится симуляцией на высоте {SIM_BOOTSTRAP_INJECT_HEIGHT} "
                    f"(не genesis). ValidatorSet seed = {GENESIS_VALIDATOR_ADDR}; "
                    f"EpochBlocks={BLOCKS_PER_EPOCH}."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_VALIDATOR_LZN,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=float(LZN_GENESIS_ACTIVATED),
                asset_type="lzn",
                details=(
                    f"Seed: {LZN_GENESIS_ACTIVATED} LZN на {GENESIS_VALIDATOR_ADDR} "
                    "(минимальный задел для создания блоков до bootstrap-когорты)."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_LZN_ACTIVATE,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=float(LZN_GENESIS_ACTIVATED),
                asset_type="lzn",
                details=f"Seed: {LZN_GENESIS_ACTIVATED} LZN активировано под майнинг.",
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_VALIDATOR_ANT,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=float(GENESIS_VALIDATOR_ANT_BALANCE),
                asset_type="ant",
                details=(
                    f"Seed: {GENESIS_VALIDATOR_ANT_BALANCE:.0f} ANT на {GENESIS_VALIDATOR_ADDR} "
                    "(электричество для Σb_i > 0)."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
        ]

        L_g = float(LZN_GENESIS_ACTIVATED)
        genesis_block = {
            "height": 0,
            "hash": "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
            "timestamp": ts,
            "transactions": txs,
            "tx_count": len(txs),
            "competition": {
                "kind": "genesis",
                "lambda": float(BURN_CAP_LAMBDA),
                "K": 150,
                "L_total": L_g,
                "floor": burn_floor(L_g),
                "cap": burn_ceiling(L_g),
                "B_candidates": 0.0,
                "B_selected": 0.0,
                "candidates_count": 0,
                "selected_count": 0,
                "culled_lambda_count": 0,
                "culled_k_count": 0,
                "deferred_count": 0,
                "entries": [],
            },
        }
        self.blocks.append(genesis_block)
        try:
            self.block_ledger.append_block(genesis_block)
        except OSError:
            pass
        self._index_block(genesis_block)
        self.consensus_validator_set = default_consensus_validator_set(self.accounts)

    def inject_sim_bootstrap_cohort(self, txs_in_block: list) -> bool:
        """Симуляционный старт 5.0-sim: 5 поставщиков + 5 валидаторов.

        Поставщики получают первую продукцию ANT и LZN (§5.5). Валидаторы получают
        мощность через sim-fill сделок LZN у Поставщиков (не «эмиссия с неба»).
        Не канон mainnet §6.3 — алгоритм стенда. Идемпотентно.
        """
        import uuid

        if self.sim_bootstrap_injected:
            return False
        if all(a in self.accounts for a in GENESIS_BOOTSTRAP_VALIDATORS) and all(
            a in self.accounts for a in GENESIS_BOOTSTRAP_PROVIDERS
        ):
            self.sim_bootstrap_injected = True
            return False

        ts = time.time()
        txs_in_block.append(
            {
                "tx_hash": uuid.uuid4().hex,
                "tx_type": "sim_bootstrap_message",
                "details": (
                    f"Sim bootstrap height={SIM_BOOTSTRAP_INJECT_HEIGHT} (5.0-sim): "
                    f"{GENESIS_BOOTSTRAP_PROVIDER_COUNT} поставщиков "
                    f"(ANT_total={ANT_GENESIS_TOTAL:.0f}, LZN_total={LZN_BOOTSTRAP_TOTAL:.0f}); "
                    f"{GENESIS_BOOTSTRAP_VALIDATOR_COUNT} валидаторов получают LZN "
                    f"через sim-fill рынка (по {GENESIS_BOOTSTRAP_VALIDATOR_LZN}). "
                    f"Не канон mainnet §6.3."
                ),
                "timestamp": ts,
            }
        )

        # 1) Поставщики: ZKP + роль + продукция ANT + LZN на продажу.
        for i, addr in enumerate(GENESIS_BOOTSTRAP_PROVIDERS):
            acc = self.create_account(addr)
            acc.role = Role.PROVIDER
            acc.wrt_balance = 0.0
            acc.lzn_balance = float(LZN_BOOTSTRAP_PER_PROVIDER)
            acc.ant_balance = float(ANT_GENESIS_PER_PROVIDER)
            acc.zkp_verified = True
            for tx in (
                Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.ZKP_VERIFY,
                    sender=addr,
                    receiver=addr,
                    details=f"Sim bootstrap ZKP §3.1: поставщик[{i}].",
                    timestamp=ts,
                ),
                Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.SET_ROLE,
                    sender=addr,
                    receiver=addr,
                    role=Role.PROVIDER,
                    details=f"Sim bootstrap §4.2: Поставщик {addr}.",
                    timestamp=ts,
                ),
                Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.GENESIS_PROVIDER_ANT,
                    receiver=addr,
                    amount=float(ANT_GENESIS_PER_PROVIDER),
                    asset_type="ant",
                    details=(
                        f"Sim bootstrap ANT: {ANT_GENESIS_PER_PROVIDER:.0f} "
                        f"(доля от EpochBlocks×λ×L_total={ANT_GENESIS_TOTAL:.0f})."
                    ),
                    timestamp=ts,
                ),
                Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.EPOCH_LZN_CREDIT,
                    receiver=addr,
                    amount=float(LZN_BOOTSTRAP_PER_PROVIDER),
                    asset_type="lzn",
                    details=(
                        f"Sim bootstrap LZN §5.5: {LZN_BOOTSTRAP_PER_PROVIDER:.0f} "
                        f"продукция Поставщика (fill {GENESIS_BOOTSTRAP_VALIDATOR_LZN} + "
                        f"книга {LZN_BOOTSTRAP_MARKET_EXTRA:.0f})."
                    ),
                    timestamp=ts,
                ),
            ):
                txs_in_block.append(tx.model_dump(mode="json"))

        # 2) Валидаторы: ZKP + роль; мощность — sim-fill LZN у соответствующего Поставщика.
        # Стартовый ANT = λ·L_i — чтобы сразу набрать низ коридора Σb_i ≥ λ·L_total
        # (sim-fill «уже купленного» топлива; иначе L_total скачет, а ANT=0 → stall).
        for i, addr in enumerate(GENESIS_BOOTSTRAP_VALIDATORS):
            acc = self.create_account(addr)
            acc.role = Role.VALIDATOR
            acc.wrt_balance = 0.0
            acc.lzn_balance = 0.0
            acc.lzn_frozen_mining = 0.0
            acc.ant_balance = 0.0
            acc.zkp_verified = True
            for tx in (
                Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.ZKP_VERIFY,
                    sender=addr,
                    receiver=addr,
                    details=f"Sim bootstrap ZKP §3.1: валидатор[{i}].",
                    timestamp=ts,
                ),
                Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.SET_ROLE,
                    sender=addr,
                    receiver=addr,
                    role=Role.VALIDATOR,
                    details=f"Sim bootstrap §4.2: Валидатор {addr}.",
                    timestamp=ts,
                ),
            ):
                txs_in_block.append(tx.model_dump(mode="json"))

            prov_addr = GENESIS_BOOTSTRAP_PROVIDERS[i]
            prov = self.accounts[prov_addr]
            lot = float(GENESIS_BOOTSTRAP_VALIDATOR_LZN)
            if prov.lzn_balance + 1e-12 < lot:
                lot = float(prov.lzn_balance)
            if lot <= 0:
                continue
            prov.lzn_balance -= lot
            acc.lzn_frozen_mining += lot
            starter_ant = float(BURN_CAP_LAMBDA) * lot
            acc.ant_balance += starter_ant
            self.epoch_lzn_sold_volume += lot
            txs_in_block.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "sim_bootstrap_lzn_fill",
                    "sender": prov_addr,
                    "receiver": addr,
                    "amount": lot,
                    "asset_type": "lzn",
                    "details": (
                        f"Sim bootstrap §5.2/§6.3: исполненная сделка LZN "
                        f"{lot:.0f} {prov_addr} → {addr} (активировано под майнинг)."
                    ),
                    "timestamp": ts,
                }
            )
            txs_in_block.append(
                {
                    "tx_hash": uuid.uuid4().hex,
                    "tx_type": "sim_bootstrap_ant_fill",
                    "receiver": addr,
                    "amount": starter_ant,
                    "asset_type": "ant",
                    "details": (
                        f"Sim bootstrap §5.4: стартовый ANT {starter_ant:.4f} "
                        f"(= λ·L_i) для низа коридора после fill LZN."
                    ),
                    "timestamp": ts,
                }
            )

        self.sim_bootstrap_injected = True
        return True

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

    @staticmethod
    def _is_declare_like(tx: Transaction) -> bool:
        """§5.4: declare/burn — один слот на адрес в мемпуле (replace-by-sender)."""
        return tx.tx_type in (
            TransactionType.DECLARE_PARTICIPATION,
            TransactionType.BURN,
        )

    def mempool_admit(self, tx: Transaction) -> bool:
        """Приём tx в глобальный мемпул.

        Для declare/burn: новый tx того же sender **заменяет** старый
        (правило стенда / vNext: не копить дубли на высоту).
        Остальные типы — append.
        Возвращает True, если был replace (старый вытеснен).
        """
        replaced = False
        if self._is_declare_like(tx) and tx.sender:
            sid = tx.sender
            before = len(self.mempool)
            self.mempool = [
                t
                for t in self.mempool
                if not (t.sender == sid and self._is_declare_like(t))
            ]
            replaced = len(self.mempool) < before
        self.mempool.append(tx)
        return replaced

    def compact_declare_mempool(self) -> int:
        """Оставить только последний declare/burn на sender. Возвращает число снятых."""
        kept: List[Transaction] = []
        last_declare: Dict[str, Transaction] = {}
        declare_order: List[str] = []
        removed = 0
        for tx in self.mempool:
            if self._is_declare_like(tx) and tx.sender:
                sid = tx.sender
                if sid in last_declare:
                    removed += 1
                else:
                    declare_order.append(sid)
                last_declare[sid] = tx
            else:
                kept.append(tx)
        for sid in declare_order:
            kept.append(last_declare[sid])
        self.mempool = kept
        return removed

    def submit_tx(self, tx: Transaction) -> str:
        """Единая подача tx: NetworkSim (если есть) или локальный мемпул.

        Declare/burn на сети тоже replace-by-sender (см. NetworkSim.submit_to).
        """
        net = getattr(self, "network", None)
        if net is not None:
            addr = getattr(tx, "sender", "") or ""
            try:
                if addr:
                    net.submit_from_addr(addr, tx)
                else:
                    net.submit_to("node_0", tx)
                return "network"
            except Exception:
                pass
        self.mempool_admit(tx)
        return "local"

    def list_open_orders_for_address(self, address: str) -> List[dict]:
        return [
            o.model_dump(mode="json")
            for o in self.orders.values()
            if o.owner == address
        ]

    def get_orderbook(self):
        def _book(asset: str) -> dict:
            bids = [
                o.model_dump(mode="json")
                for o in self.orders.values()
                if o.order_type == OrderType.BUY and order_asset(o) == asset
            ]
            asks = [
                o.model_dump(mode="json")
                for o in self.orders.values()
                if o.order_type == OrderType.SELL and order_asset(o) == asset
            ]
            bids.sort(key=lambda x: (-x["price"], x["timestamp"]))
            asks.sort(key=lambda x: (x["price"], x["timestamp"]))
            if asset == "lzn":
                last = self.last_lzn_price
                hist = self.lzn_price_history
            else:
                last = self.last_price
                hist = self.price_history
            if len(hist) > MARKET_HISTORY_WS_MAX:
                hist = hist[-MARKET_HISTORY_WS_MAX:]
            return {
                "bids": bids[:10],
                "asks": asks[:10],
                "last_price": last,
                "history": hist,
            }

        ant = _book("ant")
        lzn = _book("lzn")
        # Совместимость: верхний уровень = книга ANT (как раньше).
        return {
            "bids": ant["bids"],
            "asks": ant["asks"],
            "last_price": ant["last_price"],
            "history": ant["history"],
            "ant": ant,
            "lzn": lzn,
        }

    def sim_timestamp(self, *, for_height: Optional[int] = None) -> float:
        """Время симуляции для графиков/блоков: не wall-clock.

        Один блок = CANONICAL_BLOCK_INTERVAL_SEC сим. секунд. При ускорении ×N
        тики рынка и блоки идут плотнее по реальному времени, но ts растёт
        пропорционально высоте — график остаётся синхронен цепи.
        """
        h = int(self.current_height if for_height is None else for_height)
        return float(self.genesis_unix) + max(0, h) * CANONICAL_BLOCK_INTERVAL_SEC

    def record_trade_price(self, match_price: float, *, asset: str = "ant") -> None:
        """Тик цены в sim-time (высота блока + субсекундный индекс сделки)."""
        self._trades_on_height += 1
        t = self.sim_timestamp(for_height=self.current_height + 1)
        t += self._trades_on_height * 1e-3
        row = {
            "time": time.strftime("%H:%M:%S", time.localtime(t)),
            "price": match_price,
            "ts": t,
            "asset": asset,
        }
        if asset == "lzn":
            self.last_lzn_price = match_price
            self.lzn_price_history.append(row)
            return
        self.last_price = match_price
        self.price_history.append(row)
        jsonl = self._price_history_jsonl_path(self._active_state_path)
        try:
            self._append_price_history_jsonl(jsonl, row)
        except OSError:
            pass

    def note_block_committed(self, height: int) -> None:
        """Сброс счётчика сделок после фиксации блока (новый слот высоты)."""
        self._trades_on_height = 0
        _ = height

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
            "epoch_lzn_sold_volume": self.epoch_lzn_sold_volume,
            "epoch_lzn_sold_last": self.epoch_lzn_sold_last,
            "epoch_lzn_emission_coefficient": self.epoch_lzn_emission_coefficient,
            "blocks_per_epoch": BLOCKS_PER_EPOCH,
            "genesis_validator": GENESIS_VALIDATOR_ADDR,
            "consensus_validators": list(self.consensus_validator_set),
            "next_proposer": select_proposer_for_height(
                self.current_height + 1,
                self.consensus_validator_set,
                GENESIS_VALIDATOR_ADDR,
            ),
            "genesis_provider": GENESIS_PROVIDER_ADDR,
            "genesis_bootstrap_validators": list(GENESIS_BOOTSTRAP_VALIDATORS),
            "genesis_bootstrap_providers": list(GENESIS_BOOTSTRAP_PROVIDERS),
            "sim_bootstrap_inject_height": SIM_BOOTSTRAP_INJECT_HEIGHT,
            "sim_bootstrap_injected": self.sim_bootstrap_injected,
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
            "epoch_lzn_sold_volume": self.epoch_lzn_sold_volume,
            "epoch_lzn_sold_last": self.epoch_lzn_sold_last,
            "epoch_lzn_emission_coefficient": self.epoch_lzn_emission_coefficient,
            "last_lzn_price": self.last_lzn_price,
            "lzn_price_history": self.lzn_price_history,
            "last_block_wallet_delta": self.last_block_wallet_delta,
            "sim_block_interval_sec": self.sim_block_interval_sec,
            "sim_speed": self.sim_speed,
            "genesis_unix": self.genesis_unix,
            "consensus_validator_set": list(self.consensus_validator_set),
            "genesis_validator": GENESIS_VALIDATOR_ADDR,
            "genesis_provider": GENESIS_PROVIDER_ADDR,
            "genesis_bootstrap_validators": list(GENESIS_BOOTSTRAP_VALIDATORS),
            "genesis_bootstrap_providers": list(GENESIS_BOOTSTRAP_PROVIDERS),
            "sim_bootstrap_inject_height": SIM_BOOTSTRAP_INJECT_HEIGHT,
            "sim_bootstrap_injected": self.sim_bootstrap_injected,
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
            self.epoch_lzn_sold_volume = data.get("epoch_lzn_sold_volume", 0.0)
            self.epoch_lzn_sold_last = data.get("epoch_lzn_sold_last", 0.0)
            self.epoch_lzn_emission_coefficient = data.get(
                "epoch_lzn_emission_coefficient", 1.0
            )
            self.last_lzn_price = float(data.get("last_lzn_price", 0.0) or 0.0)
            self.lzn_price_history = data.get("lzn_price_history", []) or []
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
            _gu = data.get("genesis_unix")
            if _gu is not None:
                try:
                    self.genesis_unix = float(_gu)
                except (TypeError, ValueError):
                    self.genesis_unix = time.time() - self.current_height * CANONICAL_BLOCK_INTERVAL_SEC
            else:
                # Legacy: восстановить якорь так, чтобы текущая высота совпала с «сейчас».
                self.genesis_unix = time.time() - self.current_height * CANONICAL_BLOCK_INTERVAL_SEC
            self._trades_on_height = 0
            self.sim_bootstrap_injected = bool(data.get("sim_bootstrap_injected", False))
            if not self.sim_bootstrap_injected:
                # Legacy state с уже созданной когортой в accounts.
                if all(a in (data.get("accounts") or {}) for a in GENESIS_BOOTSTRAP_VALIDATORS) and all(
                    a in (data.get("accounts") or {}) for a in GENESIS_BOOTSTRAP_PROVIDERS
                ):
                    self.sim_bootstrap_injected = True

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
        self.epoch_lzn_sold_volume = 0.0
        self.epoch_lzn_sold_last = 0.0
        self.epoch_lzn_emission_coefficient = 1.0
        self.last_lzn_price = 0.0
        self.lzn_price_history = []
        self.last_block_wallet_delta = {}
        self.sim_block_interval_sec = CANONICAL_BLOCK_INTERVAL_SEC
        self.sim_speed = 1.0
        self.genesis_unix = time.time()
        self._trades_on_height = 0
        self.sim_bootstrap_injected = False
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
