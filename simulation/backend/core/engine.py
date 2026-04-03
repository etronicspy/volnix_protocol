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
import uuid

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

    async def produce_block(self):
        # Process mempool
        txs_in_block = []
        for tx in self.state.mempool:
            if tx.tx_type == TransactionType.TRANSFER:
                sender = self.state.accounts.get(tx.sender)
                receiver = self.state.accounts.get(tx.receiver)
                
                if sender and receiver and sender.wrt_balance >= tx.amount:
                    sender.wrt_balance -= tx.amount
                    receiver.wrt_balance += tx.amount
                    txs_in_block.append(tx.dict())
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
                    txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.SET_ROLE:
                receiver = self.state.accounts.get(tx.receiver)
                if receiver:
                    if tx.role == Role.VALIDATOR and receiver.lzn_balance <= 0:
                        pass # Cannot become validator without LZN
                    else:
                        receiver.role = tx.role
                        txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.BURN:
                sender = self.state.accounts.get(tx.sender)
                if sender and sender.role == Role.VALIDATOR and sender.ant_balance >= tx.amount:
                    sender.ant_balance -= tx.amount
                    self.state.current_epoch_burn += tx.amount
                    txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.CREATE_ORDER:
                owner = self.state.accounts.get(tx.sender)
                if owner and owner.role != Role.GUEST:
                    # Basic validation
                    if tx.order_type == OrderType.BUY and owner.wrt_balance >= (tx.price * tx.amount):
                        # Lock funds (simplified: we just check at match time, but let's lock for realism)
                        owner.wrt_balance -= (tx.price * tx.amount)
                        order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                        self.state.orders[tx.tx_hash] = order
                        txs_in_block.append(tx.dict())
                    elif tx.order_type == OrderType.SELL and owner.ant_balance >= tx.amount:
                        owner.ant_balance -= tx.amount
                        order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                        self.state.orders[tx.tx_hash] = order
                        txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.CANCEL_ORDER:
                order = self.state.orders.get(tx.order_id)
                if order and order.owner == tx.sender:
                    owner = self.state.accounts.get(tx.sender)
                    if order.order_type == OrderType.BUY:
                        owner.wrt_balance += (order.price * (order.amount - order.filled))
                    else:
                        owner.ant_balance += (order.amount - order.filled)
                    del self.state.orders[tx.order_id]
                    txs_in_block.append(tx.dict())
        
        # Run Matching Engine
        trades = self._match_orders()
        txs_in_block.extend(trades)
        
        # Block Reward (WRT Emission)
        reward_amount = 50.0
        validators = [acc for acc in self.state.accounts.values() if acc.role == Role.VALIDATOR and acc.lzn_balance > 0]
        
        if validators:
            total_lzn = sum(v.lzn_balance for v in validators)
            for v in validators:
                share = (v.lzn_balance / total_lzn) * reward_amount
                v.wrt_balance += share
                
            details = "Block reward distributed to validators"
            if self.state.current_height == 0: # Producing Block 1
                details = "The Times 03/Apr/2026 Volnix Protocol Genesis: Overcoming the Justice Trilemma"
                
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.BLOCK_REWARD,
                "receiver": "validators",
                "amount": reward_amount,
                "asset_type": "wrt",
                "details": details,
                "timestamp": time.time()
            })
        
        # Epoch Logic (every 10 blocks)
        if self.state.current_height > 0 and self.state.current_height % 10 == 0:
            providers = [acc for acc in self.state.accounts.values() if acc.role == Role.PROVIDER]
            
            wiped_ant = 0.0
            for p in providers:
                wiped_ant += p.ant_balance
                p.ant_balance = 0.0
                
            # Distribute new ANT based on previous burn
            emission = max(self.state.current_epoch_burn * 1.0, 100.0) # Base emission for sim
            if providers:
                per_provider = emission / len(providers)
                for p in providers:
                    p.ant_balance += per_provider
                    
            txs_in_block.append({
                "tx_hash": uuid.uuid4().hex,
                "tx_type": TransactionType.EPOCH_EMISSION,
                "amount": emission,
                "asset_type": "ant",
                "details": f"Wiped {wiped_ant:.2f} ANT. Emitted {emission:.2f} ANT to {len(providers)} providers",
                "timestamp": time.time()
            })
            self.state.current_epoch_burn = 0.0
        
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
