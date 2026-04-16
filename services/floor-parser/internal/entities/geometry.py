from __future__ import annotations

from dataclasses import dataclass
from enum import StrEnum


@dataclass(frozen=True)
class Point:
    x: float
    y: float

    def to_dict(self) -> dict[str, float]:
        return {"x": self.x, "y": self.y}


class GeometryType(StrEnum):
    LINE = "line"
    POLYLINE = "polyline"
    ARC = "arc"
    TEXT = "text"
    INSERT = "insert"


@dataclass(frozen=True)
class BaseEntity:
    id: str
    layer: str
    geometry_type: GeometryType


@dataclass(frozen=True)
class LineEntity(BaseEntity):
    start: Point
    end: Point

    def __init__(self, id: str, layer: str, start: Point, end: Point) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "geometry_type", GeometryType.LINE)
        object.__setattr__(self, "start", start)
        object.__setattr__(self, "end", end)


@dataclass(frozen=True)
class PolylineEntity(BaseEntity):
    points: list[Point]
    closed: bool

    def __init__(self, id: str, layer: str, points: list[Point], closed: bool = False) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "geometry_type", GeometryType.POLYLINE)
        object.__setattr__(self, "points", points)
        object.__setattr__(self, "closed", closed)


@dataclass(frozen=True)
class ArcEntity(BaseEntity):
    center: Point
    radius: float
    start_angle: float
    end_angle: float

    def __init__(
        self,
        id: str,
        layer: str,
        center: Point,
        radius: float,
        start_angle: float,
        end_angle: float,
    ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "geometry_type", GeometryType.ARC)
        object.__setattr__(self, "center", center)
        object.__setattr__(self, "radius", radius)
        object.__setattr__(self, "start_angle", start_angle)
        object.__setattr__(self, "end_angle", end_angle)


@dataclass(frozen=True)
class TextEntity(BaseEntity):
    text: str
    insert: Point
    is_multiline: bool

    def __init__(
        self,
        id: str,
        layer: str,
        text: str,
        insert: Point,
        is_multiline: bool = False,
    ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "geometry_type", GeometryType.TEXT)
        object.__setattr__(self, "text", text)
        object.__setattr__(self, "insert", insert)
        object.__setattr__(self, "is_multiline", is_multiline)


@dataclass(frozen=True)
class InsertEntity(BaseEntity):
    block_name: str
    insert: Point
    rotation: float | None

    def __init__(
        self,
        id: str,
        layer: str,
        block_name: str,
        insert: Point,
        rotation: float | None = None,
    ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "geometry_type", GeometryType.INSERT)
        object.__setattr__(self, "block_name", block_name)
        object.__setattr__(self, "insert", insert)
        object.__setattr__(self, "rotation", rotation)


NormalizedEntity = LineEntity | PolylineEntity | ArcEntity | TextEntity | InsertEntity
