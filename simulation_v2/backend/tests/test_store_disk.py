"""Disk-backed JSONL stores: seek by offset index, not full RAM maps."""

from __future__ import annotations

from pathlib import Path

from volnix.store.blockstore import BlockStore
from volnix.store.jsonl import JsonlStore
from volnix.store.resultstore import ResultStore
from volnix.store.txindex import TxIndex, TxLocation
from volnix.types.block import Block, BlockResults, Header, BlockID, PartSetHeader


def _header(height: int, app_hash: str = "aa") -> Header:
    return Header(
        version={"block": "11", "app": "1"},
        chain_id="volnix-sim-2",
        height=height,
        time="2026-01-01T00:00:00Z",
        last_block_id=BlockID(hash="", parts=PartSetHeader()),
        last_commit_hash="",
        data_hash="",
        validators_hash="",
        next_validators_hash="",
        consensus_hash="",
        app_hash=app_hash,
        last_results_hash="",
        evidence_hash="",
        proposer_address="proposer",
    )


def test_jsonl_idx_roundtrip(tmp_path: Path):
    store = JsonlStore(tmp_path / "t.jsonl")
    store.append({"n": 0})
    store.append({"n": 1})
    store.append({"n": 2})
    store.flush()
    assert store.record_count() == 3
    assert store.read_record(1) == {"n": 1}
    idx = store.idx_path
    assert idx.exists()
    idx.unlink()
    store2 = JsonlStore(tmp_path / "t.jsonl")
    assert store2.record_count() == 3
    assert store2.read_record(2) == {"n": 2}


def test_blockstore_get_after_drop_cache(tmp_path: Path):
    bs = BlockStore(tmp_path)
    for h in range(5):
        b = Block(header=_header(h, app_hash=f"{h:02x}"), data={"txs": []})
        bs.append(b)
    bs.flush()
    tip_hash = bs.get(4).hash()
    bs.drop_cache()
    assert bs.get(2) is not None
    assert bs.get(2).header.height == 2
    assert bs.get_by_hash(tip_hash).header.height == 4
    assert [b.header.height for b in bs.range(1, 3)] == [1, 2, 3]


def test_blockstore_hash_idx_rebuild(tmp_path: Path):
    bs = BlockStore(tmp_path)
    for h in range(3):
        bs.append(Block(header=_header(h), data={"txs": []}))
    bs.flush()
    h2 = bs.get(2).hash()
    bs._hash_path.unlink()
    bs2 = BlockStore(tmp_path)
    bs2.load()
    assert bs2.get_by_hash(h2).header.height == 2


def test_resultstore_disk_get(tmp_path: Path):
    rs = ResultStore(tmp_path)
    for h in range(4):
        rs.append(BlockResults(height=h))
    rs.flush()
    rs.drop_cache()
    assert rs.get(1).height == 1
    assert rs.get(3).height == 3


def test_txindex_newest_hashes(tmp_path: Path):
    idx = TxIndex(tmp_path)
    for i in range(5):
        idx.append(TxLocation(tx_hash=f"h{i}", height=i, index=0, sender="a"))
    assert idx.newest_hashes(2) == ["h4", "h3"]
    assert idx.newest_hashes(2, offset=2) == ["h2", "h1"]
    assert idx.all_hashes_newest_first()[0] == "h4"
    assert len(idx) == 5
