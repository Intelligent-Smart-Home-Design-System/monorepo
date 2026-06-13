from __future__ import annotations

import os

import structlog
import uvicorn
from fastapi import FastAPI

from internal.api.routes import router as api_router
from internal.logging_config import setup_logging

log = structlog.get_logger("floor-parser")


def create_app() -> FastAPI:
    app = FastAPI(title="floor-parser", version="0.1.0")
    app.include_router(api_router)
    return app


def main() -> None:
    setup_logging(service="floor-parser")
    app = create_app()
    host = os.getenv("PARSER_HOST", "0.0.0.0")
    port = int(os.getenv("PARSER_PORT", "8080"))

    log.info("starting http server", host=host, port=port)
    uvicorn.run(app, host=host, port=port)


if __name__ == "__main__":
    main()
