import json
import outlines
import typer
import asyncio
from pathlib import Path
from extractor.config import Settings
import structlog
from openai import AsyncOpenAI, OpenAI
import os
import time

from extractor.worker.worker import Worker
from extractor.domain.models import ListingSnapshot
from extractor.adapters.outlines_extractor import OutlinesExtractor
from extractor.evaluation.evaluate import evaluate_listing
from extractor.evaluation.metrics import ModelMetrics
from extractor.adapters.postgres_repository import PostgresExtractionRepository
from extractor.domain.models import ExtractionSnapshot

from datetime import datetime

app = typer.Typer()

def make_client(settings: Settings) -> AsyncOpenAI:
    api_key = os.environ.get("YANDEX_CLOUD_API_KEY", "")
    if not api_key:
        raise ValueError("YANDEX_CLOUD_API_KEY env var not set")
    return AsyncOpenAI(
        api_key=settings.yandex_cloud.api_key,
        base_url="https://ai.api.cloud.yandex.net/v1",
        project=settings.yandex_cloud.folder,
        default_headers={"Authorization": f"Api-Key {api_key}"},
    )

def make_extractor(settings: Settings) -> OutlinesExtractor:
    outlines_model = outlines.from_openai(
        make_client(settings),
        f"gpt://{settings.yandex_cloud.folder}/{settings.yandex_cloud.llm_model}"
    )
    taxonomy = json.loads(Path(settings.taxonomy.path).read_text())
    return OutlinesExtractor(
        taxonomy=taxonomy,
        model=outlines_model,
        extraction=settings.extraction,
        temperature=settings.yandex_cloud.temperature
    )

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
    log = structlog.get_logger()
    
    repo = await PostgresExtractionRepository.create(settings.database, log)
    extractor = make_extractor(settings)

    try:
        worker = Worker(extractor=extractor, repository=repo, model=settings.yandex_cloud.llm_model)
        await worker.run()
    finally:
        await repo.close()


@app.command()
def run_sample(
    config_path: Path = typer.Option(Path("config.toml"), "--config", "-c"),
    parsed_listing_path: Path = typer.Option(Path("parsed_listing_lamp.json"), "--parsed-listing", "-i")
):
    """Run extraction on a single parsed listing and output result in stdout"""
    settings = Settings.from_toml(config_path)
    taxonomy = json.loads(Path(settings.taxonomy.path).read_text())
    listing = ListingSnapshot.model_validate_json(parsed_listing_path.read_text())
    
    typer.echo(f"Loaded listing: {listing.name}", err=True)
    typer.echo(f"Known device types: {', '.join(taxonomy.keys())}", err=True)

    extractor = make_extractor(settings)

    detected = asyncio.run(extractor.detect_device_type(listing))
    typer.echo(f"Detected type: {detected.type} (confidence: {detected.confidence:.2f})", err=True)

    result = asyncio.run(extractor.extract(listing, detected.type))

    print(result)


@app.command()
def run_evaluation(
    config_path: Path = typer.Option(Path("config.toml"), "--config", "-c"),
    listings_path: Path = typer.Option(Path("evaluation/listings.json"), "--listings", "-l"),
    output_dir: Path = typer.Option(Path("evaluation/results"), "--output", "-o"),
    model: str = typer.Option(None, "--model", "-m", help="Model to evaluate. Defaults to model from config."),
):
    """Run golden test set evaluation and write results to output dir."""
    asyncio.run(_run_evaluation(
        config_path=config_path,
        listings_path=listings_path,
        output_dir=output_dir,
        model=model
    ))

async def _run_evaluation(
    config_path: Path = typer.Option(Path("config.toml"), "--config", "-c"),
    listings_path: Path = typer.Option(Path("evaluation/listings.json"), "--listings", "-l"),
    output_dir: Path = typer.Option(Path("evaluation/results"), "--output", "-o"),
    model: str = typer.Option(None, "--model", "-m", help="Model to evaluate. Defaults to model from config."),
):
    settings = Settings.from_toml(config_path)
    test_cases = json.loads(listings_path.read_text())
    taxonomy = json.loads(Path(settings.taxonomy.path).read_text())
    output_dir.mkdir(parents=True, exist_ok=True)

    if model is not None:
        settings.yandex_cloud.llm_model = model # override
    model_name = settings.yandex_cloud.llm_model
    extractor = make_extractor(settings)

    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    all_summaries = []

    typer.echo(f"\nModel: {model_name} ({len(test_cases)} listings)")
    metrics = ModelMetrics(model_name=model_name)
    
    for i, case in enumerate(test_cases):
        result = await evaluate_listing(
            extractor=extractor,
            listing_id=i,
            listing_raw=case["listing"],
            ground_truth=case["ground_truth"]
        )
        time.sleep(1.0)
        metrics.listing_results.append(result)
        status = "OK" if result.type_correct else "FAIL"
        perfect = "[PERFECT]" if result.perfect else ""
        typer.echo(
            f"{status}{perfect} ({i+1}/{len(test_cases)}) "
            f"[{result.actual_type}] {result.listing_name[:55]}",
            err=False,
        )
        if result.error:
            typer.echo(f"  error: {result.error}", err=True)

    summary = metrics.compute_summary(taxonomy)
    all_summaries.append((model_name, summary))

    output = {
        "model": model_name,
        "summary": summary,
        "listings": [
            {
                "id": r.listing_id,
                "name": r.listing_name,
                "expected_type": r.expected_type,
                "actual_type": r.actual_type,
                "type_correct": r.type_correct,
                "perfect": r.perfect,
                "error": r.error,
                "expected_output": r.expected_output,
                "actual_output": r.actual_output,
            }
            for r in metrics.listing_results
        ],
    }

    safe_name = model_name.replace("/", "_")
    out_path = output_dir / f"{timestamp}_{safe_name}.json"
    out_path.write_text(json.dumps(output, ensure_ascii=False, indent=2))
    typer.echo(f"Results: {out_path}")
    typer.echo(f"Type accuracy: {summary['type_accuracy']:.0%}")
    typer.echo(f"Perfect extraction: {summary['perfect_extraction_rate']:.0%}")


if __name__ == "__main__":
    app()
