"""Unit tests for adaptive attempt window (canon §6.2 / 5.5-sim)."""

import pytest

from volnix.node.pace import AttemptPace, BASE_BLOCK_TIME, T_MIN


def test_empty_halves_down_to_one():
    p = AttemptPace()
    assert p.attempt_window_sec == 60
    expected = [30, 15, 7, 3, 1, 1]
    for want in expected:
        assert p.on_empty_attempt() == want
        assert p.attempt_window_sec == want
    assert p.missed_budget_sec == 60 + 30 + 15 + 7 + 3 + 1


def test_debt_from_missed_budget_catchup_then_reset():
    p = AttemptPace()
    p.missed_budget_sec = 150  # 2 target minutes
    p.attempt_window_sec = 1
    # First valid: debt=2, then decrement → 1, T=1
    assert p.on_valid_block() == T_MIN
    assert p.pace_debt_blocks == 1
    assert p.missed_budget_sec == 0
    # Second valid: debt→0, T=60
    assert p.on_valid_block() == BASE_BLOCK_TIME
    assert p.pace_debt_blocks == 0
    assert p.attempt_window_sec == BASE_BLOCK_TIME


def test_valid_without_missed_stays_at_base():
    p = AttemptPace()
    assert p.on_valid_block() == BASE_BLOCK_TIME
    assert p.pace_debt_blocks == 0
    assert p.missed_budget_sec == 0


def test_missed_under_one_minute_no_debt():
    p = AttemptPace()
    p.on_empty_attempt()  # missed=60, T=30
    p.on_empty_attempt()  # missed=90, T=15 — still debt only applied on valid
    # Force missed below 2 minutes but >= 1: after first empty alone debt=1
    p2 = AttemptPace()
    p2.on_empty_attempt()  # missed=60
    assert p2.on_valid_block() == BASE_BLOCK_TIME  # debt=1 then -=1 → 0 → reset
    assert p2.pace_debt_blocks == 0


def test_reset():
    p = AttemptPace()
    p.on_empty_attempt()
    p.on_empty_attempt()
    p.reset()
    assert p.attempt_window_sec == BASE_BLOCK_TIME
    assert p.missed_budget_sec == 0
    assert p.pace_debt_blocks == 0


def test_snapshot_compat_produce_interval():
    p = AttemptPace()
    p.on_empty_attempt()
    snap = p.snapshot()
    assert snap["attempt_window_sec"] == 30
    assert snap["time_scale"] == 60.0
    assert snap["wall_sleep_sec"] == 0.5
    assert snap["produce_interval_sec"] == 0.5
    assert snap["base_block_time"] == 60


def test_wall_sleep_with_time_scale():
    p = AttemptPace()
    assert p.attempt_window_sec == 60
    assert p.time_scale == 60.0
    assert p.wall_sleep_sec() == 1.0
    p.set_time_scale(60)
    p.attempt_window_sec = 30
    assert p.wall_sleep_sec() == 0.5
    p.set_time_scale(1)
    assert p.wall_sleep_sec() == 30.0
    p.set_time_scale(3600)
    p.attempt_window_sec = 60
    assert p.wall_sleep_sec() == pytest.approx(60 / 3600)
    p.set_time_scale(0.5)  # clamped to 1
    assert p.time_scale == 1.0
