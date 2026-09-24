"""Epoch boundary — canon §5.5."""

from __future__ import annotations

import math

from volnix.app.modules.anteil import cancel_ant_orders
from volnix.app.state import LZN_HALVING_ERAS, SCALE, AppState, EpochRecord
from volnix.types.events import Event, ev


def is_boundary(height: int, epoch_blocks: int) -> bool:
    return height > 0 and epoch_blocks > 0 and height % epoch_blocks == 0


def remaining_boundaries(height: int, epoch_blocks: int, h_end: int) -> int:
    """Number of remaining epoch boundaries with height <= H_end, including current."""
    if epoch_blocks <= 0 or height > h_end:
        return 0
    # boundaries at k*epoch_blocks for k >= 1 and k*epoch_blocks <= h_end
    last_k = h_end // epoch_blocks
    current_k = height // epoch_blocks
    if current_k < 1:
        return last_k
    return max(0, last_k - current_k + 1)


def lzn_epoch_amount(remaining_pool: int, n_rem: int) -> int:
    if n_rem <= 0 or remaining_pool <= 0:
        return 0
    return math.ceil(remaining_pool / n_rem)


def split_even(total: int, addresses: list[str]) -> dict[str, int]:
    if not addresses or total <= 0:
        return {a: 0 for a in addresses}
    n = len(addresses)
    q, r = divmod(total, n)
    addrs = sorted(addresses)
    return {a: q + (1 if i < r else 0) for i, a in enumerate(addrs)}


def process_boundary(state: AppState) -> list[Event]:
    events: list[Event] = []
    events.extend(cancel_ant_orders(state))

    wiped = 0
    for acc in state.accounts.values():
        wiped += acc.ant
        acc.ant = 0
    events.append(ev("anteil.epoch_reset", epoch=state.epoch + 1, wiped=wiped, height=state.height))

    l_total = state.l_total()
    suppliers = state.active_suppliers()
    supplier_addrs = [a.address for a in suppliers]

    ant_emit = 0
    if suppliers:
        ant_emit = l_total * state.params.epoch_blocks
        shares = split_even(ant_emit, supplier_addrs)
        for addr, amt in shares.items():
            state.accounts[addr].ant += amt
        events.append(
            ev(
                "anteil.epoch_emission",
                denom="uant",
                total_emitted=ant_emit,
                suppliers=len(suppliers),
                height=state.height,
            )
        )

    h_end = LZN_HALVING_ERAS * state.params.halving_interval
    lzn_emit_tokens = 0
    if state.height <= h_end and state.lzn_pool_remaining > 0:
        n_rem = remaining_boundaries(state.height, state.params.epoch_blocks, h_end)
        quota = lzn_epoch_amount(state.lzn_pool_remaining, n_rem)
        if suppliers and quota > 0:
            lzn_emit_tokens = quota
            shares_t = split_even(quota, supplier_addrs)
            for addr, tokens in shares_t.items():
                state.accounts[addr].lzn += tokens * SCALE
            state.lzn_pool_remaining -= quota
            state.lzn_minted_tokens += quota
            events.append(
                ev(
                    "anteil.epoch_emission",
                    denom="ulzn",
                    total_emitted=quota,
                    suppliers=len(suppliers),
                    height=state.height,
                )
            )

    state.epoch += 1
    rec = EpochRecord(
        epoch=state.epoch,
        height=state.height,
        l_total=l_total,
        ant_wiped=wiped,
        ant_emit=ant_emit,
        lzn_emit=lzn_emit_tokens,
        suppliers=supplier_addrs,
    )
    state.epochs.append(rec)
    return events
