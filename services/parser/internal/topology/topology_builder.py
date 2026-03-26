from __future__ import annotations

from math import sqrt

from services.parser.internal.classification.classifier import ClassifiedEntities
from services.parser.internal.entities.floor import Door, FloorMetadata, FloorPlan, Wall, Window
from services.parser.internal.entities.geometry import LineEntity, Point, PolylineEntity


class TopologyBuilder:
    def build_floor(self, source_file: str, classified_entities: ClassifiedEntities, parsed_entity_count: int) -> FloorPlan:
        walls = self.build_walls(classified_entities)
        doors = self.build_doors(classified_entities)
        windows = self.build_windows(classified_entities)

        return FloorPlan(
            schema_version="0.1.0",
            source_file=source_file,
            walls=walls,
            doors=doors,
            windows=windows,
            metadata=FloorMetadata(
                parsed_entity_count=parsed_entity_count,
                supported_attributes=self._collect_supported_attributes(walls, doors, windows)
            )
        )

    def build_walls(self, classified_entities: ClassifiedEntities) -> list[Wall]:
        return self._build_linear_entities(
            entities=classified_entities.walls,
            line_builder=self._build_line_wall,
            segment_builder=self._build_segment_wall,
        )

    def build_doors(self, classified_entities: ClassifiedEntities) -> list[Door]:
        return self._build_linear_entities(
            entities=classified_entities.doors,
            line_builder=self._build_line_door,
            segment_builder=self._build_segment_door,
        )

    def build_windows(self, classified_entities: ClassifiedEntities) -> list[Window]:
        return self._build_linear_entities(
            entities=classified_entities.windows,
            line_builder=self._build_line_window,
            segment_builder=self._build_segment_window,
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

    def _build_linear_entities(self, entities, line_builder, segment_builder):
        built_entities = []

        for entity in entities:
            if isinstance(entity, LineEntity):
                built_entity = line_builder(entity)
                if built_entity is not None:
                    built_entities.append(built_entity)
                continue

            if isinstance(entity, PolylineEntity):
                built_entities.extend(
                    self._build_polyline_segments(
                        entity=entity,
                        segment_builder=segment_builder,
                    )
                )

        return built_entities

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

    def _build_line_door(self, entity):
        length = self._get_length(entity.start, entity.end)

        return Door(
            id=entity.id,
            layer=entity.layer,
            start=entity.start,
            end=entity.end,
            length=length,
            source_entity_ids=[entity.id],
        )

    def _build_line_window(self, entity):
        length = self._get_length(entity.start, entity.end)

        return Window(
            id=entity.id,
            layer=entity.layer,
            start=entity.start,
            end=entity.end,
            length=length,
            source_entity_ids=[entity.id],
        )

    def _build_polyline_segments(self, entity, segment_builder):
        built_entities = []
        points = entity.points

        if len(points) < 2:
            return built_entities

        for index in range(len(points) - 1):
            built_entity = segment_builder(
                entity_id=f"{entity.id}:{index + 1}",
                layer=entity.layer,
                start=points[index],
                end=points[index + 1],
                source_entity_id=entity.id,
            )
            built_entities.append(built_entity)

        if entity.closed:
            built_entity = segment_builder(
                entity_id=f"{entity.id}:closing",
                layer=entity.layer,
                start=points[-1],
                end=points[0],
                source_entity_id=entity.id,
            )
            built_entities.append(built_entity)

        return built_entities

    def _build_segment_wall(self, entity_id: str, layer: str, start: Point, end: Point, source_entity_id: str):
        length = self._get_length(start, end)
        return Wall(
            id=entity_id,
            layer=layer,
            start=start,
            end=end,
            length=length,
            source_entity_ids=[source_entity_id]
        )

    def _build_segment_door(self, entity_id: str, layer: str, start: Point, end: Point, source_entity_id: str):
        length = self._get_length(start, end)
        return Door(
            id=entity_id,
            layer=layer,
            start=start,
            end=end,
            length=length,
            source_entity_ids=[source_entity_id],
        )

    def _build_segment_window(self, entity_id: str, layer: str, start: Point, end: Point, source_entity_id: str):
        length = self._get_length(start, end)
        return Window(
            id=entity_id,
            layer=layer,
            start=start,
            end=end,
            length=length,
            source_entity_ids=[source_entity_id],
        )

    def _get_length(self, start: Point, end: Point) -> float:
        return round(sqrt((end.x - start.x)**2 + (end.y - start.y)**2), 6)
