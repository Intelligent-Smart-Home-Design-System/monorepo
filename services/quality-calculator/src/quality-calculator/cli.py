from __future__ import annotations

import asyncio
import logging
import os
from pathlib import Path

import structlog
import typer


def setup_logging() -> None:
    log_format = os.getenv("LOG_FORMAT", "json")
    log_level = os.getenv("LOG_LEVEL", "INFO")

    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.stdlib.add_log_level,
            structlog.stdlib.add_logger_name,
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.UnicodeDecoder(),
            structlog.processors.add_log_args,
            structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
        ],
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )

    handler: logging.Handler
    if log_format == "json":
        formatter = structlog.stdlib.ProcessorFormatter(
            processor=structlog.processors.JSONRenderer(),
            foreign_pre_chain=structlog.get_config()["processors"],
        )
        handler = logging.StreamHandler()
        handler.setFormatter(formatter)
    else:
        formatter = structlog.stdlib.ProcessorFormatter(
            processor=structlog.dev.ConsoleRenderer(),
            foreign_pre_chain=structlog.get_config()["processors"],
        )
        handler = logging.StreamHandler()
        handler.setFormatter(formatter)

    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(getattr(logging, log_level.upper(), logging.INFO))


app = typer.Typer()
log = structlog.get_logger("quality-calculator")


@app.command()
def run(
    config_path: Path = typer.Option(
        Path("config.toml"),
        "--config", "-c",
        help="Path to config file",
    ),
) -> None:
    """Run the quality-calculator service."""
    asyncio.run(_run(config_path))


@app.command(name="help-cmd")
def help_cmd() -> None:
    """Quality calculator service help."""
    typer.echo("Usage: quality-calculator run [-c config.toml]")


async def _run(config_path: Path) -> None:
    setup_logging()
    log.info("starting quality-calculator", config_path=str(config_path))
    log.info("quality-calculator finished")
