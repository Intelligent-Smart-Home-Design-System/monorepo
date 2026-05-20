from __future__ import annotations

from internal.classification.classifier import ClassifiedEntities
from internal.entities.floor import Door, FloorMetadata, FloorPlan, Wall, Window


class TopologyBuilder:
    def build_floor(self, source_file: str, classified_entities: ClassifiedEntities, parsed_entity_count: int) -> FloorPlan:
        walls = classified_entities.walls
        doors = classified_entities.doors
        windows = classified_entities.windows

        return FloorPlan(
            schema_version="0.1.0",
            source_file=source_file,
            walls=walls,
            doors=doors,
            windows=windows,
            metadata=FloorMetadata(
                parsed_entity_count=parsed_entity_count,
                supported_attributes=self._collect_supported_attributes(walls, doors, windows),
            ),
        )

    def _collect_supported_attributes(
        self,
        walls: list[Wall],
        doors: list[Door],
        windows: list[Window],
    ) -> list[str]:
        supported_attributes = []
        if walls:
            supported_attributes.append("walls")
        if doors:
            supported_attributes.append("doors")
        if windows:
            supported_attributes.append("windows")
        return supported_attributes
