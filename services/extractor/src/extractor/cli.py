import json
import outlines
import transformers
import typer
import asyncio
from pathlib import Path
from extractor.config import Settings
import structlog
import ollama

from extractor.domain.models import ListingSnapshot
from extractor.adapters.outlines_extractor import OutlinesExtractor

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


@app.command()
def run_sample(
    config_path: Path = typer.Option(Path("config.toml"), "--config", "-c"),
    parsed_listing_path: Path = typer.Option(Path("parsed_listing.json"), "--parsed-listing", "-i")
):
    """Run extraction on a single parsed listing and output result in stdout"""
    try:
        settings = Settings.from_toml(config_path)
        typer.echo(f"✓ Config valid: {config_path}")
        typer.echo(f"  Database: {settings.database.host}:{settings.database.port}")
    except Exception as e:
        typer.echo(f"✗ Invalid config: {e}", err=True)
        raise typer.Exit(1)

    try:
        taxonomy = json.loads(Path(settings.taxonomy.path).read_text())
    except Exception as e:
        typer.echo(f"✗ Failed to load taxonomy: {e}", err=True)
        raise typer.Exit(1)

    try:
        listing = ListingSnapshot.model_validate_json(parsed_listing_path.read_text())
    except Exception as e:
        typer.echo(f"✗ Failed to load parsed listing: {e}", err=True)
        raise typer.Exit(1)
    
    typer.echo(f"Loaded listing: {listing.name}", err=True)
    typer.echo(f"Known device types: {', '.join(taxonomy.keys())}", err=True)

    # MODEL_NAME = "mistralai/Mistral-7B-Instruct-v0.3"

    # model = outlines.from_transformers(
    #    transformers.AutoModelForCausalLM.from_pretrained(MODEL_NAME),
    #    transformers.AutoTokenizer.from_pretrained(MODEL_NAME),
    # )
    model = outlines.from_ollama(ollama.Client(), "mistral")

    extractor = OutlinesExtractor(taxonomy=taxonomy, model=model)

    detected = asyncio.run(extractor.detect_device_type(listing))
    typer.echo(f"Detected type: {detected.type} (confidence: {detected.confidence:.2f})", err=True)

    result = asyncio.run(extractor.extract(listing, detected.type))

    print(result)


if __name__ == "__main__":
    app()
