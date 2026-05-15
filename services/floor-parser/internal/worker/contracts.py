from __future__ import annotations

from dataclasses import dataclass


@dataclass
class ParseFloorInput:
    request_id: str
    source_path: str
    output_path: str


@dataclass
class ParseFloorOutput:
    request_id: str
    output_path: str
    wall_count: int
    door_count: int
    window_count: int
    warning_count: int
