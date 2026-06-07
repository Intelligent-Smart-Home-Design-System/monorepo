from __future__ import annotations

import re
from typing import Any

from internal.entities.geometry import Point
from internal.entities.raw_plan import RawArc, RawInsert, RawLine, RawPlan, RawPolyline, RawText
from internal.readers.dxf.reader import DxfReadResult


class DxfExtractor:
    _unicode_escape_pattern = re.compile(r"\\U\+([0-9A-Fa-f]{4})")

    def extract(self, read_result: DxfReadResult) -> RawPlan:
        entities = []
        for entity in read_result.modelspace:
            entities.extend(self._extract_entity(entity))

        return RawPlan(metadata=read_result.metadata, entities=entities)

    def _extract_entity(
        self,
        entity: Any,
        *,
        inherited_layer: str | None = None,
        source_insert_id: str | None = None,
        source_block_name: str | None = None,
        id_prefix: str | None = None,
    ) -> list[RawLine | RawPolyline | RawArc | RawText | RawInsert]:
        dxf_type = entity.dxftype()
        entity_id = self._entity_id(entity, id_prefix)
        layer = self._resolve_layer(entity, inherited_layer)

        if dxf_type == "LINE":
            return [
                self._extract_line(
                    entity,
                    entity_id=entity_id,
                    layer=layer,
                    source_insert_id=source_insert_id,
                    source_block_name=source_block_name,
                )
            ]

        if dxf_type == "LWPOLYLINE":
            return [
                self._extract_lwpolyline(
                    entity,
                    entity_id=entity_id,
                    layer=layer,
                    source_insert_id=source_insert_id,
                    source_block_name=source_block_name,
                )
            ]

        if dxf_type == "ARC":
            return [
                self._extract_arc(
                    entity,
                    entity_id=entity_id,
                    layer=layer,
                    source_insert_id=source_insert_id,
                    source_block_name=source_block_name,
                )
            ]

        if dxf_type == "TEXT":
            return [
                self._extract_text(
                    entity,
                    entity_id=entity_id,
                    layer=layer,
                    source_insert_id=source_insert_id,
                    source_block_name=source_block_name,
                )
            ]

        if dxf_type == "MTEXT":
            return [
                self._extract_mtext(
                    entity,
                    entity_id=entity_id,
                    layer=layer,
                    source_insert_id=source_insert_id,
                    source_block_name=source_block_name,
                )
            ]

        if dxf_type == "INSERT":
            return self._extract_insert(
                entity,
                entity_id=entity_id,
                layer=layer,
                source_insert_id=source_insert_id,
                source_block_name=source_block_name,
            )

        return []

    def _extract_line(
        self,
        entity: Any,
        *,
        entity_id: str,
        layer: str,
        source_insert_id: str | None = None,
        source_block_name: str | None = None,
    ) -> RawLine:
        start = Point(x=float(entity.dxf.start.x), y=float(entity.dxf.start.y))
        end = Point(x=float(entity.dxf.end.x), y=float(entity.dxf.end.y))

        return RawLine(
            id=entity_id,
            layer=layer,
            start=start,
            end=end,
            source_insert_id=source_insert_id,
            source_block_name=source_block_name,
        )

    def _extract_lwpolyline(
        self,
        entity: Any,
        *,
        entity_id: str,
        layer: str,
        source_insert_id: str | None = None,
        source_block_name: str | None = None,
    ) -> RawPolyline:
        points = [
            Point(x=float(point[0]), y=float(point[1]))
            for point in entity.get_points()
        ]
        return RawPolyline(
            id=entity_id,
            layer=layer,
            points=points,
            closed=bool(entity.closed),
            source_insert_id=source_insert_id,
            source_block_name=source_block_name,
        )

    def _extract_arc(
        self,
        entity: Any,
        *,
        entity_id: str,
        layer: str,
        source_insert_id: str | None = None,
        source_block_name: str | None = None,
    ) -> RawArc:
        center = Point(x=float(entity.dxf.center.x), y=float(entity.dxf.center.y))

        return RawArc(
            id=entity_id,
            layer=layer,
            center=center,
            radius=float(entity.dxf.radius),
            start_angle=float(entity.dxf.start_angle),
            end_angle=float(entity.dxf.end_angle),
            source_insert_id=source_insert_id,
            source_block_name=source_block_name,
        )

    def _extract_text(
        self,
        entity: Any,
        *,
        entity_id: str,
        layer: str,
        source_insert_id: str | None = None,
        source_block_name: str | None = None,
    ) -> RawText:
        insert = Point(x=float(entity.dxf.insert.x), y=float(entity.dxf.insert.y))

        return RawText(
            id=entity_id,
            layer=layer,
            text=self._decode_text(str(entity.dxf.text)),
            insert=insert,
            is_multiline=False,
            source_insert_id=source_insert_id,
            source_block_name=source_block_name,
        )

    def _extract_mtext(
        self,
        entity: Any,
        *,
        entity_id: str,
        layer: str,
        source_insert_id: str | None = None,
        source_block_name: str | None = None,
    ) -> RawText:
        insert = Point(x=float(entity.dxf.insert.x), y=float(entity.dxf.insert.y))

        return RawText(
            id=entity_id,
            layer=layer,
            text=self._decode_text(str(entity.plain_text())),
            insert=insert,
            is_multiline=True,
            source_insert_id=source_insert_id,
            source_block_name=source_block_name,
        )

    def _extract_insert(
        self,
        entity: Any,
        *,
        entity_id: str,
        layer: str,
        source_insert_id: str | None = None,
        source_block_name: str | None = None,
    ) -> list[RawLine | RawPolyline | RawArc | RawText | RawInsert]:
        insert = Point(x=float(entity.dxf.insert.x), y=float(entity.dxf.insert.y))

        block_name = str(entity.dxf.name)
        root_insert_id = source_insert_id or entity_id
        root_block_name = source_block_name or block_name

        extracted_entities: list[RawLine | RawPolyline | RawArc | RawText | RawInsert] = [
            RawInsert(
                id=entity_id,
                layer=layer,
                block_name=block_name,
                insert=insert,
                rotation=float(entity.dxf.rotation) if entity.dxf.hasattr("rotation") else None,
                source_insert_id=source_insert_id,
                source_block_name=source_block_name,
            )
        ]

        for index, child in enumerate(entity.virtual_entities(), start=1):
            extracted_entities.extend(
                self._extract_entity(
                    child,
                    inherited_layer=layer,
                    source_insert_id=root_insert_id,
                    source_block_name=root_block_name,
                    id_prefix=f"{entity_id}@{index}",
                )
            )

        return extracted_entities

    def _entity_id(self, entity: Any, id_prefix: str | None) -> str:
        if entity.dxf.hasattr("handle") and entity.dxf.handle is not None:
            return str(entity.dxf.handle)
        if id_prefix is not None:
            return id_prefix
        return f"virtual-{id(entity)}"

    def _resolve_layer(self, entity: Any, inherited_layer: str | None) -> str:
        layer = str(entity.dxf.layer) if entity.dxf.hasattr("layer") else "0"
        if inherited_layer is not None and layer == "0":
            return inherited_layer
        return layer

    def _decode_text(self, value: str) -> str:
        return self._unicode_escape_pattern.sub(
            lambda match: chr(int(match.group(1), 16)),
            value,
        )
