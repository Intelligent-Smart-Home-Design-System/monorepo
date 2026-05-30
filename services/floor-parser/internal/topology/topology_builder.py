from __future__ import annotations

from internal.classification.classifier import ClassifiedEntities
from internal.entities.floor import Door, FloorMetadata, FloorPlan, Room, Wall, Window
from internal.topology.room_builder import RoomBuilder


class TopologyBuilder:
    def __init__(self) -> None:
        self._room_builder = RoomBuilder()

    def build_floor(
        self,
        source_file: str,
        classified_entities: ClassifiedEntities,
        parsed_entity_count: int,
        units: str | None = None,
    ) -> FloorPlan:
        walls = classified_entities.walls
        rooms, doors, windows = self._room_builder.build(
            walls,
            classified_entities.doors,
            classified_entities.windows,
            classified_entities.texts,
            units=units,
        )

        return FloorPlan(
            schema_version="0.1.0",
            source_file=source_file,
            walls=walls,
            doors=doors,
            windows=windows,
            rooms=rooms,
            metadata=FloorMetadata(
                parsed_entity_count=parsed_entity_count,
                supported_attributes=self._collect_supported_attributes(walls, doors, windows, rooms),
            ),
        )

    def _collect_supported_attributes(
        self,
        walls: list[Wall],
        doors: list[Door],
        windows: list[Window],
        rooms: list[Room],
    ) -> list[str]:
        supported_attributes = []
        if walls:
            supported_attributes.append("walls")
        if doors:
            supported_attributes.append("doors")
        if windows:
            supported_attributes.append("windows")
        if rooms:
            supported_attributes.append("rooms")
        return supported_attributes
