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

    def to_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "layer": self.layer,
            "start": self.start.to_dict(),
            "end": self.end.to_dict(),
            "length": self.length,
            "sourceEntityIds": self.source_entity_ids,
        }


@dataclass(frozen=True)
class Window:
    id: str
    layer: str
    start: Point
    end: Point
    length: float
    source_entity_ids: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, object]:
        return {
            "id": self.id,
            "layer": self.layer,
            "start": self.start.to_dict(),
            "end": self.end.to_dict(),
            "length": self.length,
            "sourceEntityIds": self.source_entity_ids,
        }


@dataclass(frozen=True)
class FloorMetadata:
    parsed_entity_count: int
    supported_attributes: list[str]

    def to_dict(self) -> dict[str, object]:
        return {
            "parsedEntityCount": self.parsed_entity_count,
            "supportedAttributes": self.supported_attributes,
        }


@dataclass(frozen=True)
class FloorPlan:
    schema_version: str
    source_file: str
    walls: list[Wall] = field(default_factory=list)
    windows: list[Window] = field(default_factory=list)
    metadata: FloorMetadata = field(default_factory=lambda: FloorMetadata(parsed_entity_count=0, supported_attributes=[]))

    def to_dict(self) -> dict[str, object]:
        return {
            "schemaVersion": self.schema_version,
            "sourceFile": self.source_file,
            "walls": [wall.to_dict() for wall in self.walls],
            "windows": [window.to_dict() for window in self.windows],
            "metadata": self.metadata.to_dict(),
        }
