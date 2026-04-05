import json
import os
import time
from typing import Dict, List, Optional

from core.models import Account, Role, Transaction, Order, OrderType, TransactionType
from core.canon_log import CanonLogBuffer

# Эпоха ANT: технические блоки (§5.5, §7.2 п.11) — эталон 1 блок/мин × 7 суток
BLOCKS_PER_EPOCH = 7 * 24 * 60  # 10080
# §4.2: не более ⌊эталон/3⌋ активированных LZN на адрес (целые токены; эталон 10_000 → 3333).
# Genesis-валидатор: 6667 активированных (остаток ликвидности = 10_000 − 6667).
LZN_TOTAL_SUPPLY_REF = 10_000
LZN_GENESIS_ACTIVATED = 6_667
LZN_MAX_FROZEN_PER_ADDRESS = LZN_TOTAL_SUPPLY_REF // 3

# §6.3: два фиксированных genesis-адреса, без ZKP, без роли «Супервизор»
GENESIS_VALIDATOR_ADDR = "volnix1gval0validator0genesis0"
# Симуляция: стартовый ANT на genesis-валидаторе (удобство declare / рынок; не смешивать с §6.3 провайдера)
GENESIS_VALIDATOR_ANT_BALANCE = 1_000_000_000.0
GENESIS_PROVIDER_ADDR = "volnix1gprov0provider00genesis0"
# Вне цепочки: резерв симулятора для God Mode (не в genesis-блоке)
SIM_TREASURY_ADDR = "sim_treasury_reserve_godmode"

# Сколько последних тиков отдаём в WebSocket/API (полная история остаётся в памяти и price_history.jsonl).
MARKET_HISTORY_WS_MAX = 30_000


def account_total_lzn(acc: Account) -> float:
    return acc.lzn_balance + acc.lzn_frozen_mining


def eligible_for_validator_role(address: str, acc: Account) -> bool:
    """§3.1 + §6.3: genesis-валидатор без ZKP в цепочке; остальные — ZKP и LZN."""
    if address == GENESIS_VALIDATOR_ADDR:
        return True
    return acc.zkp_verified and account_total_lzn(acc) > 0


def eligible_for_provider_role(address: str, acc: Account) -> bool:
    """Поставщик: как валидатор, кроме genesis-поставщика (§6.3: без LZN на кошельке)."""
    if address == GENESIS_PROVIDER_ADDR:
        return True
    return acc.zkp_verified and account_total_lzn(acc) > 0


class StateManager:
    def __init__(self, data_dir: str = "data"):
        self.data_dir = data_dir
        self._active_state_path = os.path.join(data_dir, "state.json")
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
        self.canon_log = CanonLogBuffer(maxlen=400)
        # После каждого блока: изменение балансов за блок (для бота — цена vs фактические WRT/ANT)
        self.last_block_wallet_delta: Dict[str, Dict[str, float]] = {}

    @staticmethod
    def _price_history_jsonl_path(state_json_path: str) -> str:
        d = os.path.dirname(os.path.abspath(state_json_path))
        return os.path.join(d if d else ".", "price_history.jsonl")

    @staticmethod
    def _read_price_history_jsonl(jsonl_path: str) -> List[dict]:
        if not os.path.exists(jsonl_path):
            return []
        out: List[dict] = []
        with open(jsonl_path, "r", encoding="utf-8") as f:
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
                    f"EpochBlocks={BLOCKS_PER_EPOCH}; ANT_genesis=EpochBlocks×L_total={ant_genesis:.0f}; "
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
                    f"Симуляция: стартовый ANT на genesis-Валидаторе = {GENESIS_VALIDATOR_ANT_BALANCE:.0f} "
                    "(для declare §5.4 и внутреннего рынка)."
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

        self.blocks.append({
            "height": 0,
            "hash": "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
            "timestamp": ts,
            "transactions": txs,
            "tx_count": len(txs),
        })

    def create_account(self, address: str) -> Account:
        if address not in self.accounts:
            self.accounts[address] = Account(address=address)
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
            "genesis_provider": GENESIS_PROVIDER_ADDR,
            "sim_treasury": SIM_TREASURY_ADDR,
            "canon_log": self.canon_log.to_list_newest_first(),
            "last_block_wallet_delta": self.last_block_wallet_delta,
        }
        
    def add_block(self, block: dict):
        self.blocks.append(block)
        self.current_height += 1

    def save_state(self, filepath: Optional[str] = None):
        filepath = filepath or os.path.join(self.data_dir, "state.json")
        self._active_state_path = filepath
        d = os.path.dirname(filepath)
        if d:
            os.makedirs(d, exist_ok=True)
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
            "genesis_validator": GENESIS_VALIDATOR_ADDR,
            "genesis_provider": GENESIS_PROVIDER_ADDR,
            "sim_treasury": SIM_TREASURY_ADDR,
            "accounts": {addr: acc.model_dump(mode="json") for addr, acc in self.accounts.items()},
            "orders": {oid: o.model_dump(mode="json") for oid, o in self.orders.items()},
            "blocks": self.blocks
        }
        with open(filepath, "w", encoding="utf-8") as f:
            json.dump(data, f)
        self._rewrite_price_history_jsonl(self._price_history_jsonl_path(filepath), self.price_history)

    def load_state(self, filepath: Optional[str] = None):
        filepath = filepath or os.path.join(self.data_dir, "state.json")
        self._active_state_path = filepath
        if os.path.exists(filepath):
            with open(filepath, "r", encoding="utf-8") as f:
                data = json.load(f)
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
            self.accounts = merged_accounts
            
            orders_data = data.get("orders", {})
            self.orders = {oid: Order(**o_data) for oid, o_data in orders_data.items()}
            
            self.mempool = []

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
        self.canon_log.clear()
        self.init_genesis()
        if os.path.exists(filepath):
            os.remove(filepath)
        jpath = self._price_history_jsonl_path(filepath)
        if os.path.exists(jpath):
            os.remove(jpath)
        tmp = jpath + ".tmp"
        if os.path.exists(tmp):
            os.remove(tmp)
