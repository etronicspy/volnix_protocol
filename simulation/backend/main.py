import asyncio
import uuid
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from core.engine import SimulationEngine
from core.state import StateManager
from core.models import Role
from core.bot_engine import BotEngine

app = FastAPI(title="Volnix Simulation API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

state_manager = StateManager()
engine = SimulationEngine(state_manager)
bot_engine = BotEngine(state_manager)

@app.on_event("startup")
async def startup_event():
    # Load state from file if it exists
    state_manager.load_state()
    asyncio.create_task(engine.start())
    # Bot engine starts stopped by default

@app.on_event("shutdown")
async def shutdown_event():
    # Save state on shutdown
    state_manager.save_state()
    bot_engine.stop()
    engine.stop()

@app.get("/")
def read_root():
    return {
        "status": "Simulation Engine is running", 
        "block_height": state_manager.current_height,
        "block_time": engine.block_time
    }

@app.get("/api/state")
def get_state():
    return state_manager.get_full_state()

# --- God Mode API ---

class BlockTimeRequest(BaseModel):
    time_sec: float

@app.post("/api/god-mode/block-time")
def set_block_time(req: BlockTimeRequest):
    engine.set_block_time(req.time_sec)
    return {"status": "success", "new_block_time": engine.block_time}

class CreateAccountsRequest(BaseModel):
    count: int

@app.post("/api/god-mode/accounts")
def create_accounts(req: CreateAccountsRequest):
    new_accounts = []
    for _ in range(req.count):
        addr = f"volnix{uuid.uuid4().hex[:16]}"
        state_manager.create_account(addr)
        new_accounts.append(addr)
    return {"status": "success", "created": len(new_accounts), "addresses": new_accounts}

class MintRequest(BaseModel):
    address: str
    amount: float
    asset_type: str = "vlnx" # "vlnx" or "shares"

@app.post("/api/god-mode/mint")
def mint_tokens(req: MintRequest):
    state_manager.mint_tokens(req.address, req.amount, req.asset_type)
    return {"status": "success", "address": req.address, "balance": state_manager.accounts[req.address].balance}

class RoleRequest(BaseModel):
    address: str
    role: Role

@app.post("/api/god-mode/role")
def set_role(req: RoleRequest):
    state_manager.set_role(req.address, req.role)
    return {"status": "success", "address": req.address, "role": req.role}

class OrderRequest(BaseModel):
    address: str
    order_type: str # "buy" or "sell"
    price: float
    amount: float

@app.post("/api/god-mode/order")
def create_order(req: OrderRequest):
    from core.models import Transaction, TransactionType, OrderType
    import uuid
    import time
    
    tx = Transaction(
        tx_hash=uuid.uuid4().hex,
        tx_type=TransactionType.CREATE_ORDER,
        sender=req.address,
        order_type=OrderType.BUY if req.order_type == "buy" else OrderType.SELL,
        price=req.price,
        amount=req.amount,
        timestamp=time.time()
    )
    state_manager.mempool.append(tx)
    return {"status": "success", "message": "Order added to mempool"}

@app.post("/api/god-mode/save")
def save_state():
    state_manager.save_state()
    return {"status": "success", "message": "State saved to disk"}

@app.post("/api/god-mode/reset")
async def reset_state():
    state_manager.reset_state()
    # Force UI update immediately
    await engine.ws_manager.broadcast({
        "type": "new_block",
        "data": {
            "block": {"height": 0, "transactions": [], "tx_count": 0},
            "state": state_manager.get_full_state(),
            "block_time": engine.block_time
        }
    })
    return {"status": "success", "message": "State reset and deleted from disk"}

# --- Bot Engine API ---

class BotControlRequest(BaseModel):
    action: str # "start" or "stop"
    intensity: float = 1.0 # tx/s

@app.post("/api/bot/control")
def control_bot(req: BotControlRequest):
    if req.action == "start":
        bot_engine.set_intensity(req.intensity)
        if not bot_engine.is_running:
            asyncio.create_task(bot_engine.start())
        return {"status": "success", "message": "Bot engine started", "intensity": bot_engine.tx_per_second}
    elif req.action == "stop":
        bot_engine.stop()
        return {"status": "success", "message": "Bot engine stopped"}
    
@app.get("/api/bot/status")
def get_bot_status():
    return {
        "is_running": bot_engine.is_running,
        "intensity": bot_engine.tx_per_second
    }

# --- WebSocket ---

@app.websocket("/ws")
async def websocket_endpoint(websocket: WebSocket):
    await engine.ws_manager.connect(websocket)
    try:
        # Send initial state
        await websocket.send_json({
            "type": "init",
            "data": {
                "state": state_manager.get_full_state(),
                "block_time": engine.block_time
            }
        })
        while True:
            # Keep connection alive and listen for potential client messages
            data = await websocket.receive_text()
    except WebSocketDisconnect:
        engine.ws_manager.disconnect(websocket)
