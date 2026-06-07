from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum
from typing import ClassVar


@dataclass(frozen=True)
class Point:
    x: float
    y: float


class GeometryType(StrEnum):
    LINE = "line"
    POLYLINE = "polyline"
    ARC = "arc"
    TEXT = "text"
    INSERT = "insert"


@dataclass(frozen=True, kw_only=True)
class BaseEntity:
    id: str
    layer: str
    source_insert_id: str | None = None
    source_block_name: str | None = None


@dataclass(frozen=True, kw_only=True)
class LineEntity(BaseEntity):
    geometry_type: ClassVar[GeometryType] = GeometryType.LINE
    start: Point
    end: Point


@dataclass(frozen=True, kw_only=True)
class PolylineEntity(BaseEntity):
    geometry_type: ClassVar[GeometryType] = GeometryType.POLYLINE
    points: list[Point]
    closed: bool = False


@dataclass(frozen=True, kw_only=True)
class ArcEntity(BaseEntity):
    geometry_type: ClassVar[GeometryType] = GeometryType.ARC
    center: Point
    radius: float
    start_angle: float
    end_angle: float


@dataclass(frozen=True, kw_only=True)
class TextEntity(BaseEntity):
    geometry_type: ClassVar[GeometryType] = GeometryType.TEXT
    text: str
    insert: Point
    is_multiline: bool = False


@dataclass(frozen=True, kw_only=True)
class InsertEntity(BaseEntity):
    geometry_type: ClassVar[GeometryType] = GeometryType.INSERT
    block_name: str
    insert: Point
    rotation: float | None = None


NormalizedEntity = LineEntity | PolylineEntity | ArcEntity | TextEntity | InsertEntity
