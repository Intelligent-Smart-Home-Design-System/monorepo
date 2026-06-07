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
    width: float
    geometry_role: str = "centerline"
    run_id: str | None = None
    source_entity_ids: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Window:
    id: str
    layer: str
    start: Point
    end: Point
    length: float
    wall_id: str | None = None
    room: str | None = None
    support_wall_ids: tuple[str, ...] = ()
    source_entity_ids: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Door:
    id: str
    layer: str
    start: Point
    end: Point
    length: float
    wall_id: str | None = None
    rooms: tuple[str, ...] = ()
    support_wall_ids: tuple[str, ...] = ()
    opens_towards_wall_side: str | None = None
    opens_towards_room: str | None = None
    swing: str | None = None
    hinge_side: str | None = None
    source_entity_ids: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Furniture:
    id: str
    layer: str
    category: str
    points: list[Point]
    room: str | None = None
    rotation: float | None = None
    source_block_name: str | None = None
    source_entity_ids: list[str] = field(default_factory=list)


@dataclass(frozen=True)
class Room:
    id: str
    name: str
    area: list[Point]
    area_m2: float
    windows: tuple[str, ...] = ()
    doors: tuple[str, ...] = ()
    walls: tuple[str, ...] = ()
    furniture: tuple[str, ...] = ()
@dataclass(frozen=True)
class FloorPlan:
    schema_version: str
    source_file: str
    walls: list[Wall] = field(default_factory=list)
    doors: list[Door] = field(default_factory=list)
    windows: list[Window] = field(default_factory=list)
    rooms: list[Room] = field(default_factory=list)
    furniture: list[Furniture] = field(default_factory=list)
