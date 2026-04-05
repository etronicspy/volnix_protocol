import json
import os
from typing import Dict, List

from core.models import Account, Role, Transaction, Order, OrderType, TransactionType

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
    def __init__(self):
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

    def init_genesis(self):
        import time
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
        
        return {"bids": bids[:10], "asks": asks[:10], "last_price": self.last_price, "history": self.price_history}

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
        }
        
    def add_block(self, block: dict):
        self.blocks.append(block)
        self.current_height += 1

    def save_state(self, filepath="data/state.json"):
        os.makedirs(os.path.dirname(filepath), exist_ok=True)
        data = {
            "current_height": self.current_height,
            "last_price": self.last_price,
            "price_history": self.price_history,
            "tps_history": self.tps_history,
            "current_epoch_burn": self.current_epoch_burn,
            "epoch_ant_sold_volume": self.epoch_ant_sold_volume,
            "epoch_ant_sold_last": self.epoch_ant_sold_last,
            "epoch_emission_coefficient": self.epoch_emission_coefficient,
            "genesis_validator": GENESIS_VALIDATOR_ADDR,
            "genesis_provider": GENESIS_PROVIDER_ADDR,
            "sim_treasury": SIM_TREASURY_ADDR,
            "accounts": {addr: acc.model_dump(mode="json") for addr, acc in self.accounts.items()},
            "orders": {oid: o.model_dump(mode="json") for oid, o in self.orders.items()},
            "blocks": self.blocks
        }
        with open(filepath, "w") as f:
            json.dump(data, f)

    def load_state(self, filepath="data/state.json"):
        if os.path.exists(filepath):
            with open(filepath, "r") as f:
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
            
            accounts_data = data.get("accounts", {})
            merged_accounts = {}
            for addr, raw in accounts_data.items():
                acc_data = dict(raw)
                acc_data.setdefault("lzn_frozen_mining", 0.0)
                acc_data.setdefault("zkp_verified", False)
                merged_accounts[addr] = Account(**acc_data)
            self.accounts = merged_accounts
            
            orders_data = data.get("orders", {})
            self.orders = {oid: Order(**o_data) for oid, o_data in orders_data.items()}
            
            self.mempool = []

    def reset_state(self, filepath="data/state.json"):
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
        self.init_genesis()
        if os.path.exists(filepath):
            os.remove(filepath)
