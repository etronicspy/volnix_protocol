"""FastAPI entrypoint for the Volnix simulator v2 node."""

from __future__ import annotations

import uvicorn

from config.settings import load_settings
from volnix.api.app import create_app

settings = load_settings()
app = create_app(settings)


if __name__ == "__main__":
    uvicorn.run("main:app", host=settings.host, port=settings.port, reload=False)
