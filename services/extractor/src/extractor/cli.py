# src/extractor/cli.py
import typer
import asyncio
from pathlib import Path
from extractor.config import Settings
import structlog

app = typer.Typer()


def setup_logging(settings: Settings):
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer()
            if settings.logging.format == "json"
            else structlog.dev.ConsoleRenderer(),
        ],
    )


@app.command()
def run(
    config_path: Path = typer.Option(
        Path("config.toml"), 
        "--config", "-c",
        help="Path to config file"
    ),
):
    """Run the extraction service."""
    settings = Settings.from_toml(config_path)
    setup_logging(settings)
    
    asyncio.run(_run(settings))


async def _run(settings: Settings):
    # Initialize adapters
    
    return


@app.command()
def validate_config(
    config_path: Path = typer.Option(Path("config.toml"), "--config", "-c"),
):
    """Validate config file."""
    try:
        settings = Settings.from_toml(config_path)
        typer.echo(f"✓ Config valid: {config_path}")
        typer.echo(f"  Database: {settings.database.host}:{settings.database.port}")
    except Exception as e:
        typer.echo(f"✗ Invalid config: {e}", err=True)
        raise typer.Exit(1)


if __name__ == "__main__":
    app()
