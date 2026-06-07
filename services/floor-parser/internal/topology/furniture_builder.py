from __future__ import annotations

from collections import defaultdict
from dataclasses import replace
from math import inf

from internal.classification.config import UNITS_TO_MILLIMETERS
from internal.entities.floor import Furniture, Room
from internal.entities.geometry import ArcEntity, InsertEntity, LineEntity, Point, PolylineEntity, TextEntity

KITCHEN_LAYER = "kitchen"
KITCHEN_LAYER_ALIASES = frozenset({KITCHEN_LAYER, "kitchencabinets.us"})
FURNITURE_LAYERS = frozenset({"roomitems", "plumbing", *KITCHEN_LAYER_ALIASES})
ROOM_BATH_TOKENS = frozenset({"bath", "bathroom", "toil", "restroom"})
ROOM_BEDROOM_TOKENS = frozenset({"bedroom"})
ROOM_LIVING_TOKENS = frozenset({"living", "great", "family", "lounge"})
ROOM_DINING_TOKENS = frozenset({"dining"})
ROOM_KITCHEN_TOKENS = frozenset({"kitchen"})
ROOM_STORAGE_TOKENS = frozenset({"closet", "wic", "wardrobe"})
DETAIL_DENSITY_SCALE = 1_000_000.0
RECTANGULAR_MAX_ASPECT_RATIO = 3.6
BED_LONG_SIDE_RANGE_MM = (1_650.0, 2_350.0)
BED_SHORT_SIDE_RANGE_MM = (850.0, 2_050.0)
BED_MAX_DETAIL_DENSITY = 220.0
SOFA_LONG_SIDE_RANGE_MM = (1_400.0, 4_500.0)
SOFA_SHORT_SIDE_RANGE_MM = (800.0, 2_500.0)
SOFA_MIN_AREA_MM2 = 1_200_000.0
SOFA_MIN_DETAIL_DENSITY = 80.0
TABLE_LONG_SIDE_RANGE_MM = (1_000.0, 2_400.0)
TABLE_SHORT_SIDE_RANGE_MM = (700.0, 1_800.0)
TABLE_MIN_AREA_MM2 = 1_000_000.0
TABLE_MAX_DETAIL_DENSITY = 80.0
BATHTUB_LONG_SIDE_RANGE_MM = (1_400.0, 2_400.0)
BATHTUB_SHORT_SIDE_RANGE_MM = (650.0, 1_500.0)
BATHTUB_MAX_DETAIL_DENSITY = 220.0
TOILET_LONG_SIDE_RANGE_MM = (1_000.0, 1_250.0)
TOILET_SHORT_SIDE_RANGE_MM = (620.0, 850.0)
TOILET_MIN_DETAIL_DENSITY = 700.0
SINK_LONG_SIDE_RANGE_MM = (300.0, 1_200.0)
SINK_SHORT_SIDE_RANGE_MM = (280.0, 750.0)
SINK_MIN_DETAIL_DENSITY = 500.0
STOVE_LONG_SIDE_RANGE_MM = (550.0, 1_250.0)
STOVE_SHORT_SIDE_RANGE_MM = (500.0, 760.0)
STOVE_MIN_DETAIL_DENSITY = 1_800.0


