from pydantic import BaseModel
from typing import Optional, List, Dict
from enum import Enum

class Role(str, Enum):
    NONE = "none"
    CITIZEN = "citizen"
    PROVIDER = "provider"
    VALIDATOR = "validator"

class Account(BaseModel):
    address: str
    balance: float = 0.0  # VLNX tokens
    shares: float = 0.0   # Anteil shares
    role: Role = Role.NONE

class TransactionType(str, Enum):
    TRANSFER = "transfer"
    MINT = "mint"
    SET_ROLE = "set_role"
    CREATE_ORDER = "create_order"
    CANCEL_ORDER = "cancel_order"

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
    price: Optional[float] = None
    order_type: Optional[OrderType] = None
    order_id: Optional[str] = None
    role: Optional[Role] = None
    timestamp: float
