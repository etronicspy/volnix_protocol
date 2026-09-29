"""MOA strip must eject validator from current/next ValidatorSet (§5.3)."""

from __future__ import annotations

from volnix.app.abci import BaseApp
from volnix.app.state import SCALE, Account
from volnix.types.role import Role
from volnix.types.validator import Validator, ValidatorSet, voting_power


def _val(addr: str, l_i: int = SCALE, s_i: int = SCALE // 2) -> Validator:
    return Validator(
        address=addr,
        pub_hex="ab" * 32,
        l_i=l_i,
        s_i=s_i,
        w_i=s_i / l_i,
        power=voting_power(s_i, l_i),
    )


def test_moa_removes_idle_validator_from_sets():
    app = BaseApp()
    app.state.height = 0
    app.state.params.moa_validator_window = 10
    keep = "volnix1keep"
    idle = "volnix1idle"
    for addr, last in ((keep, 100), (idle, 0)):
        app.state.accounts[addr] = Account(
            address=addr,
            role=Role.VALIDATOR,
            lzn_activated=SCALE,
            last_tx_height=last,
            created_height=last,
        )
    vset = ValidatorSet(
        validators=[_val(keep), _val(idle)],
        proposer=keep,
    )
    app.validator_set = vset.copy()
    app.next_validator_set = vset.copy()

    app.begin_block(height=20)

    assert app.state.accounts[idle].role == Role.CITIZEN
    assert app.state.accounts[keep].role == Role.VALIDATOR
    assert idle not in app.validator_set.by_address()
    assert idle not in app.next_validator_set.by_address()
    assert keep in app.validator_set.by_address()


def test_moa_keeps_sole_validator_in_set_fallback():
    """Stand safety: do not empty the only proposer slot."""
    app = BaseApp()
    app.state.params.moa_validator_window = 5
    sole = "volnix1sole"
    app.state.accounts[sole] = Account(
        address=sole,
        role=Role.VALIDATOR,
        lzn_activated=SCALE,
        last_tx_height=0,
        created_height=0,
    )
    app.validator_set = ValidatorSet(validators=[_val(sole)], proposer=sole)
    app.next_validator_set = app.validator_set.copy()

    app.begin_block(height=100)

    assert app.state.accounts[sole].role == Role.CITIZEN
    assert sole in app.validator_set.by_address()
