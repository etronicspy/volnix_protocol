import asyncio
import os
import shutil
import uuid
from contextlib import asynccontextmanager
from typing import Optional

from fastapi import FastAPI, WebSocket
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import PlainTextResponse, Response
from pydantic import BaseModel

from core import analytics, auto_declare, exporters
from core.bot_engine import BotEngine
from core.canon_audit import log_wallet_rejection
from core.engine import SimulationEngine
from core.market_bars import bars_to_echarts_payload, ticks_to_ohlc_bars
from core.metrics import CONTENT_TYPE_LATEST, get_metrics
from core.models import Role
from core.scenarios import (
    ScenarioRunner,
    list_scenarios,
    load_scenario_file,
    run_scenario_file,
)
from core.settings import get_settings
from core.state import SIM_TREASURY_ADDR, StateManager
from core.wallet_validate import validate_and_build_tx, validate_treasury_mint

_sim_settings = get_settings()

state_manager = StateManager()
engine = SimulationEngine(state_manager)
bot_engine = BotEngine(state_manager)


def _mempool_append_persist(tx) -> None:
    """Подать tx в мемпул: NetworkSim (если включён) или legacy.

    Канон-нейтрально: ни одна проверка admission не меняется, только маршрут
    доставки до engine.produce_block.
    """
    net = getattr(state_manager, "network", None)
    addr = getattr(tx, "sender", "") or ""
    pushed = False
    if net is not None:
        try:
            if addr:
                net.submit_from_addr(addr, tx)
            else:
                net.submit_to("node_0", tx)
            pushed = True
        except Exception:
            pushed = False
    if not pushed:
        state_manager.mempool.append(tx)
    state_manager.try_save_state()


@asynccontextmanager
async def _lifespan(app: FastAPI):
    state_manager.load_state()
    engine.set_speed(state_manager.sim_speed)
    if not state_manager.blocks:
        state_manager.init_genesis()
        try:
            state_manager.save_state()
        except OSError as e:
            print(f"Warning: save_state after genesis failed: {e}")

    # NetworkSim: при num_nodes > 1 включаем per-node mempool + gossip.
    if _sim_settings.num_nodes > 1 and getattr(state_manager, "network", None) is None:
        try:
            from core.network import NetworkSim

            net = NetworkSim(
                num_nodes=_sim_settings.num_nodes,
                gossip_latency_ms=_sim_settings.gossip_latency_ms,
                gossip_loss_pct=_sim_settings.gossip_loss_pct,
                quorum_pct=_sim_settings.gossip_quorum_pct,
            )
            net.attach(state_manager)
            state_manager.network = net
        except Exception as e:  # pragma: no cover
            print(f"NetworkSim init failed: {e}")

    engine_task = asyncio.create_task(engine.start())
    bot_task: Optional[asyncio.Task] = None
    auto_declare_task: Optional[asyncio.Task] = None
    if _sim_settings.bot_autostart:
        try:
            bot_engine.set_intensity(float(_sim_settings.bot_default_intensity))
        except Exception:
            pass
        bot_engine.is_running = True
        bot_task = asyncio.create_task(bot_engine.start())
    if _sim_settings.auto_declare:
        # Batch-режим высоких скоростей: демон с тиком ≥0.5 с не успевает за
        # тысячами блоков/с, поэтому движок зовёт step_once перед каждым блоком.
        engine.pre_block_hook = lambda: auto_declare.step_once(state_manager, engine)
        auto_declare_task = asyncio.create_task(
            auto_declare.run(state_manager, engine)
        )
    try:
        yield
    finally:
        state_manager.try_save_state()
        bot_engine.stop()
        engine.stop()
        for task in (auto_declare_task, bot_task, engine_task):
            if task is None:
                continue
            task.cancel()
            try:
                await task
            except (asyncio.CancelledError, Exception):
                pass


app = FastAPI(title="Volnix Simulation API", lifespan=_lifespan)

