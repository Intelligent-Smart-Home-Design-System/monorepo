from __future__ import annotations

from services.floor_parser.internal.entities.floor import FloorPlan
from services.floor_parser.internal.entities.warnings import ParseWarning


class FloorExporter:
    def export(
        self,
        floor_plan: FloorPlan,
        source: str = "dxf",
        units: str | None = None,
        warnings: list[ParseWarning] | None = None
    ) -> dict[str, object]:
        return {
            "schema_version": floor_plan.schema_version,
            "meta": {
                "source": source,
                "source_ref": floor_plan.source_file,
                "units": units or "unknown",
            },
            "walls": [self._export_wall(wall) for wall in floor_plan.walls],
            "doors": [self._export_door(door) for door in floor_plan.doors],
            "windows": [self._export_window(window) for window in floor_plan.windows],
            "rooms": [],
            "warnings": [warning.to_dict() for warning in (warnings or [])]
        }

    def _export_wall(self, wall) -> dict[str, object]:
        return {
            "id": wall.id,
            "points": [
                [wall.start.x, wall.start.y],
                [wall.end.x, wall.end.y],
            ],
            "width": 0.0
        }

    def _export_door(self, door) -> dict[str, object]:
        return {
            "id": door.id,
            "points": [
                [door.start.x, door.start.y],
                [door.end.x, door.end.y],
            ],
            "width": door.length,
            "rooms": [],
        }

    def _export_window(self, window) -> dict[str, object]:
        return {
            "id": window.id,
            "points": [
                [window.start.x, window.start.y],
                [window.end.x, window.end.y],
            ],
            "width": window.length,
            "rooms": [],
        }
