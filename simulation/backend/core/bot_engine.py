import asyncio
import random
import time
import uuid
from core.state import StateManager
from core.models import Role, Transaction, TransactionType

class BotEngine:
    def __init__(self, state_manager: StateManager):
        self.state = state_manager
        self.is_running = False
        self.tx_per_second = 1.0

    def set_intensity(self, tx_per_second: float):
        self.tx_per_second = max(0.1, min(tx_per_second, 100.0))

    async def start(self):
        self.is_running = True
        print(f"Bot Engine started. Intensity: {self.tx_per_second} tx/s")
        while self.is_running:
            try:
                self.generate_traffic()
            except Exception as e:
                print(f"Bot error: {e}")
            # Sleep based on intensity
            await asyncio.sleep(1.0 / self.tx_per_second)

    def stop(self):
        self.is_running = False
        print("Bot Engine stopped.")

    def generate_traffic(self):
        accounts = list(self.state.accounts.values())
        
        # 1. Ensure we have enough accounts (auto-generate if needed, max 100)
        if len(accounts) < 10 or (len(accounts) < 100 and random.random() < 0.05):
            new_addr = f"bot_{uuid.uuid4().hex[:8]}"
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.MINT,
                receiver=new_addr,
                amount=round(random.uniform(50, 200), 2),
                asset_type="wrt",
                timestamp=time.time()
            )
            supervisor = self.state.accounts.get("supervisor")
            if supervisor and supervisor.wrt_balance >= tx.amount:
                self.state.mempool.append(tx)
            return

        # 2. Choose random action based on weights
        # 30% transfer, 25% trade, 10% cancel_order, 15% mint, 10% set_role, 10% burn
        action = random.choices(
            ["transfer", "trade", "cancel_order", "mint", "set_role", "burn"], 
            weights=[0.30, 0.25, 0.10, 0.15, 0.10, 0.10]
        )[0]

        if action == "burn":
            validators = [acc for acc in accounts if acc.role == Role.VALIDATOR and acc.ant_balance > 0.1]
            if validators:
                val = random.choice(validators)
                amount = round(random.uniform(0.1, val.ant_balance * 0.5), 2)
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.BURN,
                    sender=val.address,
                    amount=amount,
                    asset_type="ant",
                    timestamp=time.time()
                )
                self.state.mempool.append(tx)
            return

        elif action == "set_role":
            target = random.choice(accounts)
            new_role = random.choice([Role.CITIZEN, Role.PROVIDER, Role.VALIDATOR, Role.GUEST])
            
            # Rule: Cannot become validator without LZN
            if new_role == Role.VALIDATOR and target.lzn_balance <= 0:
                new_role = Role.PROVIDER
                
            if target.role != new_role:
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.SET_ROLE,
                    receiver=target.address,
                    role=new_role,
                    timestamp=time.time()
                )
                self.state.mempool.append(tx)
            return

        elif action == "mint":
            target = random.choice(accounts)
            possible_assets = ["wrt", "lzn"]
            if target.role != Role.GUEST:
                possible_assets.append("ant")
            
            asset = random.choice(possible_assets)
            amount = round(random.uniform(10, 100), 2)
            
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.MINT,
                receiver=target.address,
                amount=amount,
                asset_type=asset,
                timestamp=time.time()
            )
            
            supervisor = self.state.accounts.get("supervisor")
            if supervisor:
                if asset == "wrt" and supervisor.wrt_balance >= amount:
                    self.state.mempool.append(tx)
                elif asset == "lzn" and supervisor.lzn_balance >= amount:
                    self.state.mempool.append(tx)
                elif asset == "ant" and supervisor.ant_balance >= amount:
                    self.state.mempool.append(tx)
            return

        elif action == "transfer":
            valid_senders = [acc for acc in accounts if acc.wrt_balance > 0.1]
            if not valid_senders:
                return
            sender = random.choice(valid_senders)
            
            possible_receivers = [acc for acc in accounts if acc.address != sender.address]
            if not possible_receivers:
                return
            receiver = random.choice(possible_receivers)

            amount = round(random.uniform(0.1, sender.wrt_balance * 0.2), 2)
            if amount < 0.1:
                amount = sender.wrt_balance
                
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.TRANSFER,
                sender=sender.address,
                receiver=receiver.address,
                amount=amount,
                timestamp=time.time()
            )
            self.state.mempool.append(tx)
            return

        elif action == "cancel_order":
            if not self.state.orders:
                return
            order_to_cancel = random.choice(list(self.state.orders.values()))
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.CANCEL_ORDER,
                sender=order_to_cancel.owner,
                order_id=order_to_cancel.id,
                timestamp=time.time()
            )
            self.state.mempool.append(tx)
            return

        elif action == "trade":
            valid_traders = [acc for acc in accounts if acc.role != Role.GUEST]
            if not valid_traders:
                return
            trader = random.choice(valid_traders)
            
            order_type = random.choice([OrderType.BUY, OrderType.SELL])
            base_price = self.state.last_price if self.state.last_price > 0 else 10.0
            price = round(base_price * random.uniform(0.8, 1.2), 2)
            shares_amount = round(random.uniform(1, 20), 2)
            
            if order_type == OrderType.BUY and trader.wrt_balance >= (price * shares_amount):
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.CREATE_ORDER,
                    sender=trader.address,
                    order_type=order_type,
                    price=price,
                    amount=shares_amount,
                    timestamp=time.time()
                )
                self.state.mempool.append(tx)
            elif order_type == OrderType.SELL and trader.ant_balance >= shares_amount:
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.CREATE_ORDER,
                    sender=trader.address,
                    order_type=order_type,
                    price=price,
                    amount=shares_amount,
                    timestamp=time.time()
                )
                self.state.mempool.append(tx)
            return
