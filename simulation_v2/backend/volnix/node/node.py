"""Single-process blockchain node: stores, ABCI app, mempool, consensus loop."""

from __future__ import annotations

import asyncio
from datetime import timedelta
from pathlib import Path
from typing import Any, Awaitable, Callable

from volnix.app.abci import BaseApp
from volnix.app.genesis import format_time, parse_time
from volnix.consensus.evidence import EvidencePool
from volnix.consensus.rounds import ConsensusEngine, Decision
from volnix.consensus.validator_set import increment_proposer_priority
from volnix.crypto.hashing import hash_obj
from volnix.node.faults import FaultController
from volnix.node.mempool import Mempool
from volnix.node.pace import AttemptPace
from volnix.store.appsnapshot import AppSnapshotStore
from volnix.store.blockstore import BlockStore
from volnix.store.resultstore import ResultStore
from volnix.store.txindex import TxIndex, TxLocation
from volnix.store.valsetstore import ValidatorSetStore
from volnix.types.block import (
    Block,
    BlockID,
    BlockResults,
    Header,
    PartSetHeader,
    empty_block_id,
    evidence_hash,
    results_hash,
    txs_data_hash,
)
from volnix.types.tx import Tx
from volnix.types.validator import ValidatorSet


class Node:
    def __init__(
        self,
        data_dir: Path,
        genesis_path: Path,
        auto_declare: bool = True,
        time_scale: float = 60.0,
    ) -> None:
        self.data_dir = data_dir
        self.genesis_path = genesis_path
        self.auto_declare = auto_declare
        self.pace = AttemptPace()
        self.pace.set_time_scale(time_scale)
        self.app = BaseApp()
        self.blocks = BlockStore(data_dir)
        self.results = ResultStore(data_dir)
        self.tx_index = TxIndex(data_dir)
        self.valsets = ValidatorSetStore(data_dir)
        self.snapshot = AppSnapshotStore(data_dir)
        self.mempool = Mempool(self.app)
        self.consensus = ConsensusEngine()
        self.evidence = EvidencePool()
        self.faults = FaultController()
        self.consensus.faults = self.faults.model
        self._listeners: list[Callable[[dict[str, Any]], Awaitable[None]]] = []
        self._task: asyncio.Task[None] | None = None
        self._running = False
        self._last_commit = None
        self.consensus_hash = ""
        self._interval_wake = asyncio.Event()

    @property
    def produce_interval(self) -> float:
        """Compat alias: wall-clock sleep for current attempt (stand time_scale)."""
        return self.pace.wall_sleep_sec()

    def enqueue_auto_declares(self) -> None:
        """Stand: inject genesis MsgDeclareParticipation so PoVB burns ANT (§5.4)."""
        if not self.auto_declare:
            return
        from volnix.node.autodeclare import build_genesis_declare_tx

        tx = build_genesis_declare_tx(self)
        if tx is not None:
            self.mempool.insert(tx)

    def _mempool_has_declare(self) -> bool:
        for tx in self.mempool.pending():
            if any(m.type == "povb/MsgDeclareParticipation" for m in tx.body.messages):
                return True
        return False

    def _participation_ready(self) -> bool:
        """Commit only with new MsgDeclareParticipation in mempool (§5.4 / 5.5-sim)."""
        return self._mempool_has_declare()

    def subscribe(self, cb: Callable[[dict[str, Any]], Awaitable[None]]) -> None:
        self._listeners.append(cb)

    async def _emit(self, event: dict[str, Any]) -> None:
        for cb in list(self._listeners):
            try:
                await cb(event)
            except Exception:
                pass

    def load_or_init(self) -> None:
        self.data_dir.mkdir(parents=True, exist_ok=True)
        self.app.init_chain(self.genesis_path)
        self.consensus_hash = hash_obj(self.app.consensus_params)
        genesis_block = self.app.genesis_block
        assert genesis_block is not None

        self.blocks.load()
        self.results.load()
        self.tx_index.load()
        self.valsets.load()

        if self.blocks.get(0) is None:
            self.blocks.append(genesis_block)
            self.results.append(BlockResults(height=0))
            self.valsets.append(0, self.app.validator_set)
            self.snapshot.save(self.app.state.to_snapshot())
            self._last_commit = genesis_block.last_commit
            return

        # Replay blocks 1..N from JSONL (source of truth).
        heights = [h for h in self.blocks.heights() if h >= 1]
        for h in heights:
            block = self.blocks.get(h)
            if block is None:
                continue
            evidence = list(block.evidence)
            self.app.begin_block(h, evidence)
            for tx in block.txs():
                self.app.deliver_tx(tx)
            next_set, _, _ = self.app.end_block()
            app_hash = self.app.commit()
            if app_hash != block.header.app_hash:
                raise RuntimeError(
                    f"replay app_hash mismatch at height {h}: {app_hash} != {block.header.app_hash}"
                )
            self.app.validator_set = next_set
        latest = self.blocks.get(self.blocks.latest_height())
        if latest is not None:
            self._last_commit = latest.last_commit
        # persist reconstructed snapshot
        self.snapshot.save(self.app.state.to_snapshot())

    def broadcast_tx(self, tx: Tx) -> dict[str, Any]:
        result = self.mempool.insert(tx)
        return {"code": result.code, "log": result.log, "hash": tx.tx_hash()}

    def produce_block(self) -> Block | None:
        prev = self.blocks.get(self.app.state.height if self.app.state.height > 0 else 0)
        if prev is None:
            prev = self.blocks.get(0)
        assert prev is not None
        height = prev.header.height + 1
        prev_time = parse_time(prev.header.time)
        block_time = format_time(prev_time + timedelta(seconds=self.app.state.params.base_block_time))

        snap = self.app.state.snapshot()
        vset_snap = self.app.validator_set.copy()

        self.enqueue_auto_declares()
        if not self._participation_ready():
            return None
        proposer = increment_proposer_priority(self.app.validator_set)
        evidence_items = self.evidence.drain()
        txs = self.mempool.reap(self.app.state.params.block_gas_limit, self.app.state.params.max_block_bytes)

        self.app.begin_block(height, evidence_items)
        tx_results = [self.app.deliver_tx(tx) for tx in txs]
        next_set, _, _ = self.app.end_block()
        app_hash = self.app.commit()
        results = self.app.collect_results()

        tx_dicts = [tx.to_dict() for tx in txs]
        tx_hashes = [tx.tx_hash() for tx in txs]
        last_commit = self._last_commit
        last_commit_hash = last_commit.hash() if last_commit else hash_obj({})
        last_block_id = BlockID(hash=prev.hash(), parts=PartSetHeader(total=1, hash=prev.hash()))

        header = Header(
            version={"block": "11", "app": "1"},
            chain_id=self.app.state.chain_id,
            height=height,
            time=block_time,
            last_block_id=last_block_id,
            last_commit_hash=last_commit_hash,
            data_hash=txs_data_hash(tx_hashes),
            validators_hash=self.app.validator_set.hash(),
            next_validators_hash=next_set.hash(),
            consensus_hash=self.consensus_hash,
            app_hash=app_hash,
            last_results_hash=results_hash(tx_results),
            evidence_hash=evidence_hash(evidence_items),
            proposer_address=proposer.address,
        )
        # tentative hash for voting
        block_hash = header.hash()
        self.consensus.faults = self.faults.model
        decision = self.consensus.run_rounds(height, block_time, block_hash, self.app.validator_set)
        if decision.decision != Decision.COMMIT:
            self.app.state.restore(snap)
            self.app.validator_set = vset_snap
            # return txs to mempool
            for tx in txs:
                self.mempool.insert(tx)
            return None

        block = Block(
            header=header,
            data={"txs": tx_dicts},
            evidence=evidence_items,
            last_commit=last_commit if last_commit is not None else None,
        )
        # last_commit of THIS block is the previous commit; next block will use current decision
        self._last_commit = decision.commit
        self.blocks.append(block)
        self.results.append(results)
        self.valsets.append(height, next_set)
        for i, tx in enumerate(txs):
            self.tx_index.append(
                TxLocation(tx_hash=tx.tx_hash(), height=height, index=i, sender=tx.signer())
            )
        self.app.validator_set = next_set
        self.snapshot.save(self.app.state.to_snapshot())
        self.pace.on_valid_block()
        return block

    async def produce_and_notify(self) -> Block | None:
        block = self.produce_block()
        if block is None:
            return None
        await self._emit(
            {
                "type": "new_block",
                "height": block.header.height,
                "hash": block.hash(),
                "time": block.header.time,
                "proposer": block.header.proposer_address,
                "n_txs": len(block.data.get("txs", [])),
                "app_hash": block.header.app_hash,
            }
        )
        if self.app.state.set_updated:
            await self._emit(
                {
                    "type": "validator_set_update",
                    "height": block.header.height,
                    "validators": [v.to_dict() for v in self.app.validator_set.validators],
                }
            )
        if self.app.state.epochs and self.app.state.epochs[-1].height == block.header.height:
            rec = self.app.state.epochs[-1]
            await self._emit({"type": "epoch_boundary", **rec.to_dict()})
        for txd in block.data.get("txs", []):
            await self._emit({"type": "new_tx", "hash": txd.get("hash", ""), "height": block.header.height})
        return block

    def pace_snapshot(self) -> dict[str, Any]:
        snap = self.pace.snapshot()
        snap["auto_produce"] = bool(self._running)
        return snap

    def set_time_scale(self, scale: float) -> dict[str, Any]:
        """Stand-only wall-clock acceleration (does not change canonical T)."""
        self.pace.set_time_scale(scale)
        self._interval_wake.set()
        return self.pace_snapshot()

    def reset_pace(self) -> dict[str, Any]:
        """Force-reset attempt window to BaseBlockTime (tests / operator)."""
        self.pace.reset()
        self._interval_wake.set()
        return self.pace_snapshot()

    async def loop(self) -> None:
        """Canon §6.2: sleep wall_sleep(T/scale), then commit or empty attempt."""
        self._running = True
        while self._running:
            self._interval_wake.clear()
            try:
                await asyncio.wait_for(
                    self._interval_wake.wait(),
                    timeout=self.pace.wall_sleep_sec(),
                )
            except asyncio.TimeoutError:
                pass
            if not self._running:
                break
            block = await self.produce_and_notify()
            if block is None:
                self.pace.on_empty_attempt()

    def start(self) -> None:
        if self._task is None:
            self._task = asyncio.create_task(self.loop())

    async def stop(self) -> None:
        self._running = False
        self._interval_wake.set()
        if self._task:
            self._task.cancel()
            try:
                await self._task
            except (asyncio.CancelledError, Exception):
                pass
            self._task = None

    def status(self) -> dict[str, Any]:
        h = self.blocks.latest_height()
        latest = self.blocks.get(h)
        return {
            "node_info": {"id": "volnix-sim-2", "network": self.app.state.chain_id, "moniker": "sim2"},
            "sync_info": {
                "latest_block_hash": latest.hash() if latest else "",
                "latest_app_hash": latest.header.app_hash if latest else "",
                "latest_block_height": str(h),
                "latest_block_time": latest.header.time if latest else "",
                "catching_up": False,
            },
            "validator_info": {
                "address": self.app.validator_set.proposer,
                "voting_power": str(self.app.validator_set.total_power()),
            },
        }

    def replay_verify(self, genesis_doc: dict | None = None) -> str:
        """Rebuild state from genesis + blocks.jsonl and return final app_hash."""
        fresh = BaseApp()
        if genesis_doc is not None:
            from volnix.app.genesis import init_state

            fresh.state, fresh.validator_set, _ = init_state(genesis_doc)
            fresh.next_validator_set = fresh.validator_set.copy()
            fresh.consensus_params = genesis_doc.get("consensus_params", {})
        else:
            fresh.init_chain(self.genesis_path)
        for h in [x for x in self.blocks.heights() if x >= 1]:
            block = self.blocks.get(h)
            assert block is not None
            fresh.begin_block(h, list(block.evidence))
            for tx in block.txs():
                fresh.deliver_tx(tx)
            next_set, _, _ = fresh.end_block()
            app_hash = fresh.commit()
            if app_hash != block.header.app_hash:
                raise RuntimeError(f"verify mismatch at {h}")
            fresh.validator_set = next_set
        return fresh.state.app_hash()
