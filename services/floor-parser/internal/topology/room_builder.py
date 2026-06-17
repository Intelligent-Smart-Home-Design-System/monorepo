from __future__ import annotations

import re
from collections import defaultdict, deque
from dataclasses import dataclass, replace
from math import ceil

from internal.classification.config import UNITS_TO_MILLIMETERS
from internal.entities.floor import Door, Room, Wall, Window
from internal.entities.geometry import Point, TextEntity

GRID_TARGET_CELLS = 256
MIN_COMPONENT_CELLS = 4
MIN_ROOM_AREA_M2 = 0.5
MIN_LABELED_ROOM_AREA_M2 = 0.25
MIN_PLAUSIBLE_PLAN_AREA_M2 = 5.0
SUSPICIOUS_PLAN_MIN_COMPONENT_RATIO = 0.04
DEFAULT_ROOM_NAME_PREFIX = "Room"
ROOM_LABEL_CLUSTER_DISTANCE_CELLS = 12.0
MAX_ROOM_LABELS_PER_COMPONENT = 5
BOUNDARY_FACE_MIN_RADIUS_CELLS = 2.25
SEALED_GAP_MIN_NEIGHBORS = 3
GENERIC_ROOM_LABELS = frozenset({"room", "space", "area"})
SECONDARY_ROOM_LABEL_KEYWORDS = frozenset({"clo", "closet", "wic"})
CAD_TEXT_CONTROL_PATTERN = re.compile(r"%%[A-Za-z]")
ROOM_LABEL_KEYWORDS = frozenset(
    {
        "bath",
        "bathroom",
        "bedroom",
        "car",
        "clo",
        "closet",
        "covered",
        "deck",
        "dining",
        "entry",
        "ensuite",
        "family",
        "foyer",
        "garage",
        "great",
        "hall",
        "hallway",
        "kitchen",
        "laundry",
        "living",
        "master",
        "mstr",
        "office",
        "pantry",
        "porch",
        "room",
        "study",
        "toil",
        "toilet",
        "util",
        "utility",
        "vestibule",
        "wic",
    }
)


@dataclass(frozen=True)
class _GridSpec:
    origin_x: float
    origin_y: float
    cell_size: float
    width: int
    height: int
    span_x: float
    span_y: float


@dataclass(frozen=True)
class _RoomComponent:
    cells: frozenset[tuple[int, int]]
    polygon: tuple[Point, ...]
    wall_ids: tuple[str, ...]
    centroid_x: float
    centroid_y: float
    area_units: float
    area_m2: float


@dataclass(frozen=True)
class _RoomLabelFragment:
    text: str
    point: Point


