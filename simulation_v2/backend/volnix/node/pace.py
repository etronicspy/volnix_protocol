"""Adaptive declare attempt window — canon §6.2 (5.5-sim).

Canonical T ∈ [1, 60] drives missed_budget / debt. Stand-only time_scale
compresses wall-clock sleep: wall_sleep = max(0.001, T / time_scale).
"""

from __future__ import annotations

from dataclasses import dataclass


BASE_BLOCK_TIME = 60
T_MIN = 1
TIME_SCALE_MIN = 1.0
TIME_SCALE_MAX = 3600.0
TIME_SCALE_DEFAULT = 60.0
WALL_SLEEP_MIN = 0.001


@dataclass
class AttemptPace:
    """Canonical attempt window + stand time_scale for wall-clock sleep."""

    base_block_time: int = BASE_BLOCK_TIME
    t_min: int = T_MIN
    attempt_window_sec: int = BASE_BLOCK_TIME
    missed_budget_sec: int = 0
    pace_debt_blocks: int = 0
    time_scale: float = TIME_SCALE_DEFAULT

    def set_time_scale(self, scale: float) -> float:
        self.time_scale = max(TIME_SCALE_MIN, min(TIME_SCALE_MAX, float(scale)))
        return self.time_scale

    def wall_sleep_sec(self) -> float:
        """Wall-clock seconds to wait for the current attempt window."""
        return max(WALL_SLEEP_MIN, float(self.attempt_window_sec) / float(self.time_scale))

    def on_empty_attempt(self) -> int:
        """Record a failed window; return the next attempt window T."""
        self.missed_budget_sec += self.attempt_window_sec
        self.attempt_window_sec = max(self.t_min, self.attempt_window_sec // 2)
        return self.attempt_window_sec

    def on_valid_block(self) -> int:
        """After a finalized height; return the next attempt window T."""
        if self.pace_debt_blocks <= 0 and self.missed_budget_sec > 0:
            self.pace_debt_blocks = self.missed_budget_sec // self.base_block_time
            self.missed_budget_sec = 0

        if self.pace_debt_blocks > 0:
            self.pace_debt_blocks -= 1
            if self.pace_debt_blocks > 0:
                self.attempt_window_sec = self.t_min
            else:
                self.attempt_window_sec = self.base_block_time
                self.missed_budget_sec = 0
            return self.attempt_window_sec

        self.attempt_window_sec = self.base_block_time
        self.missed_budget_sec = 0
        return self.attempt_window_sec

    def reset(self) -> None:
        """Force-reset attempt window to BaseBlockTime (tests / operator)."""
        self.attempt_window_sec = self.base_block_time
        self.missed_budget_sec = 0
        self.pace_debt_blocks = 0

    def snapshot(self) -> dict[str, int | float]:
        wall = self.wall_sleep_sec()
        return {
            "attempt_window_sec": int(self.attempt_window_sec),
            "pace_debt_blocks": int(self.pace_debt_blocks),
            "missed_budget_sec": int(self.missed_budget_sec),
            "base_block_time": int(self.base_block_time),
            "time_scale": float(self.time_scale),
            "wall_sleep_sec": float(wall),
            # Compat for traffic poll: effective wall-clock between attempts.
            "produce_interval_sec": float(wall),
        }
