from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum
from typing import ClassVar

from .geometry import Point


class SourceFormat(StrEnum):
    DXF = "dxf"


class RawEntityType(StrEnum):
    LINE = "line"
    POLYLINE = "polyline"
    ARC = "arc"
    TEXT = "text"
    INSERT = "insert"


@dataclass(frozen=True)
class SourceMetadata:
    source_format: SourceFormat
    source_file: str
    units: str | None = None
    entity_count: int = 0


@dataclass(frozen=True, kw_only=True)
class RawEntity:
    id: str
    layer: str
    source_insert_id: str | None = None
    source_block_name: str | None = None


@dataclass(frozen=True, kw_only=True)
class RawLine(RawEntity):
    entity_type: ClassVar[RawEntityType] = RawEntityType.LINE
    start: Point
    end: Point


@dataclass(frozen=True, kw_only=True)
class RawPolyline(RawEntity):
    entity_type: ClassVar[RawEntityType] = RawEntityType.POLYLINE
    points: list[Point]
    closed: bool = False


@dataclass(frozen=True, kw_only=True)
class RawArc(RawEntity):
    entity_type: ClassVar[RawEntityType] = RawEntityType.ARC
    center: Point
    radius: float
    start_angle: float
    end_angle: float


@dataclass(frozen=True, kw_only=True)
class RawText(RawEntity):
    entity_type: ClassVar[RawEntityType] = RawEntityType.TEXT
    text: str
    insert: Point
    is_multiline: bool = False


@dataclass(frozen=True, kw_only=True)
class RawInsert(RawEntity):
    entity_type: ClassVar[RawEntityType] = RawEntityType.INSERT
    block_name: str
    insert: Point
    rotation: float | None = None


RawPlanEntity = RawLine | RawPolyline | RawArc | RawText | RawInsert


@dataclass(frozen=True)
class RawPlan:
    metadata: SourceMetadata
    entities: list[RawPlanEntity] = field(default_factory=list)
