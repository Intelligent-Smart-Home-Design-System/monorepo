from __future__ import annotations

from dataclasses import dataclass, field

from services.floor_parser.internal.entities.geometry import NormalizedEntity


WALL_LAYER_MARKERS = (
    "wall",
    "walls",
    "стена",
    "стены",
)


@dataclass(frozen=True)
class ClassifiedEntities:
    walls: list[NormalizedEntity] = field(default_factory=list)
    windows: list[NormalizedEntity] = field(default_factory=list)
    doors: list[NormalizedEntity] = field(default_factory=list)
    rooms: list[NormalizedEntity] = field(default_factory=list)
    unknown: list[NormalizedEntity] = field(default_factory=list)


class SemanticClassifier:
    def classify(self, entities: list[NormalizedEntity]) -> ClassifiedEntities:
        walls: list[NormalizedEntity] = []

        for entity in entities:
            if self._is_wall_layer(entity.layer):
                walls.append(entity)
                continue


        return ClassifiedEntities(
            walls=walls
        )

    def _is_wall_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return any(marker in normalized_layer for marker in WALL_LAYER_MARKERS)
