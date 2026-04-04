import json
import os
from typing import Dict, List

from core.models import Account, Role, Transaction, Order, OrderType, TransactionType

# Эпоха ANT: технические блоки (§5.5, §7.2 п.11) — эталон 1 блок/мин × 7 суток
BLOCKS_PER_EPOCH = 7 * 24 * 60  # 10080

# §6.3: два фиксированных genesis-адреса, без ZKP, без роли «Супервизор»
GENESIS_VALIDATOR_ADDR = "volnix1gval0validator0genesis0"
GENESIS_PROVIDER_ADDR = "volnix1gprov0provider00genesis0"
# Вне цепочки: резерв симулятора для God Mode (не в genesis-блоке)
SIM_TREASURY_ADDR = "sim_treasury_reserve_godmode"

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
        l_total_genesis = 1.0
        ant_genesis = float(BLOCKS_PER_EPOCH) * l_total_genesis  # §5.5: ANT_genesis = EpochBlocks × L_total_genesis

        gv = self.create_account(GENESIS_VALIDATOR_ADDR)
        gv.role = Role.VALIDATOR
        gv.wrt_balance = 0.0
        gv.lzn_balance = 9999.0
        gv.lzn_frozen_mining = 1.0
        gv.ant_balance = 0.0

        gp = self.create_account(GENESIS_PROVIDER_ADDR)
        gp.role = Role.PROVIDER
        gp.wrt_balance = 0.0
        gp.lzn_balance = 0.0
        gp.ant_balance = ant_genesis

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
                    f"EpochBlocks={BLOCKS_PER_EPOCH}; ANT_genesis=EpochBlocks×L_total={ant_genesis:.0f}."
                ),
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_VALIDATOR_LZN,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=10000.0,
                asset_type="lzn",
                details="§6.3(3): 10 000 LZN на genesis-Валидатора (полная одноразовая эмиссия лицензий).",
                timestamp=ts,
            ).model_dump(mode="json"),
            Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.GENESIS_LZN_ACTIVATE,
                receiver=GENESIS_VALIDATOR_ADDR,
                amount=1.0,
                asset_type="lzn",
                details="§6.3(3): 1 LZN активирован (заморожен под майнинг); 9 999 ликвидных.",
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

    def mint_tokens(self, address: str, amount: float, asset_type="wrt"):
        if address not in self.accounts:
            self.create_account(address)
        
        acc = self.accounts[address]
        
        if asset_type == "ant" and acc.role == Role.GUEST:
            raise ValueError("Guests cannot hold ANT tokens")
            
        if asset_type == "wrt":
            acc.wrt_balance += amount
        elif asset_type == "lzn":
            acc.lzn_balance += amount
        elif asset_type == "ant":
            acc.ant_balance += amount

    def set_role(self, address: str, role: Role):
        if address not in self.accounts:
            self.create_account(address)
        self.accounts[address].role = role

    def get_orderbook(self):
        bids = [o.dict() for o in self.orders.values() if o.order_type == OrderType.BUY]
        asks = [o.dict() for o in self.orders.values() if o.order_type == OrderType.SELL]
        
        # Sort bids descending (highest price first), asks ascending (lowest price first)
        bids.sort(key=lambda x: (-x["price"], x["timestamp"]))
        asks.sort(key=lambda x: (x["price"], x["timestamp"]))
        
        return {"bids": bids[:10], "asks": asks[:10], "last_price": self.last_price, "history": self.price_history}

    def get_full_state(self):
        return {
            "height": self.current_height,
            "mempool_size": len(self.mempool),
            "accounts_count": len(self.accounts),
            "accounts": {addr: acc.dict() for addr, acc in self.accounts.items()},
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
            "accounts": {addr: acc.dict() for addr, acc in self.accounts.items()},
            "orders": {oid: o.dict() for oid, o in self.orders.items()},
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
