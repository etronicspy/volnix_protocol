from __future__ import annotations

from enum import Enum


class Role(str, Enum):
    CITIZEN = "citizen"
    SUPPLIER = "supplier"
    VALIDATOR = "validator"
