from __future__ import annotations

from typing import Any

from services.parser.internal.entities.geometry import Point
from services.parser.internal.entities.raw_plan import RawLine, RawPlan, RawPolyline
from services.parser.internal.readers.dxf.reader import DxfReadResult


class DxfExtractor:
    def extract(self, read_result: DxfReadResult) -> RawPlan:
        entities = []
        for entity in read_result.modelspace:
            dxf_type = entity.dxftype()

            if dxf_type == "LINE":
                entities.append(self._extract_line(entity))
                continue

            if dxf_type == "LWPOLYLINE":
                entities.append(self._extract_lwpolyline(entity))

        return RawPlan(metadata=read_result.metadata, entities=entities)

    def _extract_line(self, entity: Any) -> RawLine:
        start = Point(x=float(entity.dxf.start.x), y=float(entity.dxf.start.y))
        end = Point(x=float(entity.dxf.end.x), y=float(entity.dxf.end.y))

        return RawLine(
            id=entity.dxf.handle,
            layer=entity.dxf.layer,
            start=start,
            end=end
        )

    def _extract_lwpolyline(self, entity: Any) -> RawPolyline:
        points = [
            Point(x=float(point[0]), y=float(point[1]))
            for point in entity.get_points()
        ]
        return RawPolyline(
            id=entity.dxf.handle,
            layer=entity.dxf.layer,
            points=points,
            closed=bool(entity.closed)
        )
