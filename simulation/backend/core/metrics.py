"""Prometheus метрики (опционально через VOLNIX_SIM_ENABLE_PROMETHEUS).

Безопасное no-op fallback при отсутствии prometheus_client.
"""
from __future__ import annotations

from typing import Optional

try:
    from prometheus_client import (
        CONTENT_TYPE_LATEST,
        CollectorRegistry,
        Counter,
        Gauge,
        Histogram,
        generate_latest,
    )

    HAVE_PROM = True
except ImportError:  # pragma: no cover
    HAVE_PROM = False
    CONTENT_TYPE_LATEST = "text/plain"


class SimMetrics:
    """Контейнер Prometheus-метрик симуляции."""

    def __init__(self) -> None:
        if not HAVE_PROM:
            return
        self.registry = CollectorRegistry()
        self.block_time = Histogram(
            "volnix_sim_block_time_seconds",
            "Длительность блока в секундах (как задано в engine.block_time)",
            buckets=(0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300),
            registry=self.registry,
        )
        self.blocks_total = Counter(
            "volnix_sim_blocks_total",
            "Принятые блоки",
            registry=self.registry,
        )
        self.blocks_rejected_total = Counter(
            "volnix_sim_blocks_rejected_total",
            "Отвергнутые попытки блока (canon §5.4/§6.1)",
            registry=self.registry,
        )
        self.mempool_size = Gauge(
            "volnix_sim_mempool_size",
            "Текущий размер мемпула",
            registry=self.registry,
        )
        self.tps = Gauge(
            "volnix_sim_tps",
            "Последнее значение TPS из tps_history",
            registry=self.registry,
        )
        self.canon_log_events_total = Counter(
            "volnix_sim_canon_log_total",
            "Количество canon-аудит записей",
            ["status", "category"],
            registry=self.registry,
        )
        self.supply = Gauge(
            "volnix_sim_supply",
            "Текущий supply по активу",
            ["asset"],
            registry=self.registry,
        )
        self.gini = Gauge(
            "volnix_sim_gini",
            "Gini-коэффициент по активу",
            ["asset"],
            registry=self.registry,
        )
        self.burn_ratio = Gauge(
            "volnix_sim_burn_ratio",
            "Отношение Σb_i / эмиссия по текущей эпохе",
            registry=self.registry,
        )
        self.accepted_block_ratio = Gauge(
            "volnix_sim_accepted_block_ratio",
            "Доля принятых блоков (commit / попытки)",
            registry=self.registry,
        )
        self.last_price = Gauge(
            "volnix_sim_last_price",
            "Последняя цена ANT в WRT",
            registry=self.registry,
        )

    def render(self) -> bytes:
        if not HAVE_PROM:
            return b"# prometheus_client not installed\n"
        return generate_latest(self.registry)

    def observe_state(self, state_manager, engine) -> None:
        if not HAVE_PROM:
            return
        from core.analytics import (
            accepted_block_ratio,
            balances_by_asset,
            burn_ratio,
            gini_coefficient,
        )

        self.mempool_size.set(len(state_manager.mempool))
        if state_manager.tps_history:
            self.tps.set(float(state_manager.tps_history[-1].get("tps", 0)))
        self.last_price.set(float(state_manager.last_price))
        for asset in ("wrt", "lzn", "ant"):
            balances = balances_by_asset(state_manager, asset)
            self.supply.labels(asset=asset).set(sum(balances.values()))
            self.gini.labels(asset=asset).set(gini_coefficient(list(balances.values())))
        _b, _e, r = burn_ratio(state_manager)
        self.burn_ratio.set(r)
        self.accepted_block_ratio.set(accepted_block_ratio(state_manager))


_metrics: Optional[SimMetrics] = None


def get_metrics() -> SimMetrics:
    global _metrics
    if _metrics is None:
        _metrics = SimMetrics()
    return _metrics
