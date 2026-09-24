from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field


class BroadcastTxRequest(BaseModel):
    tx: dict[str, Any]


class OperatorAccountRequest(BaseModel):
    seed: str


class OperatorMintRequest(BaseModel):
    address: str
    denom: str = "uwrt"
    amount: int


class OperatorRoleRequest(BaseModel):
    seed: str
    desired_role: str
    zkp_proof: str


class OperatorDeclareRequest(BaseModel):
    seed: str
    b_i: int
    s_i: int


class OperatorOrderRequest(BaseModel):
    seed: str
    market: str
    side: str
    order_type: str = "LIMIT"
    amount: int
    price: int = 0


class ConsensusFaultRequest(BaseModel):
    absent: list[str] = Field(default_factory=list)
    nil_vote: list[str] = Field(default_factory=list)


class ProduceRequest(BaseModel):
    count: int = 1


class ProduceIntervalRequest(BaseModel):
    """Wall-clock seconds between auto-produced blocks (1 ms … 60 s)."""

    interval_sec: float = Field(..., ge=0.001, le=60.0)
