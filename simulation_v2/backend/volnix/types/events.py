from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class EventAttribute:
    key: str
    value: str

    def to_dict(self) -> dict:
        return {"key": self.key, "value": self.value}

    @classmethod
    def from_dict(cls, d: dict) -> EventAttribute:
        return cls(key=d["key"], value=d["value"])


@dataclass
class Event:
    type: str
    attributes: list[EventAttribute] = field(default_factory=list)

    def to_dict(self) -> dict:
        return {"type": self.type, "attributes": [a.to_dict() for a in self.attributes]}

    @classmethod
    def from_dict(cls, d: dict) -> Event:
        return cls(
            type=d["type"],
            attributes=[EventAttribute.from_dict(a) for a in d.get("attributes", [])],
        )


def ev(etype: str, **attrs: object) -> Event:
    return Event(type=etype, attributes=[EventAttribute(k, str(v)) for k, v in attrs.items()])
