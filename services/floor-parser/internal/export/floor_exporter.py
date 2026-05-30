from __future__ import annotations

from internal.classification.config import FLOOR_EXPORTER_CONFIG, UNITS_TO_MILLIMETERS
from internal.entities.floor import FloorPlan
from internal.entities.warnings import ParseWarning


class FloorExporter:
    def export(
        self,
        floor_plan: FloorPlan,
        source: str = "dxf",
        units: str | None = None,
        warnings: list[ParseWarning] | None = None
    ) -> dict[str, object]:
        sorted_doors = sorted(floor_plan.doors, key=lambda door: door.id)
        sorted_windows = sorted(floor_plan.windows, key=lambda window: window.id)
        sorted_rooms = sorted(floor_plan.rooms, key=lambda room: room.id)
        linear_scale = self._linear_scale_to_mm(units)
        export_walls = self._select_export_walls(floor_plan.walls)
        export_wall_ids = {wall.id for wall in export_walls}

        return {
            "schema_version": floor_plan.schema_version,
            "meta": {
                "source": source,
                "source_ref": floor_plan.source_file,
                "units": "mm" if linear_scale is not None else (units or "unknown"),
            },
            "walls": [self._export_wall(wall, linear_scale) for wall in export_walls],
            "doors": [self._export_door(door, linear_scale) for door in sorted_doors],
            "windows": [self._export_window(window, linear_scale) for window in sorted_windows],
            "rooms": [self._export_room(room, linear_scale, export_wall_ids) for room in sorted_rooms],
            "warnings": [warning.to_dict() for warning in (warnings or [])]
        }

    def _select_export_walls(self, walls) -> list:
        selected_connector_ids = self._select_boundary_connector_ids(walls)
        return [
            wall
            for wall in walls
            if (
                wall.geometry_role != "boundary_face"
                or wall.width == 0.0
                or wall.id in selected_connector_ids
            )
        ]

    def _select_boundary_connector_ids(self, walls) -> set[str]:
        connectors = [
            wall
            for wall in walls
            if self._is_uncovered_boundary_connector(wall, walls)
        ]
        if not connectors:
            return set()

        connector_by_id = {wall.id: wall for wall in connectors}
        neighbors: dict[str, set[str]] = {wall.id: set() for wall in connectors}

        for index, left in enumerate(connectors):
            for right in connectors[index + 1 :]:
                if self._walls_touch(left, right):
                    neighbors[left.id].add(right.id)
                    neighbors[right.id].add(left.id)

        selected_ids: set[str] = set()
        seen: set[str] = set()

        for connector in connectors:
            if connector.id in seen:
                continue

            stack = [connector.id]
            component: list[str] = []
            seen.add(connector.id)

            while stack:
                current_id = stack.pop()
                component.append(current_id)
                for neighbor_id in neighbors[current_id]:
                    if neighbor_id in seen:
                        continue
                    seen.add(neighbor_id)
                    stack.append(neighbor_id)

            if len(component) == 1:
                selected_ids.add(component[0])
                continue

            selected_ids.add(
                min(
                    component,
                    key=lambda connector_id: (
                        connector_by_id[connector_id].length,
                        connector_id,
                    ),
                )
            )

        return selected_ids

    def _is_uncovered_boundary_connector(self, wall, walls) -> bool:
        if wall.geometry_role != "boundary_face" or wall.width <= 0.0:
            return False
        if wall.length > wall.width * FLOOR_EXPORTER_CONFIG.boundary_connector_max_length_ratio:
            return False

        wall_rect = self._wall_rect(wall)
        if wall_rect is None:
            return False

        cover_area = 0.0
        for candidate in walls:
            if candidate.id == wall.id or candidate.geometry_role == "boundary_face":
                continue
            candidate_rect = self._wall_rect(candidate)
            if candidate_rect is None:
                continue
            cover_area += self._rect_intersection_area(wall_rect, candidate_rect)

        wall_area = self._rect_area(wall_rect)
        if wall_area <= 0.0:
            return False
        return cover_area / wall_area < FLOOR_EXPORTER_CONFIG.boundary_connector_cover_ratio_max

    def _wall_rect(self, wall) -> tuple[float, float, float, float] | None:
        if abs(wall.start.y - wall.end.y) <= 1e-6:
            center_y = (wall.start.y + wall.end.y) / 2.0
            return (
                min(wall.start.x, wall.end.x),
                center_y - wall.width / 2.0,
                max(wall.start.x, wall.end.x),
                center_y + wall.width / 2.0,
            )
        if abs(wall.start.x - wall.end.x) <= 1e-6:
            center_x = (wall.start.x + wall.end.x) / 2.0
            return (
                center_x - wall.width / 2.0,
                min(wall.start.y, wall.end.y),
                center_x + wall.width / 2.0,
                max(wall.start.y, wall.end.y),
            )
        return None

    def _rect_area(self, rect: tuple[float, float, float, float]) -> float:
        return max(0.0, rect[2] - rect[0]) * max(0.0, rect[3] - rect[1])

    def _rect_intersection_area(
        self,
        left: tuple[float, float, float, float],
        right: tuple[float, float, float, float],
    ) -> float:
        x0 = max(left[0], right[0])
        y0 = max(left[1], right[1])
        x1 = min(left[2], right[2])
        y1 = min(left[3], right[3])
        if x1 <= x0 or y1 <= y0:
            return 0.0
        return (x1 - x0) * (y1 - y0)

    def _walls_touch(self, left, right) -> bool:
        return any(
            self._same_point(a, b)
            for a in (left.start, left.end)
            for b in (right.start, right.end)
        )

    def _same_point(self, left, right) -> bool:
        return abs(left.x - right.x) <= 1e-6 and abs(left.y - right.y) <= 1e-6

    def _export_wall(self, wall, linear_scale: float | None) -> dict[str, object]:
        return {
            "id": wall.id,
            "points": self._segment_points(wall.start, wall.end, linear_scale),
            "width": self._to_mm(wall.width, linear_scale),
        }

    def _export_door(self, door, linear_scale: float | None) -> dict[str, object]:
        payload = {
            "id": door.id,
            "points": self._segment_points(door.start, door.end, linear_scale),
            "width": self._to_mm(door.length, linear_scale),
            "rooms": list(door.rooms),
        }
        if door.opens_towards_room is not None:
            payload["opens_towards_room"] = door.opens_towards_room
        if door.swing is not None:
            payload["swing"] = door.swing
        if door.hinge_side is not None:
            payload["hinge_side"] = door.hinge_side
        return payload

    def _export_window(self, window, linear_scale: float | None) -> dict[str, object]:
        payload = {
            "id": window.id,
            "points": self._segment_points(window.start, window.end, linear_scale),
            "width": self._to_mm(window.length, linear_scale),
        }
        if window.room is not None:
            payload["room"] = window.room
        return payload

    def _export_room(
        self,
        room,
        linear_scale: float | None,
        export_wall_ids: set[str],
    ) -> dict[str, object]:
        return {
            "id": room.id,
            "name": room.name,
            "area": [self._point_to_list(point, linear_scale) for point in room.area],
            "area_m2": room.area_m2,
            "windows": list(room.windows),
            "doors": list(room.doors),
            "walls": [wall_id for wall_id in room.walls if wall_id in export_wall_ids],
        }

    def _segment_points(self, start, end, linear_scale: float | None) -> list[list[float]]:
        return [
            self._point_to_list(start, linear_scale),
            self._point_to_list(end, linear_scale),
        ]

    def _point_to_list(self, point, linear_scale: float | None) -> list[float]:
        return [
            self._to_mm(point.x, linear_scale),
            self._to_mm(point.y, linear_scale),
        ]

    def _to_mm(self, value: float, linear_scale: float | None) -> float:
        if linear_scale is None:
            return value
        if linear_scale == 1.0:
            return value
        return round(value * linear_scale, 6)

    def _linear_scale_to_mm(self, units: str | None) -> float | None:
        if units is None:
            return None
        return UNITS_TO_MILLIMETERS.get(units.strip().lower())
