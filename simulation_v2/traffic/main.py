#!/usr/bin/env python3
"""Run the simulation_v2 traffic / economy process.

Requires the node at VOLNIX_SIM2_TRAFFIC_NODE_URL (default http://127.0.0.1:8001).

    cd simulation_v2/traffic
    pip install -r requirements.txt
    python3 main.py
"""

from __future__ import annotations

import logging
import sys
from pathlib import Path

import uvicorn

# Allow `python main.py` without installing the package.
sys.path.insert(0, str(Path(__file__).resolve().parent))

from volnix_traffic.control import create_control_app
from volnix_traffic.runtime import TrafficRuntime
from volnix_traffic.settings import TrafficSettings


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    settings = TrafficSettings.load()
    runtime = TrafficRuntime(settings)
    app = create_control_app(runtime)
    uvicorn.run(
        app,
        host=settings.control_host,
        port=settings.control_port,
        log_level="info",
    )


if __name__ == "__main__":
    main()
