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
                
                if buyer and seller and buyer.balance >= total_cost and seller.shares >= match_amount:
                    buyer.balance -= total_cost
                    buyer.shares += match_amount
                    seller.shares -= match_amount
                    seller.balance += total_cost
                    
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
                
                if sender and receiver and sender.role != Role.NONE and sender.balance >= tx.amount:
                    sender.balance -= tx.amount
                    receiver.balance += tx.amount
                    txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.MINT:
                receiver = self.state.accounts.get(tx.receiver)
                if receiver:
                    receiver.balance += tx.amount
                    txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.SET_ROLE:
                receiver = self.state.accounts.get(tx.receiver)
                if receiver:
                    receiver.role = tx.role
                    txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.CREATE_ORDER:
                owner = self.state.accounts.get(tx.sender)
                if owner and owner.role != Role.NONE:
                    # Basic validation
                    if tx.order_type == OrderType.BUY and owner.balance >= (tx.price * tx.amount):
                        # Lock funds (simplified: we just check at match time, but let's lock for realism)
                        owner.balance -= (tx.price * tx.amount)
                        order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                        self.state.orders[tx.tx_hash] = order
                        txs_in_block.append(tx.dict())
                    elif tx.order_type == OrderType.SELL and owner.shares >= tx.amount:
                        owner.shares -= tx.amount
                        order = Order(id=tx.tx_hash, owner=tx.sender, order_type=tx.order_type, price=tx.price, amount=tx.amount, timestamp=tx.timestamp)
                        self.state.orders[tx.tx_hash] = order
                        txs_in_block.append(tx.dict())
            elif tx.tx_type == TransactionType.CANCEL_ORDER:
                order = self.state.orders.get(tx.order_id)
                if order and order.owner == tx.sender:
                    owner = self.state.accounts.get(tx.sender)
                    if order.order_type == OrderType.BUY:
                        owner.balance += (order.price * (order.amount - order.filled))
                    else:
                        owner.shares += (order.amount - order.filled)
                    del self.state.orders[tx.order_id]
                    txs_in_block.append(tx.dict())
        
        # Run Matching Engine
        trades = self._match_orders()
        txs_in_block.extend(trades)
        
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
