"""Env-driven конфигурация симуляции (pydantic-settings).

Все настройки читаются из переменных окружения с префиксом VOLNIX_SIM_.
Дефолты сохраняют текущее поведение, чтобы существующий код продолжал работать.
"""
from __future__ import annotations

import os
from typing import List

try:
    from pydantic_settings import BaseSettings, SettingsConfigDict
except ImportError:  # pragma: no cover
    # Fallback: pydantic v1-style without pydantic-settings
    from pydantic import BaseSettings  # type: ignore[no-redef,assignment]

    SettingsConfigDict = dict  # type: ignore[misc,assignment]


class SimSettings(BaseSettings):
    """Настройки симуляции; переопределяются через VOLNIX_SIM_*."""

    model_config = SettingsConfigDict(
        env_prefix="VOLNIX_SIM_",
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    data_dir: str = "data"
    cors_allow_origins: str = "*"
    cors_allow_credentials: bool = True
    log_level: str = "INFO"
    persist_every_n_blocks: int = 1
    snapshot_every_n_blocks: int = 200
    blocks_in_memory: int = 5000
    canon_log_capacity: int = 400
    canon_log_persist: bool = True
    enable_prometheus: bool = True

    # Auto-revive (Phase A): bot + AutoDeclareDaemon стартуют сами при boot.
    bot_autostart: bool = True
    bot_default_intensity: float = 1.0
    auto_declare: bool = True

    # Multi-node sim (Phase B): NetworkSim per-node mempool + gossip.
    num_nodes: int = 1
    gossip_latency_ms: int = 100
    gossip_loss_pct: float = 0.0
    gossip_quorum_pct: float = 2.0 / 3.0
    target_validators: int = 4

    def cors_origins_list(self) -> List[str]:
        raw = (self.cors_allow_origins or "").strip()
        if not raw or raw == "*":
            return ["*"]
        return [item.strip() for item in raw.split(",") if item.strip()]


_settings: SimSettings | None = None


def get_settings() -> SimSettings:
    global _settings
    if _settings is None:
        _settings = SimSettings()
    return _settings


def reload_settings() -> SimSettings:
    """Тесты: пересоздать настройки после изменения env."""
    global _settings
    _settings = SimSettings()
    return _settings


def resolve_data_dir() -> str:
    """Абсолютный путь к каталогу данных (создаётся при первом использовании)."""
    d = get_settings().data_dir
    if not os.path.isabs(d):
        d = os.path.abspath(d)
    os.makedirs(d, exist_ok=True)
    return d
