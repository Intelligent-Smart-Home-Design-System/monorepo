from __future__ import annotations

from dataclasses import dataclass, field

from services.parser.internal.entities.geometry import NormalizedEntity


WALL_LAYER_MARKERS = (
    "wall",
    "walls",
    "стена",
    "стены",
)

WINDOW_LAYER_MARKERS = (
    "window",
    "windows",
    "окно",
    "окна",
)

DOOR_LAYER_MARKERS = (
    "door",
    "doors",
    "дверь",
    "двери",
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
        windows: list[NormalizedEntity] = []
        doors: list[NormalizedEntity] = []
        unknown: list[NormalizedEntity] = []

        for entity in entities:
            if self._is_wall_layer(entity.layer):
                walls.append(entity)
                continue

            if self._is_window_layer(entity.layer):
                windows.append(entity)
                continue

            if self._is_door_layer(entity.layer):
                doors.append(entity)
                continue

            unknown.append(entity)

        return ClassifiedEntities(
            walls=walls,
            windows=windows,
            doors=doors,
            unknown=unknown,
        )

    def _is_wall_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return any(marker in normalized_layer for marker in WALL_LAYER_MARKERS)

    def _is_window_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return any(marker in normalized_layer for marker in WINDOW_LAYER_MARKERS)

    def _is_door_layer(self, layer: str) -> bool:
        normalized_layer = layer.strip().lower()
        return any(marker in normalized_layer for marker in DOOR_LAYER_MARKERS)
