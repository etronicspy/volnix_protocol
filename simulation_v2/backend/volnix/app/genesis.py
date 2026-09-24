from __future__ import annotations

import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from volnix.app.state import SCALE, Account, AppState, Params
from volnix.crypto.hashing import hash_obj
from volnix.crypto.keys import derive_keypair
from volnix.types.block import Block, BlockResults, Commit, Header, empty_block_id, evidence_hash
from volnix.types.role import Role
from volnix.types.validator import Validator, ValidatorSet, voting_power


def load_genesis_doc(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def init_state(doc: dict[str, Any]) -> tuple[AppState, ValidatorSet, Block]:
    app = doc.get("app_state", {})
    params = Params.from_dict(app.get("params", {}))
    gv = app.get("genesis_validator", {})
    kp = derive_keypair(gv.get("seed", "volnix-genesis-validator-v2"))

    state = AppState()
    state.params = params
    state.chain_id = doc.get("chain_id", "volnix-sim-2")
    state.genesis_time = doc.get("genesis_time", "2026-01-01T00:00:00Z")
    state.height = 0
    state.genesis_validator = kp.address

    lzn_act = int(gv.get("lzn_activated", 1)) * SCALE
    ant = int(gv.get("ant", 10_080)) * SCALE
    wrt = int(gv.get("wrt", 0))
    acc = Account(
        address=kp.address,
        pub_hex=kp.pub_hex,
        role=Role.VALIDATOR,
        wrt=wrt,
        lzn=0,
        lzn_activated=lzn_act,
        ant=ant,
        last_tx_height=0,
        genesis_no_zkp=True,
        created_height=0,
        lzn_freeze_until=params.lzn_freeze_period,
    )
    state.accounts[kp.address] = acc
    state.wrt_supply = wrt
    state.lzn_minted_tokens = 1
    state.lzn_pool_remaining = params.lzn_total_supply - 1

    # Bootstrap weight: full stake so the single genesis validator can propose.
    w_i = 1.0
    vset = ValidatorSet(
        validators=[
            Validator(
                address=kp.address,
                pub_hex=kp.pub_hex,
                l_i=lzn_act,
                s_i=lzn_act,
                w_i=w_i,
                power=voting_power(lzn_act, lzn_act),
                proposer_priority=0,
            )
        ],
        proposer=kp.address,
    )

    consensus_hash = hash_obj(doc.get("consensus_params", {}))
    header = Header(
        version={"block": "11", "app": "1"},
        chain_id=state.chain_id,
        height=0,
        time=state.genesis_time,
        last_block_id=empty_block_id(),
        last_commit_hash=hash_obj({"height": 0, "round": 0, "signatures": []}),
        data_hash=hash_obj({"txs": []}),
        validators_hash=vset.hash(),
        next_validators_hash=vset.hash(),
        consensus_hash=consensus_hash,
        app_hash=state.app_hash(),
        last_results_hash=hash_obj([]),
        evidence_hash=evidence_hash([]),
        proposer_address=kp.address,
    )
    block = Block(
        header=header,
        data={"txs": []},
        evidence=[],
        last_commit=Commit(height=0, round=0, block_id=empty_block_id(), signatures=[]),
    )
    return state, vset, block


def genesis_results() -> BlockResults:
    return BlockResults(height=0)


def parse_time(iso: str) -> datetime:
    if iso.endswith("Z"):
        iso = iso[:-1] + "+00:00"
    return datetime.fromisoformat(iso).astimezone(timezone.utc)


def format_time(dt: datetime) -> str:
    return dt.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
