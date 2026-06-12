from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class ParseWarning:
    code: str
    message: str

    def to_dict(self) -> dict[str, str]:
        return {
            "code": self.code,
            "message": self.message,
        }
