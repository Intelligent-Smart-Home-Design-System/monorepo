from __future__ import annotations

from internal.entities.geometry import ArcEntity, InsertEntity, LineEntity, NormalizedEntity, PolylineEntity, TextEntity
from internal.entities.raw_plan import RawArc, RawInsert, RawLine, RawPlan, RawPolyline, RawText


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
                continue

            if isinstance(entity, RawArc):
                normalized_entities.append(
                    ArcEntity(
                        id=entity.id,
                        layer=entity.layer,
                        center=entity.center,
                        radius=entity.radius,
                        start_angle=entity.start_angle,
                        end_angle=entity.end_angle,
                    )
                )
                continue

            if isinstance(entity, RawText):
                normalized_entities.append(
                    TextEntity(
                        id=entity.id,
                        layer=entity.layer,
                        text=entity.text,
                        insert=entity.insert,
                        is_multiline=entity.is_multiline,
                    )
                )
                continue

            if isinstance(entity, RawInsert):
                normalized_entities.append(
                    InsertEntity(
                        id=entity.id,
                        layer=entity.layer,
                        block_name=entity.block_name,
                        insert=entity.insert,
                        rotation=entity.rotation,
                    )
                )

        return normalized_entities
