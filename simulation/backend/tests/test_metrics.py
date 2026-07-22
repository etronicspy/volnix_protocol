"""Тесты Prometheus-обёртки `core.metrics` (skip без prometheus_client)."""
from __future__ import annotations

import pytest

from core import metrics


def test_get_metrics_singleton():
    a = metrics.get_metrics()
    b = metrics.get_metrics()
    assert a is b


@pytest.mark.skipif(not metrics.HAVE_PROM, reason="prometheus_client not installed")
def test_render_returns_bytes(state_manager, engine):
    m = metrics.SimMetrics()
    m.observe_state(state_manager, engine)
    body = m.render()
    assert isinstance(body, bytes)
    assert b"volnix_sim_supply" in body
    assert b"volnix_sim_gini" in body


@pytest.mark.skipif(metrics.HAVE_PROM, reason="prometheus_client installed")
def test_noop_when_no_prom():
    m = metrics.SimMetrics()
    body = m.render()
    assert b"not installed" in body
