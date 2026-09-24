"""CometBFT-style proposer priority."""

from __future__ import annotations

from volnix.types.validator import Validator, ValidatorSet


def increment_proposer_priority(vset: ValidatorSet) -> Validator:
    if not vset.validators:
        raise ValueError("empty validator set")
    total = vset.total_power()
    for v in vset.validators:
        v.proposer_priority += v.power
    proposer = max(vset.validators, key=lambda v: (v.proposer_priority, _neg_addr(v.address)))
    proposer.proposer_priority -= total
    vset.proposer = proposer.address
    return proposer


def _neg_addr(address: str) -> str:
    # smaller address wins ties → invert by using address as secondary after priority
    return address
