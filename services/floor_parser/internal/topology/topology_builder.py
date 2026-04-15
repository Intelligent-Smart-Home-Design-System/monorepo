from __future__ import annotations

from math import sqrt

from services.floor_parser.internal.classification.classifier import ClassifiedEntities
from services.floor_parser.internal.entities.floor import FloorMetadata, FloorPlan, Wall
from services.floor_parser.internal.entities.geometry import LineEntity, Point, PolylineEntity


class TopologyBuilder:
    def build_floor(self, source_file: str, classified_entities: ClassifiedEntities, parsed_entity_count: int) -> FloorPlan:
        walls = self.build_walls(classified_entities)

        return FloorPlan(
            schema_version="0.1.0",
            source_file=source_file,
            walls=walls,
            metadata=FloorMetadata(
                parsed_entity_count=parsed_entity_count,
                supported_attributes=self._collect_supported_attributes(walls)
            )
        )

    def build_walls(self, classified_entities: ClassifiedEntities) -> list[Wall]:
        walls: list[Wall] = []

        for entity in classified_entities.walls:
            if isinstance(entity, LineEntity):
                wall = self._build_line_wall(entity)
                if wall is not None:
                    walls.append(wall)
                continue

            if isinstance(entity, PolylineEntity):
                walls.extend(self._build_polyline_walls(entity))

        return walls

    def _collect_supported_attributes(
        self,
        walls: list[Wall],
    ) -> list[str]:
        supported_attributes = []
        if walls:
            supported_attributes.append("walls")
        return supported_attributes

    def _build_line_wall(self, entity):
        length = self._get_length(entity.start, entity.end)

        return Wall(
            id=entity.id,
            layer=entity.layer,
            start=entity.start,
            end=entity.end,
            length=length,
            source_entity_ids=[entity.id],
        )

    def _build_polyline_walls(self, entity):
        walls = []
        points = entity.points

        if len(points) < 2:
            return walls

        for index in range(len(points) - 1):
            wall = self._build_segment_wall(
                wall_id=f"{entity.id}:{index + 1}",
                layer=entity.layer,
                start=points[index],
                end=points[index + 1],
                source_entity_id=entity.id,
            )
            walls.append(wall)
        
        if entity.closed:
            wall = self._build_segment_wall(
                wall_id=f"{entity.id}:closing",
                layer=entity.layer,
                start=points[-1],
                end=points[0],
                source_entity_id=entity.id,
            )
            walls.append(wall)
        
        return walls

    def _build_segment_wall(self, wall_id: str, layer: str, start: Point, end: Point, source_entity_id: str):
        length = self._get_length(start, end)
        return Wall(
            id=wall_id,
            layer=layer,
            start=start,
            end=end,
            length=length,
            source_entity_ids=[source_entity_id]
        )

    def _get_length(self, start: Point, end: Point) -> float:
        return round(sqrt((end.x - start.x)**2 + (end.y - start.y)**2), 6)
