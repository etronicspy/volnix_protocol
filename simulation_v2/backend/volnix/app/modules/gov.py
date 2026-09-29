"""Limited DAO — canon §7.2."""

from __future__ import annotations

from fractions import Fraction

from volnix.app.modules.epoch import lzn_horizon
from volnix.app.state import ALPHA_MAX, ALPHA_MIN, LAMBDA_MAX, PARAM_BOUNDS, AppState, Proposal
from volnix.types.events import Event, ev
from volnix.types.msgs import MsgSubmitProposal, MsgVote


class GovError(ValueError):
    pass


ALLOWED_PARAMS = {
    "lzn_freeze_period",
    "moa_supplier_window",
    "moa_validator_window",
    "block_gas_limit",
    "max_block_bytes",
    "max_active_suppliers",
    "max_active_validators",
    "epoch_blocks",
    "halving_interval",
    "lambda_num",
    "lambda_den",
    "alpha_num",
    "alpha_den",
}


def deliver_submit(state: AppState, msg: MsgSubmitProposal) -> list[Event]:
    acc = state.accounts.get(msg.proposer)
    if acc is None:
        raise GovError("proposer not found")
    if msg.deposit < state.params.gov_min_deposit:
        raise GovError("deposit below minimum")
    if acc.wrt < msg.deposit:
        raise GovError("insufficient WRT for deposit")
    for key in msg.parameter_changes:
        if key not in ALLOWED_PARAMS:
            raise GovError(f"parameter {key} is not DAO-governable")
    acc.wrt -= msg.deposit
    pid = state.next_proposal_id
    state.next_proposal_id += 1
    prop = Proposal(
        proposal_id=pid,
        proposer=msg.proposer,
        title=msg.title,
        description=msg.description,
        parameter_changes=dict(msg.parameter_changes),
        deposit=msg.deposit,
        submit_height=state.height,
        voting_end=state.height + state.params.gov_voting_period,
        execute_height=state.height + state.params.gov_voting_period + state.params.gov_timelock,
    )
    state.proposals[pid] = prop
    return [ev("governance.proposal_submitted", proposal_id=pid, proposer=msg.proposer)]


def deliver_vote(state: AppState, msg: MsgVote) -> list[Event]:
    prop = state.proposals.get(msg.proposal_id)
    if prop is None:
        raise GovError("proposal not found")
    if prop.status != "voting":
        raise GovError("proposal not in voting")
    if state.height > prop.voting_end:
        raise GovError("voting period ended")
    if msg.option not in ("yes", "no", "abstain"):
        raise GovError("invalid option")
    acc = state.accounts.get(msg.voter)
    if acc is None or acc.wrt <= 0:
        raise GovError("voter has no WRT")
    prop.votes[msg.voter] = msg.option
    return [ev("governance.vote_cast", proposal_id=msg.proposal_id, voter=msg.voter, option=msg.option)]


def tally_and_execute(state: AppState) -> list[Event]:
    events: list[Event] = []
    total_wrt = sum(a.wrt for a in state.accounts.values()) + sum(p.deposit for p in state.proposals.values())
    for prop in state.proposals.values():
        if prop.status == "voting" and state.height > prop.voting_end:
            yes = no = voted = 0
            for addr, opt in prop.votes.items():
                w = state.accounts[addr].wrt if addr in state.accounts else 0
                voted += w
                if opt == "yes":
                    yes += w
                elif opt == "no":
                    no += w
            quorum = Fraction(state.params.gov_quorum_num, state.params.gov_quorum_den)
            thresh = Fraction(state.params.gov_threshold_num, state.params.gov_threshold_den)
            if total_wrt > 0 and Fraction(voted, total_wrt) >= quorum and yes + no > 0 and Fraction(yes, yes + no) >= thresh:
                prop.status = "passed"
            else:
                prop.status = "rejected"
                _refund(state, prop)
                events.append(ev("governance.proposal_rejected", proposal_id=prop.proposal_id))
        if prop.status == "passed" and state.height >= prop.execute_height:
            events.extend(_apply(state, prop))
            prop.status = "executed"
            _refund(state, prop)
            events.append(ev("governance.proposal_executed", proposal_id=prop.proposal_id))
    return events


def _refund(state: AppState, prop: Proposal) -> None:
    acc = state.accounts.get(prop.proposer)
    if acc and prop.deposit > 0:
        acc.wrt += prop.deposit
        prop.deposit = 0


def _apply(state: AppState, prop: Proposal) -> list[Event]:
    p = state.params
    changes = dict(prop.parameter_changes)
    period_keys = ("epoch_blocks", "halving_interval")
    period_changed = any(k in changes for k in period_keys)
    # apply numeric fields with bounds
    for key, val in list(changes.items()):
        if key in ("lambda_num", "lambda_den", "alpha_num", "alpha_den"):
            continue
        if key not in PARAM_BOUNDS:
            continue
        lo, hi = PARAM_BOUNDS[key]
        iv = int(val)
        if iv < lo or iv > hi:
            raise GovError(f"{key} out of bounds")
        setattr(p, key, iv)
    if "lambda_num" in changes or "lambda_den" in changes:
        n = int(changes.get("lambda_num", p.lambda_num))
        d = int(changes.get("lambda_den", p.lambda_den))
        lam = Fraction(n, d)
        if lam <= 0 or lam > LAMBDA_MAX:
            raise GovError("lambda out of bounds")
        p.lambda_num, p.lambda_den = n, d
    if "alpha_num" in changes or "alpha_den" in changes:
        n = int(changes.get("alpha_num", p.alpha_num))
        d = int(changes.get("alpha_den", p.alpha_den))
        a = Fraction(n, d)
        if a < ALPHA_MIN or a > ALPHA_MAX:
            raise GovError("alpha out of bounds")
        p.alpha_num, p.alpha_den = n, d
    events: list[Event] = []
    if period_changed:
        # §5.5: leftover LZN pool R is not burned; N_rem follows new H_end.
        h_end, n_rem = lzn_horizon(state)
        events.append(
            ev(
                "anteil.lzn_horizon_recomputed",
                h_end=h_end,
                n_rem=n_rem,
                pool_remaining=state.lzn_pool_remaining,
                height=state.height,
            )
        )
    return events
