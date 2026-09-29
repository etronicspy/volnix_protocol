from __future__ import annotations

from volnix.app.abci import BaseApp
from volnix.types.tx import Tx, TxResult


class Mempool:
    def __init__(self, app: BaseApp) -> None:
        self.app = app
        self._txs: dict[str, Tx] = {}
        self._order: list[str] = []

    def _signer_sequence(self, tx: Tx) -> int:
        if not tx.auth_info.signer_infos:
            return 0
        return int(tx.auth_info.signer_infos[0].sequence)

    def pending_count(self, address: str) -> int:
        return sum(1 for t in self.pending() if t.signer() == address)

    def _pending_count(self, address: str) -> int:
        return self.pending_count(address)

    def check(self, tx: Tx) -> TxResult:
        """Validate tx; sequence must be account.sequence + pending from signer."""
        info = tx.auth_info.signer_infos[0] if tx.auth_info.signer_infos else None
        if info is None:
            return self.app.check_tx(tx)
        signer = tx.signer()
        acc = self.app.state.accounts.get(signer)
        base = acc.sequence if acc else 0
        expected = base + self._pending_count(signer)
        if info.sequence != expected:
            return TxResult(
                code=1,
                log="incorrect sequence",
                gas_wanted=tx.auth_info.fee.gas_limit,
                gas_used=0,
            )
        if expected == base:
            return self.app.check_tx(tx)
        # Pending chain: temporarily align account.sequence for app check_tx.
        if acc is None:
            return self.app.check_tx(tx)
        saved = acc.sequence
        acc.sequence = info.sequence
        try:
            return self.app.check_tx(tx)
        finally:
            acc.sequence = saved

    def insert(self, tx: Tx) -> TxResult:
        result = self.check(tx)
        if result.code != 0:
            return result
        h = tx.tx_hash()
        # replace-by-sender for declare
        types = [m.type for m in tx.body.messages]
        if "povb/MsgDeclareParticipation" in types:
            sender = tx.signer()
            drop = []
            for existing_h in self._order:
                existing = self._txs[existing_h]
                if existing.signer() == sender and any(
                    m.type == "povb/MsgDeclareParticipation" for m in existing.body.messages
                ):
                    drop.append(existing_h)
            for d in drop:
                self._txs.pop(d, None)
                if d in self._order:
                    self._order.remove(d)
        if h not in self._txs:
            self._txs[h] = tx
            self._order.append(h)
        return result

    def reap(self, max_gas: int, max_bytes: int) -> list[Tx]:
        items = list(self._txs.values())
        # Preserve per-signer sequence order (send seq=n before declare seq=n+1).
        items.sort(
            key=lambda t: (
                t.signer(),
                self._signer_sequence(t),
                -t.auth_info.fee.amount,
                t.tx_hash(),
            )
        )
        selected: list[Tx] = []
        gas = 0
        size = 0
        used: list[str] = []
        for tx in items:
            g = tx.auth_info.fee.gas_limit
            s = len(tx.sign_bytes())
            if gas + g > max_gas or size + s > max_bytes:
                continue
            selected.append(tx)
            gas += g
            size += s
            used.append(tx.tx_hash())
        for h in used:
            self._txs.pop(h, None)
            if h in self._order:
                self._order.remove(h)
        return selected

    def pending(self) -> list[Tx]:
        return [self._txs[h] for h in self._order]

    def size(self) -> int:
        return len(self._txs)

    def clear(self) -> None:
        self._txs.clear()
        self._order.clear()
