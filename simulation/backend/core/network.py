"""NetworkSim — логическая многоузловость в одном процессе.

Каждый адрес детерминированно привязан к одному узлу (`hash(addr) % num_nodes`),
genesis-адреса всегда на `node_0`. Tx подаётся в local mempool узла-источника,
затем gossip-mock планирует доставку на остальные узлы с латентностью и
вероятностью потери (`gossip_loss_pct`). Engine забирает только те tx, что
дошли до кворума узлов (`>= quorum_pct`, по умолчанию 2/3).

Если `num_nodes == 1` — режим fastpath: сразу в общий мемпул, без gossip.

Канон-нейтральный модуль: сами правила admission остаются в
`wallet_validate.py` / `engine.py`, NetworkSim лишь перераспределяет tx по узлам.
"""
from __future__ import annotations

import asyncio
import random
import threading
import time
from collections import defaultdict, deque
from dataclasses import dataclass, field
from typing import Iterable, List, Optional

from core.models import Transaction

GENESIS_NODE_ID = "node_0"


def _stable_node_for_addr(addr: str, num_nodes: int) -> str:
    """Детерминированный hash → node_id (без python-process-salt)."""
    if not addr:
        return GENESIS_NODE_ID
    # Стабильный хэш (md5 hex int), чтобы при рестарте назначение не плыло.
    import hashlib

    digest = hashlib.md5(addr.encode("utf-8")).hexdigest()
    idx = int(digest, 16) % max(1, int(num_nodes))
    return f"node_{idx}"


@dataclass
class Node:
    id: str
    mempool: deque = field(default_factory=deque)
    addresses: set = field(default_factory=set)
    # tx_hash → first arrival ts (для метрик задержки gossip)
    arrived_at: dict = field(default_factory=dict)
    last_gossip_lag_ms: float = 0.0

    def has_tx(self, tx_hash: str) -> bool:
        return tx_hash in self.arrived_at

    def receive(self, tx: Transaction, ts: float) -> None:
        if tx.tx_hash in self.arrived_at:
            return
        self.arrived_at[tx.tx_hash] = ts
        self.mempool.append(tx)


