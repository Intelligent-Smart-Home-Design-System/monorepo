from __future__ import annotations

from dataclasses import dataclass, field

from internal.entities.geometry import NormalizedEntity
from internal.classification.opening_detector import OpeningDetector


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

GARAGE_DOOR_LAYER_MARKERS = (
    "garage-door",
    "garage_door",
    "гараж",
)


@dataclass(frozen=True)
class ClassifiedEntities:
    walls: list[NormalizedEntity] = field(default_factory=list)
    windows: list[NormalizedEntity] = field(default_factory=list)
    doors: list[NormalizedEntity] = field(default_factory=list)
    rooms: list[NormalizedEntity] = field(default_factory=list)
    unknown: list[NormalizedEntity] = field(default_factory=list)


class SemanticClassifier:
    def __init__(self) -> None:
        self._opening_detector = OpeningDetector()

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

        doors.extend(self._opening_detector.detect_doors(entities))
        windows.extend(self._opening_detector.detect_windows(entities))

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
        if any(marker in normalized_layer for marker in GARAGE_DOOR_LAYER_MARKERS):
            return False
        return any(marker in normalized_layer for marker in DOOR_LAYER_MARKERS)
