from __future__ import annotations

from contextlib import asynccontextmanager
from pathlib import Path
from typing import Optional

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from config.settings import Settings
from volnix.api import explorer, rpc, ws
from volnix.node.node import Node


def create_app(settings: Optional[Settings] = None, auto_produce: Optional[bool] = None) -> FastAPI:
    settings = settings or Settings()
    produce = settings.auto_produce if auto_produce is None else auto_produce

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        node: Node = app.state.node
        node.load_or_init()
        node.subscribe(ws.hub.broadcast)
        if produce:
            node.start()
        try:
            yield
        finally:
            await node.stop()

    app = FastAPI(title="Volnix Simulator v2", version="0.1.0", lifespan=lifespan)
    origins = [o.strip() for o in settings.cors_origins.split(",") if o.strip()]
    app.add_middleware(
        CORSMiddleware,
        allow_origins=origins or ["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )
    node = Node(
        data_dir=Path(settings.data_dir),
        genesis_path=Path(settings.genesis_path),
        produce_interval=settings.produce_interval,
        auto_declare=settings.auto_declare,
    )
    app.state.node = node
    app.state.settings = settings
    app.include_router(rpc.router)
    app.include_router(explorer.router)
    app.include_router(ws.router)

    @app.get("/")
    def root():
        n: Node = app.state.node
        return {
            "name": "volnix-sim-2",
            "chain_id": n.app.state.chain_id,
            "height": n.app.state.height,
            "app_hash": n.app.state.app_hash() if n.app.state.accounts else "",
        }

    return app
