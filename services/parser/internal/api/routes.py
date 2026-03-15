from __future__ import annotations

from fastapi import APIRouter, File, HTTPException, UploadFile, status

from services.parser.internal.pipeline import parse_floor


router = APIRouter()


@router.get("/health")
def healthcheck() -> dict[str, str]:
    return {"status": "ok"}


@router.post("/parse")
async def parse_dxf(file: UploadFile = File(...)) -> dict[str, object]:
    if not file.filename:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="File name is required.",
        )

    if not file.filename.lower().endswith(".dxf"):
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Only DXF files are supported.",
        )

    return await parse_floor(file)

    
