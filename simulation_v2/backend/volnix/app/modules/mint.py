"""WRT base reward (§5.1) and fee split (§5.4) — both by b_i (5.2-sim)."""

from __future__ import annotations

from volnix.app.state import LZN_HALVING_ERAS, AppState
from volnix.types.events import Event, ev


def era_at(height: int, halving_interval: int) -> int:
    if height <= 0 or halving_interval <= 0:
        return 0
    return (height - 1) // halving_interval


def block_reward(height: int, base: int, halving_interval: int) -> int:
    era = era_at(height, halving_interval)
    if era >= LZN_HALVING_ERAS:
        return 0
    return base >> era  # integer halving


def _split_by_b(total: int, earners: list) -> list[tuple[object, int]]:
    """⌊total · b_i / B⌋ with remainder +1 by ascending address (§4.1)."""
    if total <= 0 or not earners:
        return []
    b_sum = sum(rec.b_i for _, rec in earners)
    if b_sum <= 0:
        return []
    # floor shares
    shares = {acc.address: total * rec.b_i // b_sum for acc, rec in earners}
    distributed = sum(shares.values())
    rem = total - distributed
    if rem > 0:
        for addr in sorted(shares):
            if rem <= 0:
                break
            shares[addr] += 1
            rem -= 1
    return [(acc, shares[acc.address]) for acc, _ in earners]


def distribute_rewards(state: AppState, fee_pool: int) -> list[Event]:
    """Subsidy and fees by b_i among passed set. If B=0 / set not updated: no subsidy, burn F."""
    events: list[Event] = []
    reward = block_reward(state.height, state.params.base_block_reward, state.params.halving_interval)

    earners = []
    if state.set_updated:
        for addr in state.passed_validators:
            acc = state.accounts.get(addr)
            rec = state.declares.get(addr)
            if acc and rec and rec.b_i > 0 and rec.passed:
                earners.append((acc, rec))

    b_sum = sum(rec.b_i for _, rec in earners)

    if reward > 0 and b_sum > 0:
        for acc, share in _split_by_b(reward, earners):
            if share <= 0:
                continue
            acc.wrt += share
            state.wrt_supply += share
            events.append(
                ev("consensus.reward_distributed", validator=acc.address, amount=share, kind="base")
            )
    # else: no subsidy when B=0 / set not updated

    if fee_pool > 0:
        if b_sum > 0:
            for acc, share in _split_by_b(fee_pool, earners):
                if share <= 0:
                    continue
                acc.wrt += share
                events.append(
                    ev("consensus.fee_distributed", validator=acc.address, amount=share, kind="fee")
                )
            # fees already deducted from payers; still in wrt_supply (redistributed)
        else:
            # B=0: burn fees — reduce supply (already deducted from payers)
            state.wrt_supply -= fee_pool
            events.append(ev("consensus.fee_burned", amount=fee_pool, reason="B_zero"))

    return events
