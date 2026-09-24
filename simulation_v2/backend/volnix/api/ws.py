from __future__ import annotations

import asyncio
from typing import Any

from fastapi import APIRouter, WebSocket, WebSocketDisconnect

router = APIRouter()


class Hub:
    def __init__(self) -> None:
        self._clients: set[WebSocket] = set()

    async def connect(self, ws: WebSocket) -> None:
        await ws.accept()
        self._clients.add(ws)

    def disconnect(self, ws: WebSocket) -> None:
        self._clients.discard(ws)

    async def broadcast(self, event: dict[str, Any]) -> None:
        dead: list[WebSocket] = []
        for ws in list(self._clients):
            try:
                await ws.send_json(event)
            except Exception:
                dead.append(ws)
        for ws in dead:
            self._clients.discard(ws)


hub = Hub()


@router.websocket("/ws")
async def websocket_endpoint(ws: WebSocket) -> None:
    await hub.connect(ws)
    node = ws.app.state.node
    try:
        await ws.send_json(
            {
                "type": "init",
                "height": node.app.state.height,
                "chain_id": node.app.state.chain_id,
                "app_hash": node.app.state.app_hash(),
            }
        )
        while True:
            await asyncio.sleep(30)
            await ws.send_json({"type": "ping"})
    except WebSocketDisconnect:
        hub.disconnect(ws)
    except Exception:
        hub.disconnect(ws)
