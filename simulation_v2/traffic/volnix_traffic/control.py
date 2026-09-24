"""Control FastAPI for the traffic process."""

from __future__ import annotations

from contextlib import asynccontextmanager
from typing import Any, AsyncIterator, Dict, Optional

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from volnix_traffic.runtime import TrafficRuntime


class StartBody(BaseModel):
    intensity: Optional[float] = None


class IntensityBody(BaseModel):
    intensity: float = Field(..., ge=0.0, le=100.0)


def create_control_app(runtime: TrafficRuntime) -> FastAPI:
    @asynccontextmanager
    async def lifespan(_app: FastAPI) -> AsyncIterator[None]:
        await runtime.ensure_started()
        yield
        await runtime.aclose()

    app = FastAPI(title="Volnix Sim2 Traffic", version="0.1.0", lifespan=lifespan)
    app.add_middleware(
        CORSMiddleware,
        allow_origins=["*"],
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    @app.get("/status")
    def status() -> Dict[str, Any]:
        return runtime.status()

    @app.post("/start")
    def start(body: Optional[StartBody] = None) -> Dict[str, Any]:
        intensity = body.intensity if body else None
        runtime.start(intensity)
        return runtime.status()

    @app.post("/stop")
    def stop() -> Dict[str, Any]:
        runtime.stop()
        return runtime.status()

    @app.post("/intensity")
    def intensity(body: IntensityBody) -> Dict[str, Any]:
        runtime.bots.set_intensity(body.intensity)
        return runtime.status()

    @app.get("/wallets")
    def wallets() -> Dict[str, Any]:
        return {"count": len(runtime.registry), "wallets": runtime.wallets()}

    @app.post("/tick")
    async def tick() -> Dict[str, Any]:
        """Force one height tick (useful when auto_produce is off)."""
        return await runtime.tick_once()

    return app