class NetworkSim:
    """Координатор N логических узлов с gossip-mock."""

    def __init__(
        self,
        num_nodes: int = 1,
        gossip_latency_ms: int = 100,
        gossip_loss_pct: float = 0.0,
        quorum_pct: float = 2.0 / 3.0,
    ) -> None:
        self.num_nodes = max(1, int(num_nodes))
        self.gossip_latency_ms = max(0, int(gossip_latency_ms))
        self.gossip_loss_pct = max(0.0, min(100.0, float(gossip_loss_pct)))
        self.quorum_pct = max(0.0, min(1.0, float(quorum_pct)))
        self.nodes: dict[str, Node] = {
            f"node_{i}": Node(id=f"node_{i}") for i in range(self.num_nodes)
        }
        # tx_hash → set of node_ids, куда tx уже доставлен
        self._coverage: dict[str, set[str]] = defaultdict(set)
        # tx_hash → Transaction (для flush_to_global)
        self._pool: dict[str, Transaction] = {}
        # tx_hash, попавшие в один из последних блоков (защита от повторного flush)
        self._consumed: set[str] = set()
        self._rng = random.Random(0xC0FFEE)
        self._lock = threading.Lock()

    # ---------------- attach / topology ----------------

    def attach(self, state_manager) -> None:
        """Привязать NetworkSim к StateManager, разнести существующие адреса."""
        from core.state import (
            GENESIS_PROVIDER_ADDR,
            GENESIS_VALIDATOR_ADDR,
            SIM_TREASURY_ADDR,
        )

        fixed = {
            GENESIS_VALIDATOR_ADDR: GENESIS_NODE_ID,
            GENESIS_PROVIDER_ADDR: GENESIS_NODE_ID,
            SIM_TREASURY_ADDR: GENESIS_NODE_ID,
        }
        for addr in state_manager.accounts.keys():
            node_id = fixed.get(addr) or _stable_node_for_addr(addr, self.num_nodes)
            self.nodes[node_id].addresses.add(addr)
        # Уже накопленный мемпул — отдаём узлу-источнику и сразу помечаем
        # коверадж 100%, чтобы не терять tx при переходе.
        for tx in list(state_manager.mempool):
            self._ingest_full_coverage(tx)
        state_manager.mempool.clear()

    def register_address(self, addr: str) -> str:
        from core.state import (
            GENESIS_PROVIDER_ADDR,
            GENESIS_VALIDATOR_ADDR,
            SIM_TREASURY_ADDR,
        )

        if addr in (GENESIS_VALIDATOR_ADDR, GENESIS_PROVIDER_ADDR, SIM_TREASURY_ADDR):
            node_id = GENESIS_NODE_ID
        else:
            node_id = _stable_node_for_addr(addr, self.num_nodes)
        self.nodes[node_id].addresses.add(addr)
        return node_id

    def node_id_for_addr(self, addr: str) -> str:
        for node_id, node in self.nodes.items():
            if addr in node.addresses:
                return node_id
        return self.register_address(addr)

    # ---------------- submission API ----------------

    def submit_to(self, node_id: str, tx: Transaction) -> None:
        """Прямая подача tx в local mempool заданного узла."""
        if node_id not in self.nodes:
            node_id = GENESIS_NODE_ID
        with self._lock:
            self._pool[tx.tx_hash] = tx
            node = self.nodes[node_id]
            now = time.time()
            node.receive(tx, now)
            self._coverage[tx.tx_hash].add(node_id)
            arrived_ts_origin = now
        self._schedule_gossip(tx, origin_node_id=node_id, origin_ts=arrived_ts_origin)

    def submit_from_addr(self, addr: str, tx: Transaction) -> str:
        node_id = self.node_id_for_addr(addr)
        self.submit_to(node_id, tx)
        return node_id

    def _ingest_full_coverage(self, tx: Transaction) -> None:
        """Импорт legacy tx: считать его доставленным на все узлы (без gossip)."""
        now = time.time()
        with self._lock:
            self._pool[tx.tx_hash] = tx
            for node in self.nodes.values():
                node.receive(tx, now)
                self._coverage[tx.tx_hash].add(node.id)

    # ---------------- gossip ----------------

    def _schedule_gossip(
        self, tx: Transaction, *, origin_node_id: str, origin_ts: float
    ) -> None:
        """Запланировать доставку tx остальным узлам с латентностью/потерями.

        Используется `asyncio.get_event_loop().call_later`, если loop работает;
        иначе синхронный fast-deliver (для тестов вне asyncio).
        """
        targets = [nid for nid in self.nodes.keys() if nid != origin_node_id]
        if not targets:
            return
        try:
            loop = asyncio.get_event_loop()
            loop_running = loop.is_running()
        except RuntimeError:
            loop_running = False

        for nid in targets:
            if self._rng.random() * 100.0 < self.gossip_loss_pct:
                continue
            delay = max(0.0, self.gossip_latency_ms / 1000.0)
            # Нулевая задержка → доставка немедленно: иначе flush_to_global ещё в
            # этом же тике увидит только origin-узел и блок останется пустым.
            if delay <= 0.0:
                self._deliver(tx, nid, origin_ts)
                continue
            if loop_running:
                try:
                    loop.call_later(delay, self._deliver, tx, nid, origin_ts)
                    continue
                except Exception:
                    pass
            # fallback: deliver immediately (тесты вне event loop)
            self._deliver(tx, nid, origin_ts)

    def _deliver(self, tx: Transaction, node_id: str, origin_ts: float) -> None:
        with self._lock:
            node = self.nodes.get(node_id)
            if not node or tx.tx_hash in self._consumed:
                return
            now = time.time()
            node.receive(tx, now)
            node.last_gossip_lag_ms = max(0.0, (now - origin_ts) * 1000.0)
            self._coverage[tx.tx_hash].add(node_id)

    # ---------------- runtime tuning ----------------

    def set_config(
        self,
        *,
        gossip_latency_ms: Optional[int] = None,
        gossip_loss_pct: Optional[float] = None,
        quorum_pct: Optional[float] = None,
    ) -> dict:
        if gossip_latency_ms is not None:
            self.gossip_latency_ms = max(0, int(gossip_latency_ms))
        if gossip_loss_pct is not None:
            self.gossip_loss_pct = max(0.0, min(100.0, float(gossip_loss_pct)))
        if quorum_pct is not None:
            self.quorum_pct = max(0.0, min(1.0, float(quorum_pct)))
        return self.snapshot_config()

    def snapshot_config(self) -> dict:
        return {
            "num_nodes": self.num_nodes,
            "gossip_latency_ms": self.gossip_latency_ms,
            "gossip_loss_pct": self.gossip_loss_pct,
            "quorum_pct": self.quorum_pct,
        }

    # ---------------- engine consumption ----------------

    def flush_to_global(self) -> List[Transaction]:
        """Вернуть tx, дошедшие до кворума узлов; пометить их consumed.

        Сохраняет порядок поступления (по времени первого arrival).
        """
        with self._lock:
            need = max(1, int(self.num_nodes * self.quorum_pct))
            if self.num_nodes == 1:
                need = 1
            ready: List[tuple[float, Transaction]] = []
            for tx_hash, tx in list(self._pool.items()):
                if tx_hash in self._consumed:
                    continue
                if len(self._coverage.get(tx_hash, ())) >= need:
                    # время «зрелости» — самый ранний arrival
                    first_ts = min(
                        node.arrived_at.get(tx_hash, time.time())
                        for node in self.nodes.values()
                        if tx_hash in node.arrived_at
                    )
                    ready.append((first_ts, tx))
            ready.sort(key=lambda p: p[0])
            out = [tx for _, tx in ready]
            for tx in out:
                self._consumed.add(tx.tx_hash)
                # очищаем local mempool каждого узла от уже отданных tx
                for node in self.nodes.values():
                    if not node.mempool:
                        continue
                    node.mempool = deque(
                        t for t in node.mempool if t.tx_hash != tx.tx_hash
                    )
            return out

    def iter_pending_txs(self) -> Iterable[Transaction]:
        """Все ещё не отданные engine tx (для проверки дубликатов declare и т.п.)."""
        with self._lock:
            return [
                tx for tx_hash, tx in self._pool.items() if tx_hash not in self._consumed
            ]

    # ---------------- introspection ----------------

    def nodes_summary(self, last_block: Optional[dict] = None) -> List[dict]:
        out: List[dict] = []
        round_log = (last_block or {}).get("consensus_round_log", []) or []
        proposer = (last_block or {}).get("proposer", "")
        votes_by_validator: dict[str, dict] = defaultdict(lambda: {"pre_vote": "", "pre_commit": ""})
        # Берём последний round (если их было > 1: timeout → re-round).
        for entry in round_log:
            votes = entry.get("votes") or {}
            for addr, value in (votes.get("pre_vote") or {}).items():
                votes_by_validator[addr]["pre_vote"] = value
            for addr, value in (votes.get("pre_commit") or {}).items():
                votes_by_validator[addr]["pre_commit"] = value
        for node_id, node in self.nodes.items():
            node_validators = [
                addr for addr in node.addresses if votes_by_validator.get(addr)
            ]
            out.append(
                {
                    "id": node_id,
                    "addresses": sorted(node.addresses),
                    "validators": node_validators,
                    "mempool_size": len(node.mempool),
                    "last_gossip_lag_ms": round(node.last_gossip_lag_ms, 2),
                    "is_proposer": proposer in node.addresses,
                    "votes": {
                        addr: votes_by_validator.get(addr, {})
                        for addr in node_validators
                    },
                }
            )
        out.sort(key=lambda n: n["id"])
        return out

    def topology(self) -> dict:
        """Полносвязная mesh для простоты; peers = все остальные узлы."""
        peers = {nid: sorted(n for n in self.nodes if n != nid) for nid in self.nodes}
        return {
            "config": self.snapshot_config(),
            "peers": peers,
        }
