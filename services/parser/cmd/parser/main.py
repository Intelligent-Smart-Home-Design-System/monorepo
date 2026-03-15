from __future__ import annotations

import os

import uvicorn
from fastapi import FastAPI

from services.parser.internal.api.routes import router as api_router


def create_app() -> FastAPI:
    app = FastAPI(title="parser-service", version="0.1.0")
    app.include_router(api_router)

    return app


def main() -> None:
    app = create_app()
    host = os.getenv("PARSER_HOST", "0.0.0.0")
    port = int(os.getenv("PARSER_PORT", "8080"))

    uvicorn.run(app, host=host, port=port)


if __name__ == "__main__":
    main()