app.add_middleware(
    CORSMiddleware,
    allow_origins=_sim_settings.cors_origins_list(),
    allow_credentials=_sim_settings.cors_allow_credentials,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/")
def read_root():
    return {
        "status": "Simulation Engine is running", 
        "block_height": state_manager.current_height,
        "block_time": engine.block_time,
        "sim_speed": engine.speed,
        "effective_speed": engine.effective_speed,
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


# --- Панель оператора симуляции (HTTP API) ---

class BlockTimeRequest(BaseModel):
    time_sec: float

@app.post("/api/sim-operator/block-time")
def set_block_time(req: BlockTimeRequest):
    engine.set_block_time(req.time_sec)
    try:
        state_manager.save_state()
    except OSError as e:
        print(f"Warning: save_state after block-time change failed: {e}")
    return {
        "status": "success",
        "new_block_time": engine.block_time,
        "sim_speed": engine.speed,
    }


class SimSpeedRequest(BaseModel):
    """Скорость симуляции: сим. секунд за 1 реальную секунду.

    1 = реальное время (блок раз в 60 с); 604800 = 1 с реального времени ≈ 1 неделя
    симуляции (10080 блоков/с, batch-режим; фактический темп ограничен CPU).
    """

    speed: float


@app.post("/api/sim-operator/speed")
def set_sim_speed(req: SimSpeedRequest):
    engine.set_speed(req.speed)
    try:
        state_manager.save_state()
    except OSError as e:
        print(f"Warning: save_state after speed change failed: {e}")
    return {
        "status": "success",
        "sim_speed": engine.speed,
        "new_block_time": engine.block_time,
        "blocks_per_real_sec": engine.speed / 60.0,
    }


class CreateAccountsRequest(BaseModel):
    count: int

@app.post("/api/sim-operator/accounts")
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

@app.post("/api/sim-operator/mint")
def mint_tokens(req: MintRequest):
    """Как в узле: только tx mint из казначейства → мемпул → исполнение в блоке."""
    ok, msg, tx = validate_treasury_mint(
        state_manager, req.address.strip(), req.amount, req.asset_type
    )
    if not ok or tx is None:
        log_wallet_rejection(state_manager, "mint", msg, req.address.strip())
        return {"status": "error", "message": msg}
    _mempool_append_persist(tx)
    return {
        "status": "queued",
        "message": msg,
        "tx_hash": tx.tx_hash,
        "address": req.address,
    }

class RoleRequest(BaseModel):
    address: str
    role: Role

@app.post("/api/sim-operator/role")
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
    _mempool_append_persist(tx)
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

@app.post("/api/sim-operator/order")
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
    _mempool_append_persist(tx)
    return {"status": "queued", "message": msg, "tx_hash": tx.tx_hash}

@app.post("/api/sim-operator/reset")
async def reset_state():
    fp = state_manager._active_state_path or os.path.join(state_manager.data_dir, "state.json")
    async with engine._state_lock:
        if os.path.isfile(fp):
            try:
                shutil.copy2(fp, fp + ".bak")
            except OSError as e:
                print(f"Warning: could not write {fp}.bak before reset: {e}")
        state_manager.reset_state()
        engine.set_speed(state_manager.sim_speed)
        # Force UI update immediately
        await engine.ws_manager.broadcast({
            "type": "new_block",
            "data": {
                "block": {"height": 0, "transactions": [], "tx_count": 0},
                "state": state_manager.get_full_state(),
                "block_time": engine.block_time
            }
        })
    return {
        "status": "success",
        "message": "State reset; предыдущий state.json при наличии скопирован в state.json.bak",
    }

# --- Bot Engine API ---

class BotControlRequest(BaseModel):
    action: str # "start" or "stop"
    intensity: float = 1.0 # tx/s
    enable_probes: Optional[bool] = None
    probe_ratio: Optional[float] = None
    probe_transfer_ant: Optional[bool] = None
    probe_mint_ant_citizen: Optional[bool] = None
    probe_wrong_role_declare: Optional[bool] = None
    probe_wrong_role_activate_lzn: Optional[bool] = None
    probe_wrong_role_order: Optional[bool] = None
    probe_cancel_not_owned: Optional[bool] = None

@app.post("/api/bot/control")
async def control_bot(req: BotControlRequest):
    if req.action == "start":
        bot_engine.set_intensity(req.intensity)
        bot_engine.set_probe_settings(
            enable=req.enable_probes,
            ratio=req.probe_ratio,
            transfer_ant=req.probe_transfer_ant,
            mint_ant_citizen=req.probe_mint_ant_citizen,
            wrong_role_declare=req.probe_wrong_role_declare,
            wrong_role_activate_lzn=req.probe_wrong_role_activate_lzn,
            wrong_role_order=req.probe_wrong_role_order,
            cancel_not_owned=req.probe_cancel_not_owned,
        )
        if not bot_engine.is_running:
            asyncio.create_task(bot_engine.start())
        return {
            "status": "success",
            "message": "Bot engine started",
            "intensity": bot_engine.tx_per_second,
            "enable_probes": bot_engine.enable_probes,
            "probe_ratio": bot_engine.probe_ratio,
        }
    elif req.action == "stop":
        bot_engine.stop()
        return {"status": "success", "message": "Bot engine stopped"}
    
@app.get("/api/bot/status")
def get_bot_status():
    return {
        "is_running": bot_engine.is_running,
        "intensity": bot_engine.tx_per_second,
        "enable_probes": getattr(bot_engine, "enable_probes", True),
        "probe_ratio": getattr(bot_engine, "probe_ratio", 0.0),
        "probe_transfer_ant": getattr(bot_engine, "probe_transfer_ant", True),
        "probe_mint_ant_citizen": getattr(bot_engine, "probe_mint_ant_citizen", True),
        "probe_wrong_role_declare": getattr(bot_engine, "probe_wrong_role_declare", True),
        "probe_wrong_role_activate_lzn": getattr(bot_engine, "probe_wrong_role_activate_lzn", True),
        "probe_wrong_role_order": getattr(bot_engine, "probe_wrong_role_order", True),
        "probe_cancel_not_owned": getattr(bot_engine, "probe_cancel_not_owned", True),
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
    _mempool_append_persist(tx)
    return {"accepted": True, "message": msg, "tx_hash": tx.tx_hash}


@app.get("/api/wallet/open-orders")
def wallet_open_orders(address: str):
    if not address:
        return {"orders": []}
    return {"orders": state_manager.list_open_orders_for_address(address)}


# --- Explorer: tx / accounts / blocks / canon-log (Этап 2) ---

@app.get("/api/tx/{tx_hash}")
def api_get_tx(tx_hash: str):
    rec = state_manager.get_tx_record(tx_hash)
    if rec is None:
        return {"found": False, "tx_hash": tx_hash}
    return {"found": True, **rec}


@app.get("/api/account/{address}/history")
def api_account_history(address: str, limit: int = 100):
    if not address:
        return {"address": address, "history": []}
    cap = max(1, min(int(limit), 1000))
    history = state_manager.get_account_tx_history(address, limit=cap)
    return {
        "address": address,
        "history": history,
        "open_orders": state_manager.list_open_orders_for_address(address),
    }


@app.get("/api/blocks")
def api_blocks(from_height: Optional[int] = None, to_height: Optional[int] = None, tail: int = 50):
    """Если from/to не заданы — отдаём `tail` последних блоков из RAM."""
    if from_height is not None and to_height is not None:
        return {"blocks": state_manager.get_blocks_range(int(from_height), int(to_height))}
    cap = max(1, min(int(tail), 500))
    return {"blocks": state_manager.blocks[-cap:]}


@app.get("/api/blocks/{height}")
def api_block_by_height(height: int):
    blk = state_manager.get_block_by_height(int(height))
    if blk is None:
        return {"found": False, "height": height}
    return {"found": True, **blk}


class ConsensusFaultBody(BaseModel):
    p_absent: float = 0.0
    p_nil: float = 0.0
    p_double_sign: float = 0.0
    seed: Optional[int] = None


@app.post("/api/sim/consensus")
def api_set_consensus(body: ConsensusFaultBody):
    """Этап 3: модель сбоев консенсуса (PreVote/PreCommit/double_sign).

    Все нули → однопропозерный fallback (поведение Этапа 2 и раньше).
    """
    engine.set_consensus_fault_model(
        p_absent=body.p_absent,
        p_nil=body.p_nil,
        p_double_sign=body.p_double_sign,
        seed=body.seed,
    )
    fm = engine.consensus_fault_model
    return {
        "p_absent": fm.p_absent,
        "p_nil": fm.p_nil,
        "p_double_sign": fm.p_double_sign,
        "seed": fm.seed,
        "enabled": engine.is_consensus_enabled,
    }


@app.get("/api/sim/consensus")
def api_get_consensus():
    fm = engine.consensus_fault_model
    return {
        "p_absent": fm.p_absent,
        "p_nil": fm.p_nil,
        "p_double_sign": fm.p_double_sign,
        "seed": fm.seed,
        "enabled": engine.is_consensus_enabled,
        "evidence_count": len(engine._evidence_log),
    }


@app.get("/api/canon-log")
def api_canon_log(since_id: int = 0, limit: int = 200):
    """Поток canon-аудита.

    Если `since_id` > 0 и persistent ledger включён — отдаём из JSONL;
    иначе — из in-memory ringbuffer (хвост).
    """
    cap = max(1, min(int(limit), 1000))
    ledger = state_manager._canon_ledger
    if since_id > 0 and ledger is not None:
        return {"entries": ledger.read_since(int(since_id), limit=cap)}
    entries = state_manager.canon_log.to_list_newest_first()[:cap]
    return {"entries": entries}


# --- Этап 4: Аналитика / KPI ---


@app.get("/api/analytics/kpi")
def api_analytics_kpi():
    return analytics.kpi_snapshot(state_manager)


@app.get("/api/analytics/gini")
def api_gini(asset: str = "wrt"):
    balances = analytics.balances_by_asset(state_manager, asset)
    return {
        "asset": asset,
        "gini": analytics.gini_coefficient(list(balances.values())),
        "n": len(balances),
    }


# --- Этап 4: Сценарии ---


class ScenarioRunBody(BaseModel):
    path: Optional[str] = None
    yaml: Optional[str] = None
    reset_state: bool = True


@app.get("/api/scenarios")
def api_scenarios_list():
    return {"scenarios": list_scenarios()}


@app.post("/api/scenarios/run")
async def api_scenarios_run(body: ScenarioRunBody):
    if not body.path and not body.yaml:
        return {"ok": False, "message": "expected path or inline yaml"}
    if body.reset_state:
        state_manager.reset_state()
    runner = ScenarioRunner(state_manager, engine, bot_engine)
    if body.path:
        scenario = load_scenario_file(body.path)
    else:
        import yaml as _yaml

        scenario = _yaml.safe_load(body.yaml) or {}
        if not isinstance(scenario, dict):
            return {"ok": False, "message": "yaml must be a mapping"}
    report = await runner.run(scenario)
    return {"ok": report.passed, "report": report.to_dict()}


# --- Этап 4: Экспорт ---


@app.get("/api/export/blocks.jsonl")
def api_export_blocks(from_height: Optional[int] = None, to_height: Optional[int] = None):
    text = exporters.blocks_jsonl(state_manager, from_height, to_height)
    return PlainTextResponse(text, media_type="application/x-ndjson")


@app.get("/api/export/ticks.csv")
def api_export_ticks():
    return PlainTextResponse(exporters.ticks_csv(state_manager), media_type="text/csv")


@app.get("/api/export/balances.csv")
def api_export_balances():
    return PlainTextResponse(exporters.balances_csv(state_manager), media_type="text/csv")


@app.get("/api/export/canon_log.jsonl")
def api_export_canon_log(limit: int = 1000):
    return PlainTextResponse(
        exporters.canon_log_jsonl(state_manager, limit),
        media_type="application/x-ndjson",
    )


# --- Multi-node sim (NetworkSim) ---


class NetworkConfigBody(BaseModel):
    gossip_latency_ms: Optional[int] = None
    gossip_loss_pct: Optional[float] = None
    quorum_pct: Optional[float] = None


def _last_block_for_nodes() -> Optional[dict]:
    if not state_manager.blocks:
        return None
    return state_manager.blocks[-1]


@app.get("/api/network/nodes")
def api_network_nodes():
    net = getattr(state_manager, "network", None)
    if net is None:
        return {"enabled": False, "nodes": []}
    return {"enabled": True, "nodes": net.nodes_summary(_last_block_for_nodes())}


@app.get("/api/network/topology")
def api_network_topology():
    net = getattr(state_manager, "network", None)
    if net is None:
        return {"enabled": False, "config": None, "peers": {}}
    topo = net.topology()
    return {"enabled": True, **topo}


@app.post("/api/network/config")
def api_network_config(body: NetworkConfigBody):
    net = getattr(state_manager, "network", None)
    if net is None:
        return {"ok": False, "message": "NetworkSim is not enabled (set VOLNIX_SIM_NUM_NODES>=2)"}
    cfg = net.set_config(
        gossip_latency_ms=body.gossip_latency_ms,
        gossip_loss_pct=body.gossip_loss_pct,
        quorum_pct=body.quorum_pct,
    )
    return {"ok": True, "config": cfg}


# --- Prometheus ---


@app.get("/metrics")
def api_metrics():
    if not _sim_settings.enable_prometheus:
        return PlainTextResponse("# metrics disabled by env\n", media_type=CONTENT_TYPE_LATEST)
    m = get_metrics()
    m.observe_state(state_manager, engine)
    body = m.render()
    return Response(content=body, media_type=CONTENT_TYPE_LATEST)


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
                "block_time": engine.block_time,
                "sim_speed": engine.speed,
            }
        })
        while True:
            # Keep connection alive and listen for potential client messages
            data = await websocket.receive_text()
    except Exception:
        # Handle all WebSocket disconnects gracefully
        engine.ws_manager.disconnect(websocket)
