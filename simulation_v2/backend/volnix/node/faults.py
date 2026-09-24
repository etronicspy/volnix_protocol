from __future__ import annotations

from volnix.consensus.rounds import FaultModel


class FaultController:
    def __init__(self) -> None:
        self.model = FaultModel()

    def set_absent(self, addresses: list[str]) -> None:
        self.model.absent = set(addresses)

    def set_nil(self, addresses: list[str]) -> None:
        self.model.nil_vote = set(addresses)

    def clear(self) -> None:
        self.model = FaultModel()

    def snapshot(self) -> dict:
        return {"absent": sorted(self.model.absent), "nil_vote": sorted(self.model.nil_vote)}
