<<<<<<< HEAD
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
=======
from __future__ import annotations

from dataclasses import dataclass, field
from enum import StrEnum

from .geometry import Point


class SourceFormat(StrEnum):
    DXF = "dxf"


class RawEntityType(StrEnum):
    LINE = "line"
    POLYLINE = "polyline"
    ARC = "arc"
    CIRCLE = "circle"
    TEXT = "text"
    INSERT = "insert"


@dataclass(frozen=True)
class SourceMetadata:
    source_format: SourceFormat
    source_file: str
    units: str | None = None
    entity_count: int = 0


@dataclass(frozen=True)
class RawEntity:
    id: str
    entity_type: RawEntityType
    layer: str

@dataclass(frozen=True)
class RawLine(RawEntity):
    start: Point
    end: Point

    def __init__(
        self,
        id: str,
        layer: str,
        start: Point,
        end: Point,
    ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "entity_type", RawEntityType.LINE)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "start", start)
        object.__setattr__(self, "end", end)


@dataclass(frozen=True)
class RawPolyline(RawEntity):
    points: list[Point]
    closed: bool = False

    def __init__(
        self,
        id: str,
        layer: str,
        points: list[Point],
        closed: bool = False
    ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "entity_type", RawEntityType.POLYLINE)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "points", points)
        object.__setattr__(self, "closed", closed)


@dataclass(frozen=True)
class RawArc(RawEntity):
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
        end_angle: float) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "entity_type", RawEntityType.ARC)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "center", center)
        object.__setattr__(self, "radius", radius)
        object.__setattr__(self, "start_angle", start_angle)
        object.__setattr__(self, "end_angle", end_angle)


@dataclass(frozen=True)
class RawCircle(RawEntity):
    center: Point
    radius: float

    def __init__(
        self,
        id: str,
        layer: str,
        center: Point,
        radius: float,
    ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "entity_type", RawEntityType.CIRCLE)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "center", center)
        object.__setattr__(self, "radius", radius)


@dataclass(frozen=True)
class RawText(RawEntity):
    text: str
    insert: Point
    is_multiline: bool = False

    def __init__(
        self,
        id: str,
        layer: str,
        text: str,
        insert: Point,
        is_multiline: bool = False
        ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "entity_type", RawEntityType.TEXT)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "text", text)
        object.__setattr__(self, "insert", insert)
        object.__setattr__(self, "is_multiline", is_multiline)

@dataclass(frozen=True)
class RawInsert(RawEntity):
    block_name: str
    insert: Point
    rotation: float | None = None

    def __init__(
        self,
        id: str,
        layer: str,
        block_name: str,
        insert: Point,
        rotation: float | None = None,
    ) -> None:
        object.__setattr__(self, "id", id)
        object.__setattr__(self, "entity_type", RawEntityType.INSERT)
        object.__setattr__(self, "layer", layer)
        object.__setattr__(self, "block_name", block_name)
        object.__setattr__(self, "insert", insert)
        object.__setattr__(self, "rotation", rotation)

RawPlanEntity = RawLine | RawPolyline | RawArc | RawCircle | RawText | RawInsert

@dataclass(frozen=True)
class RawPlan:
    metadata: SourceMetadata
    entities: list[RawPlanEntity] = field(default_factory=list)
>>>>>>> 4bf54f8 (hz)
