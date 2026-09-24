"""Environment settings for the v2 simulator (prefix VOLNIX_SIM2_)."""

from __future__ import annotations

from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict

_BACKEND_ROOT = Path(__file__).resolve().parent.parent


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_prefix="VOLNIX_SIM2_", extra="ignore")

    data_dir: Path = _BACKEND_ROOT / "data"
    genesis_path: Path = _BACKEND_ROOT / "config" / "genesis.default.json"
    host: str = "0.0.0.0"
    port: int = 8001
    cors_origins: str = "*"
    auto_produce: bool = True
    # Stand helper: enqueue genesis MsgDeclareParticipation each height (canon
    # 5.5-sim requires a fresh declare tx every height — no last_applied replay).
    auto_declare: bool = True
    # Stand-only: wall_sleep = attempt_window / time_scale (default 60 → 1s per canon minute).
    time_scale: float = 60.0
    chain_id: str = "volnix-sim-2"


def load_settings() -> Settings:
    return Settings()