class FurnitureBuilder:
    def build(
        self,
        entities: list,
        rooms: list[Room],
        units: str | None = None,
    ) -> tuple[list[Furniture], list[Room]]:
        insert_entities = [
            entity
            for entity in entities
            if isinstance(entity, InsertEntity)
            and entity.source_insert_id is None
            and entity.layer.lower() in FURNITURE_LAYERS
        ]
        if not insert_entities:
            return [], [replace(room, furniture=()) for room in rooms]

        geometry_by_insert: dict[str, list] = defaultdict(list)
        for entity in entities:
            if isinstance(entity, (InsertEntity, TextEntity)) or entity.source_insert_id is None:
                continue
            geometry_by_insert[entity.source_insert_id].append(entity)

        linear_scale = self._linear_scale_to_mm(units)
        room_by_id = {room.id: room for room in rooms}
        furniture_items: list[Furniture] = []

        for insert in sorted(insert_entities, key=lambda item: item.id):
            geometry = geometry_by_insert.get(insert.id, [])
            points = self._build_footprint(insert, geometry)
            room_id = self._assign_room(points, insert.insert, rooms)
            room_name = room_by_id[room_id].name if room_id in room_by_id else None
            canonical_layer = self._canonical_layer_name(insert.layer)
            width_mm, height_mm = self._footprint_dimensions_mm(points, linear_scale)
            polyline_count = sum(1 for entity in geometry if isinstance(entity, PolylineEntity))
            category = self._classify_furniture(
                canonical_layer,
                room_name,
                width_mm,
                height_mm,
                polyline_count,
            )
            furniture_items.append(
                Furniture(
                    id=insert.id,
                    layer=canonical_layer,
                    category=category,
                    points=points,
                    room=room_id,
                    rotation=insert.rotation,
                    source_block_name=insert.block_name,
                    source_entity_ids=[insert.id],
                )
            )

        furniture_ids_by_room: dict[str, list[str]] = defaultdict(list)
        for item in furniture_items:
            if item.room is not None:
                furniture_ids_by_room[item.room].append(item.id)

        updated_rooms = [
            replace(room, furniture=tuple(sorted(furniture_ids_by_room.get(room.id, []))))
            for room in rooms
        ]
        return furniture_items, updated_rooms

    def _build_footprint(self, insert: InsertEntity, geometry: list) -> list[Point]:
        sampled_points: list[Point] = []
        for entity in geometry:
            sampled_points.extend(self._entity_points(entity))

        if not sampled_points:
            return [
                Point(x=insert.insert.x, y=insert.insert.y),
                Point(x=insert.insert.x, y=insert.insert.y),
                Point(x=insert.insert.x, y=insert.insert.y),
                Point(x=insert.insert.x, y=insert.insert.y),
            ]

        min_x = min(point.x for point in sampled_points)
        max_x = max(point.x for point in sampled_points)
        min_y = min(point.y for point in sampled_points)
        max_y = max(point.y for point in sampled_points)
        if abs(max_x - min_x) < 1e-9:
            max_x += 1e-6
        if abs(max_y - min_y) < 1e-9:
            max_y += 1e-6
        return [
            Point(x=min_x, y=min_y),
            Point(x=max_x, y=min_y),
            Point(x=max_x, y=max_y),
            Point(x=min_x, y=max_y),
        ]

    def _entity_points(self, entity) -> list[Point]:
        if isinstance(entity, LineEntity):
            return [entity.start, entity.end]
        if isinstance(entity, PolylineEntity):
            return entity.points
        if isinstance(entity, ArcEntity):
            return [
                Point(x=entity.center.x - entity.radius, y=entity.center.y - entity.radius),
                Point(x=entity.center.x + entity.radius, y=entity.center.y + entity.radius),
            ]
        return []

    def _assign_room(
        self,
        points: list[Point],
        insert_point: Point,
        rooms: list[Room],
    ) -> str | None:
        centroid = self._polygon_centroid(points)
        for sample in (centroid, insert_point):
            for room in rooms:
                if self._point_in_polygon(sample, room.area):
                    return room.id

        best_room_id = None
        best_distance = inf
        for room in rooms:
            room_centroid = self._polygon_centroid(room.area)
            distance = (room_centroid.x - centroid.x) ** 2 + (room_centroid.y - centroid.y) ** 2
            if distance < best_distance:
                best_distance = distance
                best_room_id = room.id
        return best_room_id

    def _classify_furniture(
        self,
        layer: str,
        room_name: str | None,
        width_mm: float,
        height_mm: float,
        polyline_count: int,
    ) -> str:
        layer_name = layer.casefold()
        room_label = (room_name or "").casefold()
        long_side = max(width_mm, height_mm)
        short_side = min(width_mm, height_mm)
        area = width_mm * height_mm
        detail_density = self._detail_density(polyline_count, area)
        is_rectangular = self._is_rectangular(long_side, short_side)
        is_bathroom = self._room_has_token(room_label, ROOM_BATH_TOKENS)
        is_bedroom = self._room_has_token(room_label, ROOM_BEDROOM_TOKENS)
        is_living = self._room_has_token(room_label, ROOM_LIVING_TOKENS)
        is_dining = self._room_has_token(room_label, ROOM_DINING_TOKENS)
        is_kitchen = self._room_has_token(room_label, ROOM_KITCHEN_TOKENS)
        is_storage = self._room_has_token(room_label, ROOM_STORAGE_TOKENS)
        has_mixed_sleep_bath_context = is_bathroom and is_bedroom

        if layer_name == KITCHEN_LAYER:
            return "countertop"

        if is_bathroom and not has_mixed_sleep_bath_context and is_rectangular and self._matches_size_range(
            long_side,
            short_side,
            BATHTUB_LONG_SIDE_RANGE_MM,
            BATHTUB_SHORT_SIDE_RANGE_MM,
        ) and detail_density <= BATHTUB_MAX_DETAIL_DENSITY:
            return "bathtub"

        if is_bedroom and not has_mixed_sleep_bath_context and is_rectangular and self._matches_size_range(
            long_side,
            short_side,
            BED_LONG_SIDE_RANGE_MM,
            BED_SHORT_SIDE_RANGE_MM,
        ) and detail_density <= BED_MAX_DETAIL_DENSITY:
            return "bed"

        if is_storage:
            return "storage"

        if is_bathroom and self._matches_size_range(
            long_side,
            short_side,
            TOILET_LONG_SIDE_RANGE_MM,
            TOILET_SHORT_SIDE_RANGE_MM,
        ) and detail_density >= TOILET_MIN_DETAIL_DENSITY:
            return "toilet"

        if self._matches_size_range(
            long_side,
            short_side,
            STOVE_LONG_SIDE_RANGE_MM,
            STOVE_SHORT_SIDE_RANGE_MM,
        ) and detail_density >= STOVE_MIN_DETAIL_DENSITY and is_kitchen:
            return "stove"

        if self._matches_size_range(
            long_side,
            short_side,
            SINK_LONG_SIDE_RANGE_MM,
            SINK_SHORT_SIDE_RANGE_MM,
        ) and detail_density >= SINK_MIN_DETAIL_DENSITY and (is_bathroom or is_kitchen):
            return "sink"

        if is_rectangular and self._matches_size_range(
            long_side,
            short_side,
            SOFA_LONG_SIDE_RANGE_MM,
            SOFA_SHORT_SIDE_RANGE_MM,
        ) and area >= SOFA_MIN_AREA_MM2 and detail_density >= SOFA_MIN_DETAIL_DENSITY and (is_living or is_dining):
            return "sofa"

        if is_rectangular and self._matches_size_range(
            long_side,
            short_side,
            TABLE_LONG_SIDE_RANGE_MM,
            TABLE_SHORT_SIDE_RANGE_MM,
        ) and area >= TABLE_MIN_AREA_MM2 and detail_density <= TABLE_MAX_DETAIL_DENSITY and (is_kitchen or is_dining or is_living):
            return "table"

        return "unknown_furniture"

    def _detail_density(self, polyline_count: int, area: float) -> float:
        return polyline_count * DETAIL_DENSITY_SCALE / max(area, 1.0)

    def _is_rectangular(self, long_side: float, short_side: float) -> bool:
        if short_side <= 0.0:
            return False
        return long_side / short_side <= RECTANGULAR_MAX_ASPECT_RATIO

    def _room_has_token(self, room_label: str, tokens: frozenset[str]) -> bool:
        return any(token in room_label for token in tokens)

    def _matches_size_range(
        self,
        long_side: float,
        short_side: float,
        long_range: tuple[float, float],
        short_range: tuple[float, float],
    ) -> bool:
        return self._value_in_range(long_side, long_range) and self._value_in_range(short_side, short_range)

    def _value_in_range(self, value: float, bounds: tuple[float, float]) -> bool:
        lower, upper = bounds
        return lower <= value <= upper

    def _canonical_layer_name(self, layer: str) -> str:
        layer_name = layer.casefold()
        if layer_name in KITCHEN_LAYER_ALIASES:
            return KITCHEN_LAYER
        return layer_name

    def _footprint_dimensions_mm(self, points: list[Point], linear_scale: float | None) -> tuple[float, float]:
        min_x = min(point.x for point in points)
        max_x = max(point.x for point in points)
        min_y = min(point.y for point in points)
        max_y = max(point.y for point in points)
        width = max_x - min_x
        height = max_y - min_y
        if linear_scale is None:
            return width, height
        return width * linear_scale, height * linear_scale

    def _polygon_centroid(self, points: list[Point]) -> Point:
        if len(points) < 3:
            avg_x = sum(point.x for point in points) / len(points)
            avg_y = sum(point.y for point in points) / len(points)
            return Point(x=avg_x, y=avg_y)

        signed_area = 0.0
        centroid_x = 0.0
        centroid_y = 0.0
        for index, point in enumerate(points):
            next_point = points[(index + 1) % len(points)]
            cross = point.x * next_point.y - next_point.x * point.y
            signed_area += cross
            centroid_x += (point.x + next_point.x) * cross
            centroid_y += (point.y + next_point.y) * cross

        signed_area *= 0.5
        if abs(signed_area) < 1e-9:
            avg_x = sum(point.x for point in points) / len(points)
            avg_y = sum(point.y for point in points) / len(points)
            return Point(x=avg_x, y=avg_y)

        return Point(
            x=centroid_x / (6.0 * signed_area),
            y=centroid_y / (6.0 * signed_area),
        )

    def _point_in_polygon(self, point: Point, polygon: list[Point]) -> bool:
        inside = False
        for index, start in enumerate(polygon):
            end = polygon[(index + 1) % len(polygon)]
            if (start.y > point.y) == (end.y > point.y):
                continue
            intersect_x = (end.x - start.x) * (point.y - start.y) / (end.y - start.y + 1e-12) + start.x
            if point.x < intersect_x:
                inside = not inside
        return inside

    def _linear_scale_to_mm(self, units: str | None) -> float | None:
        if units is None:
            return None
        return UNITS_TO_MILLIMETERS.get(units.strip().lower())
