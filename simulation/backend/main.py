import asyncio
import uuid
from typing import Optional

from fastapi import FastAPI, WebSocket
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel

from core.engine import SimulationEngine
from core.state import StateManager, SIM_TREASURY_ADDR
from core.models import Role
from core.bot_engine import BotEngine
from core.wallet_validate import validate_and_build_tx, validate_treasury_mint
from core.canon_audit import log_wallet_rejection
from core.market_bars import bars_to_echarts_payload, ticks_to_ohlc_bars

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


@app.get("/api/market/history")
def api_market_history(limit: int = 10_000):
    """История тиков цены для графика/виджета (хвост до limit, max 50_000).

    Формат тика: time (строка), price (float), ts (unix_seconds, желательно).
    Поле ts обязательно для корректной агрегации; для Apache ECharts см. GET /api/market/bars.
    """
    cap = max(1, min(int(limit), 50_000))
    hist = state_manager.price_history
    if len(hist) > cap:
        hist = hist[-cap:]
    return {
        "last_price": state_manager.last_price,
        "history": hist,
    }


@app.get("/api/market/bars")
def api_market_bars(interval_sec: int = 0, limit_ticks: int = 50_000):
    """OHLC в формате Apache ECharts candlestick: category[], values[][open,close,low,high], times[].

    interval_sec: 0 — одна свеча на сделку; 1, 60, 300, … — корзина в секундах.
    """
    cap = max(1, min(int(limit_ticks), 50_000))
    hist = state_manager.price_history
    if len(hist) > cap:
        hist = hist[-cap:]
    iv = int(interval_sec)
    bars = ticks_to_ohlc_bars(hist, iv)
    payload = bars_to_echarts_payload(bars, trade_mode=(iv <= 0))
    return {
        "interval_sec": iv,
        "last_price": state_manager.last_price,
        "bar_count": len(bars),
        **payload,
    }


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
    """Как в узле: только tx mint из казначейства → мемпул → исполнение в блоке."""
    ok, msg, tx = validate_treasury_mint(
        state_manager, req.address.strip(), req.amount, req.asset_type
    )
    if not ok or tx is None:
        log_wallet_rejection(state_manager, "mint", msg, req.address.strip())
        return {"status": "error", "message": msg}
    state_manager.mempool.append(tx)
    return {
        "status": "queued",
        "message": msg,
        "tx_hash": tx.tx_hash,
        "address": req.address,
    }

class RoleRequest(BaseModel):
    address: str
    role: Role

@app.post("/api/god-mode/role")
def set_role(req: RoleRequest):
    """Те же правила, что у кошелька: set_role только через мемпул."""
    ok, msg, tx = validate_and_build_tx(
        state_manager,
        "set_role",
        req.address.strip(),
        role=req.role,
    )
    if not ok or tx is None:
        log_wallet_rejection(state_manager, "set_role", msg, req.address.strip())
        return {"status": "error", "message": msg}
    state_manager.mempool.append(tx)
    return {
        "status": "queued",
        "message": msg,
        "tx_hash": tx.tx_hash,
        "address": req.address,
        "role": req.role.value,
    }

class OrderRequest(BaseModel):
    address: str
    order_type: str # "buy" or "sell"
    price: Optional[float] = None
    amount: float
    market: bool = False
    max_wrt: Optional[float] = None

@app.post("/api/god-mode/order")
def create_order(req: OrderRequest):
    """Те же проверки, что у кошелька (баланс, роль SELL)."""
    ok, msg, tx = validate_and_build_tx(
        state_manager,
        "create_order",
        req.address.strip(),
        side=req.order_type,
        price=req.price,
        amount=req.amount,
        market=bool(req.market),
        max_wrt=req.max_wrt,
    )
    if not ok or tx is None:
        log_wallet_rejection(state_manager, "create_order", msg, req.address.strip())
        return {"status": "error", "message": msg}
    state_manager.mempool.append(tx)
    return {"status": "queued", "message": msg, "tx_hash": tx.tx_hash}

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

# --- Wallet (on-chain style: validate → mempool → next block) ---

class WalletSubmitBody(BaseModel):
    op: str
    address: str
    to_address: Optional[str] = None
    amount: Optional[float] = None
    asset: str = "wrt"
    role: Optional[Role] = None
    price: Optional[float] = None
    order_id: Optional[str] = None
    side: Optional[str] = None
    burn_b: Optional[float] = None
    stake_s: Optional[float] = None
    market: bool = False
    max_wrt: Optional[float] = None


@app.post("/api/wallet/submit")
def wallet_submit_tx(body: WalletSubmitBody):
    ok, msg, tx = validate_and_build_tx(
        state_manager,
        body.op,
        body.address.strip(),
        to_address=(body.to_address or "").strip() or None,
        amount=body.amount,
        asset=body.asset or "wrt",
        role=body.role,
        price=body.price,
        order_id=body.order_id,
        side=body.side,
        burn_b=body.burn_b,
        stake_s=body.stake_s,
        market=bool(body.market),
        max_wrt=body.max_wrt,
    )
    if not ok or tx is None:
        log_wallet_rejection(state_manager, body.op, msg, body.address.strip())
        return {"accepted": False, "message": msg}
    state_manager.mempool.append(tx)
    return {"accepted": True, "message": msg, "tx_hash": tx.tx_hash}


@app.get("/api/wallet/open-orders")
def wallet_open_orders(address: str):
    if not address:
        return {"orders": []}
    return {"orders": state_manager.list_open_orders_for_address(address)}

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
