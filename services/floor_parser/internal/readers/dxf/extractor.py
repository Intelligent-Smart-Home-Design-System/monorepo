from __future__ import annotations

from typing import Any

from services.floor_parser.internal.entities.geometry import Point
from services.floor_parser.internal.entities.raw_plan import RawArc, RawInsert, RawLine, RawPlan, RawPolyline, RawText
from services.floor_parser.internal.readers.dxf.reader import DxfReadResult


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
                continue

            if dxf_type == "ARC":
                entities.append(self._extract_arc(entity))
                continue

            if dxf_type == "TEXT":
                entities.append(self._extract_text(entity))
                continue

            if dxf_type == "MTEXT":
                entities.append(self._extract_mtext(entity))
                continue

            if dxf_type == "INSERT":
                entities.append(self._extract_insert(entity))

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

    def _extract_arc(self, entity: Any) -> RawArc:
        center = Point(x=float(entity.dxf.center.x), y=float(entity.dxf.center.y))

        return RawArc(
            id=entity.dxf.handle,
            layer=entity.dxf.layer,
            center=center,
            radius=float(entity.dxf.radius),
            start_angle=float(entity.dxf.start_angle),
            end_angle=float(entity.dxf.end_angle),
        )

    def _extract_text(self, entity: Any) -> RawText:
        insert = Point(x=float(entity.dxf.insert.x), y=float(entity.dxf.insert.y))

        return RawText(
            id=entity.dxf.handle,
            layer=entity.dxf.layer,
            text=str(entity.dxf.text),
            insert=insert,
            is_multiline=False,
        )

    def _extract_mtext(self, entity: Any) -> RawText:
        insert = Point(x=float(entity.dxf.insert.x), y=float(entity.dxf.insert.y))

        return RawText(
            id=entity.dxf.handle,
            layer=entity.dxf.layer,
            text=str(entity.plain_text()),
            insert=insert,
            is_multiline=True,
        )

    def _extract_insert(self, entity: Any) -> RawInsert:
        insert = Point(x=float(entity.dxf.insert.x), y=float(entity.dxf.insert.y))

        return RawInsert(
            id=entity.dxf.handle,
            layer=entity.dxf.layer,
            block_name=str(entity.dxf.name),
            insert=insert,
            rotation=float(entity.dxf.rotation) if entity.dxf.hasattr("rotation") else None,
        )
