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
            self.generate_traffic()
            # Sleep based on intensity
            await asyncio.sleep(1.0 / self.tx_per_second)

    def stop(self):
        self.is_running = False
        print("Bot Engine stopped.")

    def generate_traffic(self):
        accounts = list(self.state.accounts.values())
        if len(accounts) < 2:
            return

        # Rule: Only accounts with a specific role (Citizen, Provider, Validator) can SEND transactions.
        # Accounts with role 'NONE' cannot participate in the active economy as senders.
        valid_senders = [acc for acc in accounts if acc.role != Role.NONE and acc.balance > 0.1]
        
        if not valid_senders:
            return

        sender = random.choice(valid_senders)
        
        # Receiver can be anyone else
        possible_receivers = [acc for acc in accounts if acc.address != sender.address]
        if not possible_receivers:
            return
            
        receiver = random.choice(possible_receivers)

        # Random amount between 0.1 and 10% of sender's balance
        max_amount = sender.balance * 0.1
        if max_amount < 0.1:
            amount = sender.balance # send all if very little
        else:
            amount = round(random.uniform(0.1, max_amount), 2)

        # Decide action: 70% transfer, 30% trade
        action = random.choices(["transfer", "trade"], weights=[0.7, 0.3])[0]

        if action == "transfer":
            tx = Transaction(
                tx_hash=uuid.uuid4().hex,
                tx_type=TransactionType.TRANSFER,
                sender=sender.address,
                receiver=receiver.address,
                amount=amount,
                timestamp=time.time()
            )
            self.state.mempool.append(tx)
        else:
            # Create a buy or sell order
            order_type = random.choice([OrderType.BUY, OrderType.SELL])
            
            # Use last price as a baseline, +/- 10%
            base_price = self.state.last_price if self.state.last_price > 0 else 10.0
            price = round(base_price * random.uniform(0.9, 1.1), 2)
            shares_amount = round(random.uniform(1, 10), 2)
            
            if order_type == OrderType.BUY and sender.balance >= (price * shares_amount):
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.CREATE_ORDER,
                    sender=sender.address,
                    order_type=order_type,
                    price=price,
                    amount=shares_amount,
                    timestamp=time.time()
                )
                self.state.mempool.append(tx)
            elif order_type == OrderType.SELL and sender.shares >= shares_amount:
                tx = Transaction(
                    tx_hash=uuid.uuid4().hex,
                    tx_type=TransactionType.CREATE_ORDER,
                    sender=sender.address,
                    order_type=order_type,
                    price=price,
                    amount=shares_amount,
                    timestamp=time.time()
                )
                self.state.mempool.append(tx)
