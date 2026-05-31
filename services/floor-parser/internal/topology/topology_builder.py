from __future__ import annotations

from internal.classification.classifier import ClassifiedEntities
from internal.entities.floor import FloorPlan
from internal.topology.furniture_builder import FurnitureBuilder
from internal.topology.room_builder import RoomBuilder


class TopologyBuilder:
    def __init__(self) -> None:
        self._room_builder = RoomBuilder()
        self._furniture_builder = FurnitureBuilder()

    def build_floor(
        self,
        source_file: str,
        classified_entities: ClassifiedEntities,
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
        furniture, rooms = self._furniture_builder.build(classified_entities.entities, rooms, units=units)

        return FloorPlan(
            schema_version="0.1.0",
            source_file=source_file,
            walls=walls,
            doors=doors,
            windows=windows,
            rooms=rooms,
            furniture=furniture,
        )
