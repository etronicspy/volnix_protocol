"""Supplier admission income gate — canon §5.6."""

from __future__ import annotations

from volnix.app.state import SUPPLIER_MIN_EPOCH_INCOME, AppState, EpochRecord


def supplier_income(price: int, ant_emit: int, n_suppliers: int) -> int:
    """income = price × (ANT_emit // N) in micro-WRT."""
    if n_suppliers <= 0:
        return 0
    return price * (ant_emit // n_suppliers)


def new_supplier_allowed(
    *,
    last_epoch: EpochRecord | None,
    last_ant_wrt_price: int | None,
    floor: int = SUPPLIER_MIN_EPOCH_INCOME,
) -> tuple[bool, str]:
    """Return (allowed, reason). reason empty when allowed or gate not applicable."""
    if last_epoch is None:
        return True, ""
    n = len(last_epoch.suppliers)
    ant_emit = last_epoch.ant_emit
    if n <= 0 or ant_emit <= 0:
        return True, ""
    if last_ant_wrt_price is None:
        return True, ""
    income = supplier_income(last_ant_wrt_price, ant_emit, n)
    if income <= floor:
        return False, "supplier income below floor"
    return True, ""


def check_new_supplier(state: AppState) -> None:
    """Raise ValueError-compatible message if income gate blocks admission."""
    last = state.epochs[-1] if state.epochs else None
    ok, reason = new_supplier_allowed(
        last_epoch=last,
        last_ant_wrt_price=state.last_ant_wrt_price,
    )
    if not ok:
        raise ValueError(reason)
