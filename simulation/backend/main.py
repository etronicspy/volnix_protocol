import asyncio
import uuid
from fastapi import FastAPI, WebSocket, WebSocketDisconnect
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from core.engine import SimulationEngine
from core.state import StateManager, SIM_TREASURY_ADDR
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
    state_manager.load_state()
    if not state_manager.blocks:
        state_manager.init_genesis()
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
    asset_type: str = "wrt" # "wrt", "lzn", or "ant"

@app.post("/api/god-mode/mint")
def mint_tokens(req: MintRequest):
    try:
        acc = state_manager.accounts.get(req.address)
        if not acc:
            acc = state_manager.create_account(req.address)
            
        if req.asset_type == "ant" and acc.role == Role.GUEST:
            return {"status": "error", "message": "Guests cannot hold ANT tokens"}

        treasury = state_manager.accounts.get(SIM_TREASURY_ADDR)
        if not treasury:
            return {"status": "error", "message": "Simulation treasury not initialized"}

        if req.asset_type == "wrt" and treasury.wrt_balance >= req.amount:
            treasury.wrt_balance -= req.amount
            acc.wrt_balance += req.amount
        elif req.asset_type == "lzn" and treasury.lzn_balance >= req.amount:
            treasury.lzn_balance -= req.amount
            acc.lzn_balance += req.amount
        elif req.asset_type == "ant" and treasury.ant_balance >= req.amount:
            treasury.ant_balance -= req.amount
            acc.ant_balance += req.amount
        else:
            return {"status": "error", "message": "Simulation treasury has insufficient funds"}
            
        return {
            "status": "success", 
            "address": req.address, 
            "wrt_balance": acc.wrt_balance,
            "lzn_balance": acc.lzn_balance,
            "ant_balance": acc.ant_balance
        }
    except ValueError as e:
        return {"status": "error", "message": str(e)}

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

    acc = state_manager.accounts.get(req.address)
    if req.order_type == "sell" and (not acc or acc.role != Role.PROVIDER):
        return {"status": "error", "message": "Only providers may place SELL orders (§5.2)"}

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
async def control_bot(req: BotControlRequest):
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
    except Exception as e:
        # Handle all WebSocket disconnects gracefully
        engine.ws_manager.disconnect(websocket)
