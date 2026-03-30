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


NormalizedEntity = LineEntity | PolylineEntity
