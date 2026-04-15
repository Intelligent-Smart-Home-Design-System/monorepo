from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any

import ezdxf

from services.floor_parser.internal.entities.raw_plan import SourceFormat, SourceMetadata


INSUNITS_TO_NAME: dict[int, str] = {
    1: "inches",
    2: "feet",
    4: "mm",
    5: "cm",
    6: "m",
}


@dataclass(frozen=True)
class DxfReadResult:
    source_file: str
    document: Any
    modelspace: Any
    metadata: SourceMetadata


class DxfReader:
    def read_path(self, path: str | Path) -> DxfReadResult:
        source_path = Path(path)
        document = ezdxf.readfile(str(source_path))
        modelspace = document.modelspace()

        metadata = SourceMetadata(
            source_format=SourceFormat.DXF,
            source_file=source_path.name,
            units=self._extract_units(document),
            entity_count=len(modelspace),
        )

        return DxfReadResult(
            source_file=source_path.name,
            document=document,
            modelspace=modelspace,
            metadata=metadata
        )

    def _extract_units(self, document):
        insunits = document.header.get("$INSUNITS", 0)
        return INSUNITS_TO_NAME.get(insunits)
