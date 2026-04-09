from __future__ import annotations

from dataclasses import dataclass, field

from .geometry import Point


@dataclass(frozen=True)
class Wall:
    id: str
    layer: str
    start: Point
    end: Point
    length: float
    source_entity_ids: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Window:
    id: str
    layer: str
    start: Point
    end: Point
    length: float
    source_entity_ids: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Door:
    id: str
    layer: str
    start: Point
    end: Point
    length: float
    source_entity_ids: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class FloorMetadata:
    parsed_entity_count: int
    supported_attributes: list[str]


@dataclass(frozen=True)
class FloorPlan:
    schema_version: str
    source_file: str
    walls: list[Wall] = field(default_factory=list)
    doors: list[Door] = field(default_factory=list)
    windows: list[Window] = field(default_factory=list)
    metadata: FloorMetadata = field(default_factory=lambda: FloorMetadata(parsed_entity_count=0, supported_attributes=[]))
