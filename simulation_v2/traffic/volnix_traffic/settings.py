"""Traffic process settings (env prefix VOLNIX_SIM2_TRAFFIC_)."""

from __future__ import annotations

from pathlib import Path
from typing import Any, Dict, Optional

import yaml
from pydantic import BaseModel, Field
from pydantic_settings import BaseSettings, SettingsConfigDict

_DEFAULT_YAML = Path(__file__).resolve().parent.parent / "config" / "default.yaml"


class Weights(BaseModel):
    transfer: float = 0.28
    trade_ant: float = 0.22
    trade_lzn: float = 0.10
    activate_lzn: float = 0.10
    cancel_order: float = 0.10
    prep: float = 0.20


class TrafficSettings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="VOLNIX_SIM2_TRAFFIC_",
        env_file=".env",
        extra="ignore",
    )

    node_url: str = "http://127.0.0.1:8001"
    control_host: str = "127.0.0.1"
    control_port: int = 8002
    autostart: bool = True
    intensity: float = 2.0
    target_wallets: int = 30
    bootstrap_wrt: int = 5_000_000
    min_bot_wrt: int = 500_000
    max_actions_per_tick: int = 40
    poll_interval_sec: float = 0.25
    enable_market: bool = True
    enable_bots: bool = True
    enable_declare: bool = True
    citizen_fraction: float = 0.25
    weights: Weights = Field(default_factory=Weights)
    config_path: Optional[Path] = None

    @classmethod
    def load(cls, path: Optional[Path] = None) -> "TrafficSettings":
        cfg_path = path or _DEFAULT_YAML
        data: Dict[str, Any] = {}
        if cfg_path.is_file():
            with cfg_path.open() as f:
                raw = yaml.safe_load(f) or {}
            if isinstance(raw, dict):
                data = raw
        # Env / constructor overrides YAML
        inst = cls(**{k: v for k, v in data.items() if k != "weights"})
        if "weights" in data and isinstance(data["weights"], dict):
            inst.weights = Weights(**data["weights"])
        inst.config_path = cfg_path
        # Re-apply env on top (BaseSettings already did for fields present;
        # rebuild from env after YAML for correct precedence).
        env_inst = cls()
        for name in cls.model_fields:
            if name in ("weights", "config_path"):
                continue
            env_val = getattr(env_inst, name)
            default = cls.model_fields[name].default
            if env_val != default:
                setattr(inst, name, env_val)
        return inst
