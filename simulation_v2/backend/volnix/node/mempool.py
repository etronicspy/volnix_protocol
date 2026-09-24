from __future__ import annotations

from volnix.app.abci import BaseApp
from volnix.types.tx import Tx, TxResult


class Mempool:
    def __init__(self, app: BaseApp) -> None:
        self.app = app
        self._txs: dict[str, Tx] = {}
        self._order: list[str] = []

    def check(self, tx: Tx) -> TxResult:
        return self.app.check_tx(tx)

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
        items.sort(key=lambda t: (-t.auth_info.fee.amount, t.tx_hash()))
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