class RoomBuilder:
    def build(
        self,
        walls: list[Wall],
        doors: list[Door],
        windows: list[Window],
        texts: list[TextEntity],
        units: str | None = None,
    ) -> tuple[list[Room], list[Door], list[Window]]:
        if not walls:
            return [], doors, windows

        grid = self._build_grid_spec(walls, doors, windows)
        if grid is None:
            return [], doors, windows

        occupied = [[False for _ in range(grid.width)] for _ in range(grid.height)]
        wall_cells: dict[tuple[int, int], set[str]] = defaultdict(set)
        boost_boundary_faces = any(wall.geometry_role != "boundary_face" and wall.width > 0.0 for wall in walls)
        wall_by_id, walls_by_run = self._build_wall_indexes(walls)

        self._rasterize_walls(walls, grid, occupied, wall_cells, boost_boundary_faces=boost_boundary_faces)
        self._rasterize_openings(doors, windows, grid, occupied, wall_by_id, walls_by_run)
        if boost_boundary_faces:
            self._seal_tiny_wall_gaps(occupied, wall_cells)

        outside = self._mark_outside(occupied)
        components = self._detect_components(grid, occupied, outside, wall_cells, units)
        if not components:
            return [], doors, windows

        room_definitions = self._build_room_definitions(components, texts, grid)
        room_definitions = self._filter_room_definitions(room_definitions)
        room_definitions = self._renumber_room_definitions(room_definitions)
        room_components = {
            room_definition["id"]: room_definition["component"]
            for room_definition in room_definitions
        }
        room_ids_in_order = [room_definition["id"] for room_definition in room_definitions]
        room_cell_lookup = self._build_room_cell_lookup(room_components)
        updated_doors = self._attach_doors_to_rooms(
            doors,
            grid,
            room_ids_in_order,
            room_cell_lookup,
            wall_by_id,
            walls_by_run,
        )
        updated_windows = self._attach_windows_to_rooms(
            windows,
            grid,
            room_ids_in_order,
            room_cell_lookup,
            wall_by_id,
            walls_by_run,
        )
        rooms = self._build_rooms(room_definitions, updated_doors, updated_windows)
        return rooms, updated_doors, updated_windows

    def _build_grid_spec(
        self,
        walls: list[Wall],
        doors: list[Door],
        windows: list[Window],
    ) -> _GridSpec | None:
        xs: list[float] = []
        ys: list[float] = []

        for wall in walls:
            radius = max(self._effective_wall_width(wall) / 2.0, 0.0)
            xs.extend((wall.start.x - radius, wall.start.x + radius, wall.end.x - radius, wall.end.x + radius))
            ys.extend((wall.start.y - radius, wall.start.y + radius, wall.end.y - radius, wall.end.y + radius))

        for opening in [*doors, *windows]:
            xs.extend((opening.start.x, opening.end.x))
            ys.extend((opening.start.y, opening.end.y))

        if not xs or not ys:
            return None

        min_x = min(xs)
        max_x = max(xs)
        min_y = min(ys)
        max_y = max(ys)
        span_x = max(max_x - min_x, 1.0)
        span_y = max(max_y - min_y, 1.0)
        max_span = max(span_x, span_y)

        cell_size = max(max_span / GRID_TARGET_CELLS, 1e-3)
        padding = cell_size * 4.0
        origin_x = min_x - padding
        origin_y = min_y - padding
        width = max(8, int(ceil((max_x - min_x + padding * 2.0) / cell_size)) + 1)
        height = max(8, int(ceil((max_y - min_y + padding * 2.0) / cell_size)) + 1)

        return _GridSpec(
            origin_x=origin_x,
            origin_y=origin_y,
            cell_size=cell_size,
            width=width,
            height=height,
            span_x=span_x,
            span_y=span_y,
        )

    def _rasterize_walls(
        self,
        walls: list[Wall],
        grid: _GridSpec,
        occupied: list[list[bool]],
        wall_cells: dict[tuple[int, int], set[str]],
        *,
        boost_boundary_faces: bool,
    ) -> None:
        for wall in walls:
            if wall.geometry_role == "boundary_face":
                min_radius_cells = BOUNDARY_FACE_MIN_RADIUS_CELLS if boost_boundary_faces else 0.75
                radius = max(self._effective_wall_width(wall) / 2.0, grid.cell_size * min_radius_cells)
            else:
                radius = max(self._effective_wall_width(wall) / 2.0, grid.cell_size * 0.75)
            for cell in self._cells_near_segment(wall.start, wall.end, radius, grid):
                occupied[cell[1]][cell[0]] = True
                wall_cells[cell].add(wall.id)

    def _seal_tiny_wall_gaps(
        self,
        occupied: list[list[bool]],
        wall_cells: dict[tuple[int, int], set[str]],
    ) -> None:
        height = len(occupied)
        width = len(occupied[0]) if height else 0
        to_fill: list[tuple[int, int]] = []

        for y in range(1, height - 1):
            for x in range(1, width - 1):
                if occupied[y][x]:
                    continue

                left = occupied[y][x - 1]
                right = occupied[y][x + 1]
                up = occupied[y - 1][x]
                down = occupied[y + 1][x]

                if (left and right) or (up and down) or sum((left, right, up, down)) >= SEALED_GAP_MIN_NEIGHBORS:
                    to_fill.append((x, y))

        for x, y in to_fill:
            occupied[y][x] = True
            neighboring_wall_ids: set[str] = set()
            for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
                neighboring_wall_ids.update(wall_cells.get((nx, ny), set()))
            if neighboring_wall_ids:
                wall_cells[(x, y)].update(neighboring_wall_ids)

    def _rasterize_openings(
        self,
        doors: list[Door],
        windows: list[Window],
        grid: _GridSpec,
        occupied: list[list[bool]],
        wall_by_id: dict[str, Wall],
        walls_by_run: dict[str, list[Wall]],
    ) -> None:
        for opening in [*doors, *windows]:
            host_wall = self._resolve_host_wall(opening, wall_by_id, walls_by_run)
            support_walls = [wall_by_id[wall_id] for wall_id in opening.support_wall_ids if wall_id in wall_by_id]
            widths = [self._effective_wall_width(wall) for wall in support_walls if self._effective_wall_width(wall) > 0.0]
            if host_wall is not None and self._effective_wall_width(host_wall) > 0.0:
                widths.append(self._effective_wall_width(host_wall))
            radius = max((max(widths) / 2.0) if widths else 0.0, grid.cell_size * 0.75)
            for cell in self._cells_near_segment(opening.start, opening.end, radius, grid):
                occupied[cell[1]][cell[0]] = True

    def _mark_outside(self, occupied: list[list[bool]]) -> list[list[bool]]:
        height = len(occupied)
        width = len(occupied[0]) if height else 0
        outside = [[False for _ in range(width)] for _ in range(height)]
        queue: deque[tuple[int, int]] = deque()

        for x in range(width):
            if not occupied[0][x]:
                queue.append((x, 0))
            if not occupied[height - 1][x]:
                queue.append((x, height - 1))
        for y in range(height):
            if not occupied[y][0]:
                queue.append((0, y))
            if not occupied[y][width - 1]:
                queue.append((width - 1, y))

        while queue:
            x, y = queue.popleft()
            if x < 0 or y < 0 or x >= width or y >= height:
                continue
            if outside[y][x] or occupied[y][x]:
                continue

            outside[y][x] = True
            queue.extend(((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)))

        return outside

    def _detect_components(
        self,
        grid: _GridSpec,
        occupied: list[list[bool]],
        outside: list[list[bool]],
        wall_cells: dict[tuple[int, int], set[str]],
        units: str | None,
    ) -> list[_RoomComponent]:
        owner = [[-1 for _ in range(grid.width)] for _ in range(grid.height)]
        candidate_components: list[_RoomComponent] = []
        suspicious_plan_scale = self._is_suspicious_plan_scale(grid, units)
        min_component_cells = self._min_component_cells(grid, units)

        for y in range(grid.height):
            for x in range(grid.width):
                if occupied[y][x] or outside[y][x] or owner[y][x] != -1:
                    continue

                cells = self._flood_component(x, y, occupied, outside, owner, len(candidate_components))
                if len(cells) < min_component_cells:
                    continue

                polygon = self._component_polygon(cells, grid)
                if len(polygon) < 3:
                    continue

                wall_ids = self._component_wall_ids(cells, occupied, wall_cells, grid)
                centroid_x = sum(self._cell_center(cell[0], cell[1], grid).x for cell in cells) / len(cells)
                centroid_y = sum(self._cell_center(cell[0], cell[1], grid).y for cell in cells) / len(cells)
                area_units = abs(self._polygon_area(polygon))
                area_m2 = self._area_to_square_meters(area_units, units)

                candidate_components.append(
                    _RoomComponent(
                        cells=frozenset(cells),
                        polygon=tuple(polygon),
                        wall_ids=tuple(sorted(wall_ids)),
                        centroid_x=round(centroid_x, 6),
                        centroid_y=round(centroid_y, 6),
                        area_units=round(area_units, 6),
                        area_m2=round(area_m2, 6),
                    )
                )

        if suspicious_plan_scale and candidate_components:
            min_component_cells = max(
                MIN_COMPONENT_CELLS,
                int(ceil(max(len(component.cells) for component in candidate_components) * SUSPICIOUS_PLAN_MIN_COMPONENT_RATIO)),
            )

        components = [
            component
            for component in candidate_components
            if len(component.cells) >= min_component_cells
        ]
        components.sort(key=lambda component: (-component.centroid_y, component.centroid_x))
        return components

    def _build_room_definitions(
        self,
        components: list[_RoomComponent],
        texts: list[TextEntity],
        grid: _GridSpec,
    ) -> list[dict[str, object]]:
        room_definitions: list[dict[str, object]] = []
        for index, component in enumerate(components, start=1):
            room_definitions.append(
                {
                    "id": f"room_{index}",
                    "name": f"{DEFAULT_ROOM_NAME_PREFIX} {index}",
                    "component": component,
                }
            )

        room_names = self._assign_room_names(room_definitions, texts, grid)
        for room_definition in room_definitions:
            room_name = room_names.get(room_definition["id"])
            if room_name is not None:
                room_definition["name"] = room_name
        return room_definitions

    def _filter_room_definitions(self, room_definitions: list[dict[str, object]]) -> list[dict[str, object]]:
        filtered: list[dict[str, object]] = []
        for room_definition in room_definitions:
            component: _RoomComponent = room_definition["component"]
            room_name = room_definition["name"]
            is_default_name = room_name.startswith(f"{DEFAULT_ROOM_NAME_PREFIX} ")
            min_area = MIN_ROOM_AREA_M2 if is_default_name else MIN_LABELED_ROOM_AREA_M2
            if component.area_m2 + 1e-9 < min_area:
                continue
            filtered.append(room_definition)
        return filtered

    def _renumber_room_definitions(self, room_definitions: list[dict[str, object]]) -> list[dict[str, object]]:
        renumbered: list[dict[str, object]] = []
        for index, room_definition in enumerate(room_definitions, start=1):
            renumbered.append(
                {
                    "id": f"room_{index}",
                    "name": room_definition["name"],
                    "component": room_definition["component"],
                }
            )
        return renumbered

    def _assign_room_names(
        self,
        room_definitions: list[dict[str, object]],
        texts: list[TextEntity],
        grid: _GridSpec,
    ) -> dict[str, str]:
        room_components = {
            room_definition["id"]: room_definition["component"]
            for room_definition in room_definitions
        }
        room_cell_lookup = self._build_room_cell_lookup(room_components)
        fragments_by_room: dict[str, list[_RoomLabelFragment]] = defaultdict(list)

        for text_entity in texts:
            if text_entity.source_insert_id is not None:
                continue

            room_id = self._room_id_for_text(text_entity, room_cell_lookup, grid)
            if room_id is None:
                continue

            for fragment_text in self._extract_room_label_fragments(text_entity.text):
                fragments_by_room[room_id].append(
                    _RoomLabelFragment(
                        text=fragment_text,
                        point=text_entity.insert,
                    )
                )

        room_names: dict[str, str] = {}
        for room_id, fragments in fragments_by_room.items():
            room_name = self._compose_room_name(fragments, room_components[room_id], grid)
            if room_name is not None:
                room_names[room_id] = room_name
        return room_names

    def _room_id_for_text(
        self,
        text_entity: TextEntity,
        room_cell_lookup: dict[tuple[int, int], str],
        grid: _GridSpec,
    ) -> str | None:
        offsets = [
            Point(x=0.0, y=0.0),
            Point(x=grid.cell_size * 1.5, y=0.0),
            Point(x=-grid.cell_size * 1.5, y=0.0),
            Point(x=0.0, y=grid.cell_size * 1.5),
            Point(x=0.0, y=-grid.cell_size * 1.5),
            Point(x=grid.cell_size * 3.0, y=0.0),
            Point(x=grid.cell_size * 1.5, y=grid.cell_size * 1.5),
            Point(x=grid.cell_size * 1.5, y=-grid.cell_size * 1.5),
            Point(x=-grid.cell_size * 1.5, y=grid.cell_size * 1.5),
            Point(x=-grid.cell_size * 1.5, y=-grid.cell_size * 1.5),
        ]

        for offset in offsets:
            sample = Point(
                x=text_entity.insert.x + offset.x,
                y=text_entity.insert.y + offset.y,
            )
            room_id = self._room_id_for_point(sample, room_cell_lookup, grid)
            if room_id is not None:
                return room_id

        return None

    def _extract_room_label_fragments(self, text: str) -> list[str]:
        fragments: list[str] = []
        seen: set[str] = set()

        for raw_line in text.replace("\r", "\n").splitlines():
            line = CAD_TEXT_CONTROL_PATTERN.sub("", raw_line)
            line = " ".join(line.split())
            if not line:
                continue

            candidate = line
            digit_index = next((index for index, char in enumerate(candidate) if char.isdigit()), None)
            if digit_index is not None:
                candidate = candidate[:digit_index]

            candidate = candidate.strip(" -–—,:;()[]{}")
            if (not candidate or not any(char.isalpha() for char in candidate)) and any(char.isalpha() for char in line):
                alpha_tokens = [
                    token
                    for token in (part.strip(" -–—,:;()[]{}") for part in line.split())
                    if token and any(char.isalpha() for char in token) and not any(char.isdigit() for char in token)
                ]
                candidate = " ".join(alpha_tokens)

            candidate = candidate.strip(" -–—,:;()[]{}")
            if (
                not candidate
                or not any(char.isalpha() for char in candidate)
                or len(candidate.split()) > 3
                or not self._looks_like_room_label(candidate)
            ):
                continue

            normalized_candidate = candidate.casefold()
            if normalized_candidate in seen:
                continue

            seen.add(normalized_candidate)
            fragments.append(candidate)

        return fragments

    def _compose_room_name(
        self,
        fragments: list[_RoomLabelFragment],
        component: _RoomComponent,
        grid: _GridSpec,
    ) -> str | None:
        if not fragments:
            return None

        cluster_distance = max(grid.cell_size * ROOM_LABEL_CLUSTER_DISTANCE_CELLS, grid.cell_size * 2.0)
        clusters = self._cluster_room_label_fragments(fragments, cluster_distance)
        label_entries = [
            self._build_room_label_entry(cluster, component)
            for cluster in clusters
        ]
        label_entries = [entry for entry in label_entries if entry is not None]
        if not label_entries:
            return None

        non_generic_labels = [entry for entry in label_entries if not self._is_generic_room_label(entry["label"])]
        if non_generic_labels:
            label_entries = non_generic_labels

        primary_labels = [entry for entry in label_entries if not self._is_secondary_room_label(entry["label"])]
        if primary_labels:
            label_entries = primary_labels

        seen: set[str] = set()
        unique_entries: list[dict[str, object]] = []
        for entry in label_entries:
            normalized_label = entry["label"].casefold()
            if normalized_label in seen:
                continue
            seen.add(normalized_label)
            unique_entries.append(entry)

        if len(unique_entries) > MAX_ROOM_LABELS_PER_COMPONENT:
            unique_entries.sort(key=self._room_label_priority)
            unique_entries = unique_entries[:MAX_ROOM_LABELS_PER_COMPONENT]
            unique_entries.sort(key=lambda entry: (-entry["center"].y, entry["center"].x, entry["label"].casefold()))

        labels = [entry["label"] for entry in unique_entries]
        return " / ".join(labels) if labels else None

    def _cluster_room_label_fragments(
        self,
        fragments: list[_RoomLabelFragment],
        cluster_distance: float,
    ) -> list[list[_RoomLabelFragment]]:
        neighbors: dict[int, list[int]] = defaultdict(list)

        for index, left in enumerate(fragments):
            for right_index in range(index + 1, len(fragments)):
                right = fragments[right_index]
                if self._distance_between_points(left.point, right.point) > cluster_distance:
                    continue
                neighbors[index].append(right_index)
                neighbors[right_index].append(index)

        clusters: list[list[_RoomLabelFragment]] = []
        seen: set[int] = set()

        for index, fragment in enumerate(fragments):
            if index in seen:
                continue

            stack = [index]
            cluster_indices: list[int] = []
            seen.add(index)

            while stack:
                current = stack.pop()
                cluster_indices.append(current)
                for neighbor in neighbors[current]:
                    if neighbor in seen:
                        continue
                    seen.add(neighbor)
                    stack.append(neighbor)

            cluster = [fragments[cluster_index] for cluster_index in cluster_indices]
            cluster.sort(key=lambda item: (-item.point.y, item.point.x, item.text.casefold()))
            clusters.append(cluster)

        clusters.sort(
            key=lambda cluster: (
                -sum(fragment.point.y for fragment in cluster) / len(cluster),
                sum(fragment.point.x for fragment in cluster) / len(cluster),
            )
        )
        return clusters

    def _compose_room_label_cluster(self, fragments: list[_RoomLabelFragment]) -> str | None:
        words: list[str] = []
        seen: set[str] = set()

        for fragment in fragments:
            normalized_text = fragment.text.casefold()
            if normalized_text in seen:
                continue
            seen.add(normalized_text)
            words.append(fragment.text)

        if not words:
            return None
        return " ".join(words)

    def _build_room_label_entry(
        self,
        fragments: list[_RoomLabelFragment],
        component: _RoomComponent,
    ) -> dict[str, object] | None:
        label = self._compose_room_label_cluster(fragments)
        if label is None:
            return None

        center = Point(
            x=sum(fragment.point.x for fragment in fragments) / len(fragments),
            y=sum(fragment.point.y for fragment in fragments) / len(fragments),
        )
        distance_to_centroid = self._distance_between_points(
            center,
            Point(x=component.centroid_x, y=component.centroid_y),
        )
        return {
            "label": label,
            "center": center,
            "distance_to_centroid": distance_to_centroid,
            "fragment_count": len(fragments),
        }

    def _is_generic_room_label(self, label: str) -> bool:
        tokens = [token.casefold() for token in label.split()]
        return bool(tokens) and all(token in GENERIC_ROOM_LABELS for token in tokens)

    def _is_secondary_room_label(self, label: str) -> bool:
        tokens = self._room_label_tokens(label)
        return bool(tokens) and all(token in SECONDARY_ROOM_LABEL_KEYWORDS for token in tokens)

    def _room_label_priority(self, entry: dict[str, object]) -> tuple[object, ...]:
        label = entry["label"]
        return (
            1.0 if self._is_generic_room_label(label) else 0.0,
            -float(entry["fragment_count"]),
            -float(min(len(label), 24)),
            float(entry["distance_to_centroid"]),
            label.casefold(),
        )

    def _looks_like_room_label(self, candidate: str) -> bool:
        tokens = self._room_label_tokens(candidate)
        if not tokens:
            return False
        return any(token in ROOM_LABEL_KEYWORDS for token in tokens)

    def _room_label_tokens(self, candidate: str) -> list[str]:
        tokens: list[str] = []
        for raw_token in candidate.split():
            token = raw_token.casefold().strip(".,:;()[]{}+-/&")
            if not token:
                continue
            token = token.replace("'", "")
            token = token.replace('"', "")
            token = token.replace("w.i.c", "wic")
            if token:
                tokens.append(token)
        return tokens

    def _attach_doors_to_rooms(
        self,
        doors: list[Door],
        grid: _GridSpec,
        room_ids_in_order: list[str],
        room_cell_lookup: dict[tuple[int, int], str],
        wall_by_id: dict[str, Wall],
        walls_by_run: dict[str, list[Wall]],
    ) -> list[Door]:
        updated_doors: list[Door] = []
        for door in doors:
            host_wall = self._resolve_host_wall(door, wall_by_id, walls_by_run)
            positive_room, negative_room, rooms = self._opening_room_binding(
                opening=door,
                host_wall=host_wall,
                room_ids_in_order=room_ids_in_order,
                room_cell_lookup=room_cell_lookup,
                grid=grid,
            )

            opens_towards_room = None
            if door.opens_towards_wall_side == "positive_normal":
                opens_towards_room = positive_room
            elif door.opens_towards_wall_side == "negative_normal":
                opens_towards_room = negative_room
            if opens_towards_room not in rooms:
                opens_towards_room = None

            updated_doors.append(
                replace(
                    door,
                    rooms=rooms,
                    opens_towards_room=opens_towards_room,
                )
            )

        return updated_doors

    def _attach_windows_to_rooms(
        self,
        windows: list[Window],
        grid: _GridSpec,
        room_ids_in_order: list[str],
        room_cell_lookup: dict[tuple[int, int], str],
        wall_by_id: dict[str, Wall],
        walls_by_run: dict[str, list[Wall]],
    ) -> list[Window]:
        updated_windows: list[Window] = []
        for window in windows:
            host_wall = self._resolve_host_wall(window, wall_by_id, walls_by_run)
            _positive_room, _negative_room, rooms = self._opening_room_binding(
                opening=window,
                host_wall=host_wall,
                room_ids_in_order=room_ids_in_order,
                room_cell_lookup=room_cell_lookup,
                grid=grid,
            )
            room = rooms[0] if rooms else None

            updated_windows.append(replace(window, room=room))

        return updated_windows

    def _build_rooms(
        self,
        room_definitions: list[dict[str, object]],
        doors: list[Door],
        windows: list[Window],
    ) -> list[Room]:
        room_doors: dict[str, list[str]] = defaultdict(list)
        room_windows: dict[str, list[str]] = defaultdict(list)

        for door in doors:
            for room_id in door.rooms:
                room_doors[room_id].append(door.id)

        for window in windows:
            if window.room is not None:
                room_windows[window.room].append(window.id)

        rooms: list[Room] = []
        for room_definition in room_definitions:
            room_id = room_definition["id"]
            component: _RoomComponent = room_definition["component"]
            rooms.append(
                Room(
                    id=room_id,
                    name=room_definition["name"],
                    area=list(component.polygon),
                    area_m2=component.area_m2,
                    windows=tuple(sorted(room_windows.get(room_id, []))),
                    doors=tuple(sorted(room_doors.get(room_id, []))),
                    walls=component.wall_ids,
                )
            )
        return rooms

    def _flood_component(
        self,
        start_x: int,
        start_y: int,
        occupied: list[list[bool]],
        outside: list[list[bool]],
        owner: list[list[int]],
        component_id: int,
    ) -> set[tuple[int, int]]:
        height = len(occupied)
        width = len(occupied[0]) if height else 0
        queue: deque[tuple[int, int]] = deque([(start_x, start_y)])
        cells: set[tuple[int, int]] = set()

        while queue:
            x, y = queue.popleft()
            if x < 0 or y < 0 or x >= width or y >= height:
                continue
            if occupied[y][x] or outside[y][x] or owner[y][x] != -1:
                continue

            owner[y][x] = component_id
            cells.add((x, y))
            queue.extend(((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)))

        return cells

    def _component_polygon(self, cells: set[tuple[int, int]], grid: _GridSpec) -> list[Point]:
        edges: list[tuple[tuple[int, int], tuple[int, int]]] = []
        for x, y in sorted(cells):
            if (x - 1, y) not in cells:
                edges.append(((x, y), (x, y + 1)))
            if (x, y + 1) not in cells:
                edges.append(((x, y + 1), (x + 1, y + 1)))
            if (x + 1, y) not in cells:
                edges.append(((x + 1, y + 1), (x + 1, y)))
            if (x, y - 1) not in cells:
                edges.append(((x + 1, y), (x, y)))

        if not edges:
            return []

        adjacency: dict[tuple[int, int], list[tuple[int, int]]] = defaultdict(list)
        edge_set = set(edges)
        for start, end in edges:
            adjacency[start].append(end)

        loops: list[list[tuple[int, int]]] = []
        while edge_set:
            start_edge = min(edge_set)
            start_vertex = start_edge[0]
            current_vertex = start_vertex
            loop = [current_vertex]

            while True:
                next_vertices = adjacency[current_vertex]
                if not next_vertices:
                    break

                next_vertex = next_vertices.pop()
                edge = (current_vertex, next_vertex)
                if edge not in edge_set:
                    current_vertex = next_vertex
                    if current_vertex == start_vertex:
                        break
                    continue

                edge_set.remove(edge)
                current_vertex = next_vertex
                if current_vertex == start_vertex:
                    break
                loop.append(current_vertex)

            if len(loop) >= 3:
                loops.append(loop)

        if not loops:
            return []

        best_loop = max(loops, key=lambda loop: abs(self._polygon_area(self._grid_points_to_world(loop, grid))))
        polygon = self._grid_points_to_world(best_loop, grid)
        polygon = self._simplify_polygon(polygon)
        if self._polygon_area(polygon) < 0.0:
            polygon.reverse()
        return polygon

    def _component_wall_ids(
        self,
        cells: set[tuple[int, int]],
        occupied: list[list[bool]],
        wall_cells: dict[tuple[int, int], set[str]],
        grid: _GridSpec,
    ) -> set[str]:
        wall_ids: set[str] = set()
        for x, y in cells:
            for nx, ny in ((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)):
                if nx < 0 or ny < 0 or nx >= grid.width or ny >= grid.height:
                    continue
                if not occupied[ny][nx]:
                    continue
                wall_ids.update(wall_cells.get((nx, ny), set()))
        return wall_ids

    def _opening_side_rooms(
        self,
        opening: Door | Window,
        host_wall: Wall | None,
        room_cell_lookup: dict[tuple[int, int], str],
        grid: _GridSpec,
    ) -> tuple[str | None, str | None]:
        midpoint = self._midpoint(opening.start, opening.end)
        direction = self._unit_vector(
            host_wall.start if host_wall is not None else opening.start,
            host_wall.end if host_wall is not None else opening.end,
        )
        if direction is None:
            return None, None

        normal = Point(x=-direction.y, y=direction.x)
        half_width = self._effective_wall_width(host_wall) / 2.0 if host_wall is not None else 0.0
        base_offset = max(half_width + grid.cell_size, grid.cell_size * 1.5)

        positive_room = self._find_room_at_offset(midpoint, normal, base_offset, room_cell_lookup, grid)
        negative_room = self._find_room_at_offset(midpoint, Point(x=-normal.x, y=-normal.y), base_offset, room_cell_lookup, grid)
        return positive_room, negative_room

    def _find_room_at_offset(
        self,
        midpoint: Point,
        direction: Point,
        base_offset: float,
        room_cell_lookup: dict[tuple[int, int], str],
        grid: _GridSpec,
    ) -> str | None:
        for multiplier in (1.0, 1.75, 2.5):
            sample = Point(
                x=midpoint.x + direction.x * base_offset * multiplier,
                y=midpoint.y + direction.y * base_offset * multiplier,
            )
            room_id = self._room_id_for_point(sample, room_cell_lookup, grid)
            if room_id is not None:
                return room_id
        return None

    def _room_id_for_point(
        self,
        point: Point,
        room_cell_lookup: dict[tuple[int, int], str],
        grid: _GridSpec,
    ) -> str | None:
        ix = int((point.x - grid.origin_x) / grid.cell_size)
        iy = int((point.y - grid.origin_y) / grid.cell_size)
        return room_cell_lookup.get((ix, iy))

    def _build_room_cell_lookup(
        self,
        room_components: dict[str, _RoomComponent],
    ) -> dict[tuple[int, int], str]:
        room_cell_lookup: dict[tuple[int, int], str] = {}
        for room_id, component in room_components.items():
            for cell in component.cells:
                room_cell_lookup[cell] = room_id
        return room_cell_lookup

    def _build_wall_indexes(
        self,
        walls: list[Wall],
    ) -> tuple[dict[str, Wall], dict[str, list[Wall]]]:
        wall_by_id = {wall.id: wall for wall in walls}
        walls_by_run: dict[str, list[Wall]] = defaultdict(list)
        for wall in walls:
            if wall.run_id is not None:
                walls_by_run[wall.run_id].append(wall)
        return wall_by_id, walls_by_run

    def _opening_room_binding(
        self,
        opening: Door | Window,
        host_wall: Wall | None,
        room_ids_in_order: list[str],
        room_cell_lookup: dict[tuple[int, int], str],
        grid: _GridSpec,
    ) -> tuple[str | None, str | None, tuple[str, ...]]:
        positive_room, negative_room = self._opening_side_rooms(
            opening,
            host_wall,
            room_cell_lookup,
            grid,
        )
        rooms = tuple(
            room_id for room_id in room_ids_in_order
            if room_id in {positive_room, negative_room}
        )
        return positive_room, negative_room, rooms

    def _resolve_host_wall(
        self,
        opening: Door | Window,
        wall_by_id: dict[str, Wall],
        walls_by_run: dict[str, list[Wall]],
    ) -> Wall | None:
        candidates: list[Wall] = []

        for wall_id in opening.support_wall_ids:
            wall = wall_by_id.get(wall_id)
            if wall is not None:
                candidates.append(wall)

        if opening.wall_id is not None:
            if opening.wall_id in wall_by_id:
                candidates.append(wall_by_id[opening.wall_id])
            candidates.extend(walls_by_run.get(opening.wall_id, []))

        if not candidates:
            return None

        midpoint = self._midpoint(opening.start, opening.end)
        unique_candidates = {(candidate.id, candidate.run_id): candidate for candidate in candidates}.values()
        return min(
            unique_candidates,
            key=lambda wall: (
                self._distance_point_to_segment(midpoint, wall.start, wall.end),
                -self._effective_wall_width(wall),
                -wall.length,
                wall.id,
            ),
        )

    def _effective_wall_width(self, wall: Wall | None) -> float:
        if wall is None:
            return 0.0
        return wall.width

    def _cells_near_segment(
        self,
        start: Point,
        end: Point,
        radius: float,
        grid: _GridSpec,
    ) -> list[tuple[int, int]]:
        min_x = min(start.x, end.x) - radius - grid.cell_size
        max_x = max(start.x, end.x) + radius + grid.cell_size
        min_y = min(start.y, end.y) - radius - grid.cell_size
        max_y = max(start.y, end.y) + radius + grid.cell_size

        start_x = max(0, int((min_x - grid.origin_x) / grid.cell_size))
        end_x = min(grid.width - 1, int((max_x - grid.origin_x) / grid.cell_size))
        start_y = max(0, int((min_y - grid.origin_y) / grid.cell_size))
        end_y = min(grid.height - 1, int((max_y - grid.origin_y) / grid.cell_size))

        cells: list[tuple[int, int]] = []
        for iy in range(start_y, end_y + 1):
            for ix in range(start_x, end_x + 1):
                center = self._cell_center(ix, iy, grid)
                if self._distance_point_to_segment(center, start, end) <= radius:
                    cells.append((ix, iy))
        return cells

    def _grid_points_to_world(self, vertices: list[tuple[int, int]], grid: _GridSpec) -> list[Point]:
        return [
            Point(
                x=round(grid.origin_x + vertex[0] * grid.cell_size, 6),
                y=round(grid.origin_y + vertex[1] * grid.cell_size, 6),
            )
            for vertex in vertices
        ]

    def _simplify_polygon(self, polygon: list[Point]) -> list[Point]:
        if len(polygon) < 3:
            return polygon

        simplified: list[Point] = []
        for point in polygon:
            if simplified and point.x == simplified[-1].x and point.y == simplified[-1].y:
                continue
            simplified.append(point)

        changed = True
        while changed and len(simplified) >= 3:
            changed = False
            reduced: list[Point] = []
            for index, point in enumerate(simplified):
                prev_point = simplified[index - 1]
                next_point = simplified[(index + 1) % len(simplified)]
                if self._is_collinear(prev_point, point, next_point):
                    changed = True
                    continue
                reduced.append(point)
            simplified = reduced

        return simplified

    def _min_component_cells(self, grid: _GridSpec, units: str | None) -> int:
        if units is None:
            return MIN_COMPONENT_CELLS

        factor_mm = UNITS_TO_MILLIMETERS.get(units.strip().lower())
        if factor_mm is None or factor_mm == 0.0:
            return MIN_COMPONENT_CELLS

        if self._is_suspicious_plan_scale(grid, units):
            return MIN_COMPONENT_CELLS

        cell_size_m = (grid.cell_size * factor_mm) / 1000.0
        if cell_size_m <= 0.0:
            return MIN_COMPONENT_CELLS

        min_cells_by_area = int(ceil(MIN_LABELED_ROOM_AREA_M2 / (cell_size_m * cell_size_m)))
        return max(MIN_COMPONENT_CELLS, min_cells_by_area)

    def _is_suspicious_plan_scale(self, grid: _GridSpec, units: str | None) -> bool:
        if units is None:
            return False

        factor_mm = UNITS_TO_MILLIMETERS.get(units.strip().lower())
        if factor_mm is None or factor_mm == 0.0:
            return False

        plan_area_m2 = self._area_to_square_meters(grid.span_x * grid.span_y, units)
        return plan_area_m2 < MIN_PLAUSIBLE_PLAN_AREA_M2

    def _cell_center(self, ix: int, iy: int, grid: _GridSpec) -> Point:
        return Point(
            x=grid.origin_x + (ix + 0.5) * grid.cell_size,
            y=grid.origin_y + (iy + 0.5) * grid.cell_size,
        )

    def _midpoint(self, start: Point, end: Point) -> Point:
        return Point(x=(start.x + end.x) / 2.0, y=(start.y + end.y) / 2.0)

    def _distance_point_to_segment(self, point: Point, start: Point, end: Point) -> float:
        dx = end.x - start.x
        dy = end.y - start.y
        if dx == 0.0 and dy == 0.0:
            return ((point.x - start.x) ** 2 + (point.y - start.y) ** 2) ** 0.5

        t = ((point.x - start.x) * dx + (point.y - start.y) * dy) / (dx * dx + dy * dy)
        clamped_t = max(0.0, min(1.0, t))
        closest_x = start.x + dx * clamped_t
        closest_y = start.y + dy * clamped_t
        return ((point.x - closest_x) ** 2 + (point.y - closest_y) ** 2) ** 0.5

    def _distance_between_points(self, left: Point, right: Point) -> float:
        return ((left.x - right.x) ** 2 + (left.y - right.y) ** 2) ** 0.5

    def _unit_vector(self, start: Point, end: Point) -> Point | None:
        dx = end.x - start.x
        dy = end.y - start.y
        length = (dx * dx + dy * dy) ** 0.5
        if length == 0.0:
            return None
        return Point(x=dx / length, y=dy / length)

    def _polygon_area(self, polygon: list[Point] | tuple[Point, ...]) -> float:
        if len(polygon) < 3:
            return 0.0

        doubled_area = 0.0
        for index, point in enumerate(polygon):
            next_point = polygon[(index + 1) % len(polygon)]
            doubled_area += point.x * next_point.y - next_point.x * point.y
        return doubled_area / 2.0

    def _is_collinear(self, a: Point, b: Point, c: Point) -> bool:
        return abs((b.x - a.x) * (c.y - a.y) - (b.y - a.y) * (c.x - a.x)) <= 1e-9

    def _area_to_square_meters(self, area_units: float, units: str | None) -> float:
        if units is None:
            return area_units

        factor_mm = UNITS_TO_MILLIMETERS.get(units.strip().lower())
        if factor_mm is None or factor_mm == 0.0:
            return area_units

        unit_to_meter = factor_mm / 1000.0
        return area_units * unit_to_meter * unit_to_meter
