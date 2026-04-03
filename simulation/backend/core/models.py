from pydantic import BaseModel
from typing import Optional, List, Dict
from enum import Enum

class Role(str, Enum):
    CITIZEN = "citizen"
    PROVIDER = "provider"
    VALIDATOR = "validator"
    GUEST = "guest"

class Account(BaseModel):
    address: str
    wrt_balance: float = 0.0  # WRT (Wert) - Main token
    lzn_balance: float = 0.0  # LZN (Lizenz) - License token
    ant_balance: float = 0.0  # ANT (Anteil) - Internal market coin
    role: Role = Role.CITIZEN

class TransactionType(str, Enum):
    TRANSFER = "transfer"
    MINT = "mint"
    SET_ROLE = "set_role"
    CREATE_ORDER = "create_order"
    CANCEL_ORDER = "cancel_order"
    BURN = "burn"
    EPOCH_EMISSION = "epoch_emission"
    BLOCK_REWARD = "block_reward"

class OrderType(str, Enum):
    BUY = "buy"
    SELL = "sell"

class Order(BaseModel):
    id: str
    owner: str
    order_type: OrderType
    price: float  # VLNX per Share
    amount: float # Number of shares
    filled: float = 0.0
    timestamp: float

class Transaction(BaseModel):
    tx_hash: str
    tx_type: TransactionType
    sender: Optional[str] = None
    receiver: Optional[str] = None
    amount: Optional[float] = None
    asset_type: Optional[str] = None
    price: Optional[float] = None
    order_type: Optional[OrderType] = None
    order_id: Optional[str] = None
    role: Optional[Role] = None
    timestamp: float
