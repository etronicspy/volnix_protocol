import asyncio
import time
from fastapi import WebSocket
from typing import List

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
from core.state import BLOCKS_PER_EPOCH
import uuid

COEFF_MIN = 0.75
COEFF_MAX = 1.5

class SimulationEngine:
    def __init__(self, state_manager):
        self.state = state_manager
        self.block_time = 5.0 # default 5 seconds
        self.is_running = False
        self.ws_manager = ConnectionManager()

    def set_block_time(self, time_sec: float):
        self.block_time = max(0.1, min(time_sec, 300.0))

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
                
                if buyer and seller and buyer.wrt_balance >= total_cost and seller.ant_balance >= match_amount:
                    buyer.wrt_balance -= total_cost
                    buyer.ant_balance += match_amount
                    seller.ant_balance -= match_amount
                    seller.wrt_balance += total_cost
                    if seller.role == Role.PROVIDER:
                        self.state.epoch_ant_sold_volume += match_amount
                    
                    highest_bid.filled += match_amount
                    lowest_ask.filled += match_amount
                    self.state.last_price = match_price
                    
                    self.state.price_history.append({
                        "time": time.strftime("%H:%M:%S"),
                        "price": match_price
                    })
                    if len(self.state.price_history) > 50:
                        self.state.price_history.pop(0)
                        
                    matched_txs.append({
                        "tx_hash": uuid.uuid4().hex,
                        "tx_type": "trade",
                        "buyer": buyer.address,
                        "seller": seller.address,
                        "price": match_price,
                        "amount": match_amount,
                        "timestamp": time.time()
                    })
                
                # Remove filled orders
                if highest_bid.filled >= highest_bid.amount:
                    bids.pop(0)
                    del self.state.orders[highest_bid.id]
                if lowest_ask.filled >= lowest_ask.amount:
                    asks.pop(0)
                    del self.state.orders[lowest_ask.id]
            else:
                break # No more matches possible
                
        return matched_txs

    def _apply_declare_or_burn(self, tx: Transaction, txs_in_block: list, declared_addrs: set) -> dict:
        """§5.4: b_i + s_i ≤ L_i (L_i = активированный LZN); оба сжигаются. Один declare на валидатора за блок."""
        if tx.sender in declared_addrs:
            return {}
        sender = self.state.accounts.get(tx.sender)
        if not sender or sender.role != Role.VALIDATOR:
            return {}
        if tx.tx_type == TransactionType.BURN:
            b = float(tx.amount or 0)
            s = 0.0
        else:
            b = float(tx.amount or 0)
            s = float(tx.stake_amount or 0)
        L_i = float(sender.lzn_frozen_mining)
        if L_i <= 0 or b < 0 or s < 0:
            return {}
        if b + s > L_i + 1e-9:
            return {}
        total = b + s
        if sender.ant_balance + 1e-9 < total:
            return {}
        sender.ant_balance -= total
        self.state.current_epoch_burn += total
        declared_addrs.add(tx.sender)
        d = tx.model_dump(mode="json")
        d["amount"] = b
        d["stake_amount"] = s
        txs_in_block.append(d)
        return {"b": b, "s": s, "L_i": L_i}

    def _epoch_boundary(self, block_height: int, txs_in_block: list):
        """§5.5: сброс ANT у Поставщиков; эмиссия = sold×coeff; обновление coeff."""
        if block_height <= 0 or block_height % BLOCKS_PER_EPOCH != 0:
            return

        for oid, order in list(self.state.orders.items()):
            owner = self.state.accounts.get(order.owner)
            if not owner:
                del self.state.orders[oid]
                continue
            if order.order_type == OrderType.BUY:
                owner.wrt_balance += order.price * (order.amount - order.filled)
            del self.state.orders[oid]

        providers = [a for a in self.state.accounts.values() if a.role == Role.PROVIDER]
        wiped_balance = sum(p.ant_balance for p in providers)
        for p in providers:
            p.ant_balance = 0.0

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

        if providers and emission > 0:
            per = emission / len(providers)
            for p in providers:
                p.ant_balance += per

        txs_in_block.append({
            "tx_hash": uuid.uuid4().hex,
            "tx_type": TransactionType.EPOCH_EMISSION.value,
            "amount": emission,
            "asset_type": "ant",
            "details": (
                f"§5.5 epoch boundary height {block_height}: wiped provider ANT (balances + SELL escrow) ≈ {wiped_balance:.4f}; "
                f"sold_last_epoch={sold_epoch:.4f}×coeff={coeff:.4f}→emit {emission:.4f} ANT "
                f"evenly among {len(providers)} provider(s); coeff→{self.state.epoch_emission_coefficient:.4f}"
            ),
            "timestamp": time.time(),
        })
        self.state.current_epoch_burn = 0.0

    async def produce_block(self):
        txs_in_block = []
        ts = time.time()
        declared_this_block: set = set()
        participation: dict = {}  # address -> {b, L_i} для §5.1

        # Process mempool
        for tx in self.state.mempool:
            if tx.tx_type == TransactionType.TRANSFER:
                sender = self.state.accounts.get(tx.sender)
                receiver = self.state.accounts.get(tx.receiver)
                
                if sender and receiver and sender.wrt_balance >= tx.amount:
                    sender.wrt_balance -= tx.amount
                    receiver.wrt_balance += tx.amount
                    txs_in_block.append(tx.model_dump(mode="json"))
            elif tx.tx_type == TransactionType.MINT:
                receiver = self.state.accounts.get(tx.receiver)
                if not receiver:
                    receiver = self.state.create_account(tx.receiver)
                
                asset = getattr(tx, "asset_type", "wrt") or "wrt"
                
                if asset == "ant" and receiver.role == Role.GUEST:
                    pass # Invalid, skip
                else:
                    if asset == "wrt":
                        receiver.wrt_balance += tx.amount
                    elif asset == "lzn":
                        receiver.lzn_balance += tx.amount
                    elif asset == "ant":
                        receiver.ant_balance += tx.amount
                    txs_in_block.append(tx.model_dump(mode="json"))
            elif tx.tx_type == TransactionType.SET_ROLE:
                receiver = self.state.accounts.get(tx.receiver)
                if receiver:
                    total_lzn = receiver.lzn_balance + receiver.lzn_frozen_mining
                    if tx.role == Role.VALIDATOR and total_lzn <= 0:
                        pass
                    else:
                        receiver.role = tx.role
                        txs_in_block.append(tx.model_dump(mode="json"))
            elif tx.tx_type in (TransactionType.BURN, TransactionType.DECLARE_PARTICIPATION):
                info = self._apply_declare_or_burn(tx, txs_in_block, declared_this_block)
                if info and tx.sender:
                    participation[tx.sender] = {"b": info["b"], "L_i": info["L_i"]}
            elif tx.tx_type == TransactionType.CREATE_ORDER:
                owner = self.state.accounts.get(tx.sender)
                if owner and owner.role != Role.GUEST:
                    if tx.order_type == OrderType.BUY and owner.wrt_balance >= (tx.price * tx.amount):
                        owner.wrt_balance -= (tx.price * tx.amount)
                        order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                        self.state.orders[tx.tx_hash] = order
                        txs_in_block.append(tx.model_dump(mode="json"))
                    elif tx.order_type == OrderType.SELL:
                        if owner.role != Role.PROVIDER:
                            pass
                        elif owner.ant_balance >= tx.amount:
                            owner.ant_balance -= tx.amount
                            order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                            self.state.orders[tx.tx_hash] = order
                            txs_in_block.append(tx.model_dump(mode="json"))
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
        
        # Run Matching Engine
        trades = self._match_orders()
        txs_in_block.extend(trades)
        
        # §5.1: базовая награда WRT только среди валидаторов с b_i > 0; доля ∝ активированному LZN (L_i)
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
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.BLOCK_REWARD.value,
                "receiver": "validators",
                "amount": reward_amount,
                "asset_type": "wrt",
                "details": "§5.1: WRT block reward only if b_i>0; split by activated LZN among burners this height",
                "timestamp": time.time(),
            })

        next_height = self.state.current_height + 1
        self._epoch_boundary(next_height, txs_in_block)

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
        
        self.state.mempool = [] # clear mempool after processing
        self.state.add_block(block)

        
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
