from __future__ import annotations

from dataclasses import dataclass, field

from internal.classification.wall_detector import WallDetector
from internal.entities.floor import Door, Wall, Window
from internal.entities.geometry import NormalizedEntity
from internal.classification.opening_detector import OpeningDetector

@dataclass(frozen=True)
class ClassifiedEntities:
    walls: list[Wall] = field(default_factory=list)
    windows: list[Window] = field(default_factory=list)
    doors: list[Door] = field(default_factory=list)
    rooms: list[NormalizedEntity] = field(default_factory=list)
    unknown: list[NormalizedEntity] = field(default_factory=list)


class SemanticClassifier:
    def __init__(self) -> None:
        self._opening_detector = OpeningDetector()
        self._wall_detector = WallDetector()

    def classify(self, entities: list[NormalizedEntity], units: str | None = None) -> ClassifiedEntities:
        walls = self._wall_detector.detect(entities, units=units)
        doors, windows = self._opening_detector.detect(entities, walls, units=units)

        return ClassifiedEntities(
            walls=walls,
            windows=windows,
            doors=doors,
            unknown=[],
        )
