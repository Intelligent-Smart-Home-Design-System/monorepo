from __future__ import annotations

import uuid

import structlog
from fastapi import APIRouter, File, HTTPException, Request, UploadFile, status

from internal.pipeline import parse_floor

log = structlog.get_logger("floor-parser.api")
router = APIRouter()


@router.get("/health")
def healthcheck() -> dict[str, str]:
    return {"status": "ok"}


@router.post("/parse")
async def parse_dxf(request: Request, file: UploadFile = File(...)) -> dict[str, object]:
    request_id = request.headers.get("X-Request-ID") or str(uuid.uuid4())
    log.info("parse request received", request_id=request_id, filename=file.filename)

    if not file.filename:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="File name is required.",
        )

    try:
        result = await parse_floor(file)
        log.info("parse completed", request_id=request_id, success=True)
        return result
    except ValueError as exc:
        log.warning("parse validation error", request_id=request_id, error=str(exc))
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(exc),
        ) from exc
    except RuntimeError as exc:
        log.error("parse runtime error", request_id=request_id, error=str(exc))
        raise HTTPException(
            status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
            detail=str(exc),
        ) from exc
