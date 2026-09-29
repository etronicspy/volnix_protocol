"""DAO must reject removed / non-governable params (dead DAO п.4)."""

from __future__ import annotations

import pytest

from volnix.app.modules.gov import GovError, deliver_submit
from volnix.app.state import Account, AppState
from volnix.types.msgs import MsgSubmitProposal
from volnix.types.role import Role


def test_ant_supplier_epoch_limit_not_governable():
    st = AppState()
    st.accounts["p"] = Account(address="p", role=Role.VALIDATOR, wrt=2_000_000)
    with pytest.raises(GovError, match="not DAO-governable"):
        deliver_submit(
            st,
            MsgSubmitProposal(
                proposer="p",
                title="dead",
                description="removed DAO п.4",
                parameter_changes={"ant_supplier_epoch_limit": 1},
                deposit=1_000_000,
            ),
        )
