from __future__ import annotations

from services.floor_parser.internal.entities.geometry import LineEntity, NormalizedEntity, Point, PolylineEntity
from services.floor_parser.internal.entities.raw_plan import RawLine, RawPlan, RawPolyline


class GeometryNormalizer:
    def __init__(self, precision: int = 6) -> None:
        self._precision = precision

    def normalize(self, raw_plan: RawPlan) -> list[NormalizedEntity]:
        normalized_entities = []

        for entity in raw_plan.entities:
            if isinstance(entity, RawLine):
                normalized_entities.append(
                    LineEntity(
                        id=entity.id,
                        layer=entity.layer,
                        start=entity.start,
                        end=entity.end
                    )
                )
                continue

            if isinstance(entity, RawPolyline):
                normalized_entities.append(
                    PolylineEntity(
                        id=entity.id,
                        layer=entity.layer,
                        points=entity.points,
                        closed=entity.closed,
                    )
                )

        return normalized_entities

