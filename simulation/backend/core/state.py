import json
import os
from typing import Dict, List
from core.models import Account, Role, Transaction, Order, OrderType, TransactionType

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

    def init_genesis(self):
        import time
        import uuid
        
        sup = self.create_account("supervisor")
        sup.role = Role.VALIDATOR
        sup.wrt_balance = 0.0
        sup.lzn_balance = 10000.0
        sup.ant_balance = 10000.0

        gen_tx1 = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.MINT,
            receiver="supervisor",
            amount=10000.0,
            asset_type="lzn",
            timestamp=time.time()
        )
        gen_tx2 = Transaction(
            tx_hash=uuid.uuid4().hex,
            tx_type=TransactionType.MINT,
            receiver="supervisor",
            amount=10000.0,
            asset_type="ant",
            timestamp=time.time()
        )
        
        self.blocks.append({
            "height": 0,
            "hash": "000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
            "timestamp": time.time(),
            "transactions": [gen_tx1.dict(), gen_tx2.dict()],
            "tx_count": 2
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
            "current_epoch_burn": self.current_epoch_burn
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
            self.blocks = data.get("blocks", [])
            
            accounts_data = data.get("accounts", {})
            self.accounts = {addr: Account(**acc_data) for addr, acc_data in accounts_data.items()}
            
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
        self.init_genesis()
        if os.path.exists(filepath):
            os.remove(filepath)
