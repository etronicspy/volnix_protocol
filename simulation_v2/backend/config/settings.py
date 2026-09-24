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
    # Wall-clock seconds between produced blocks (canonical time still advances BaseBlockTime).
    produce_interval: float = 1.0
    auto_produce: bool = True
    # Enqueue MsgDeclareParticipation for the genesis validator before each block
    # so PoVB burns f_i / b_i+s_i on the stand (empty blocks burn nothing per canon).
    auto_declare: bool = True
    chain_id: str = "volnix-sim-2"


def load_settings() -> Settings:
    return Settings()
