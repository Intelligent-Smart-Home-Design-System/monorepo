
import typer
import asyncio
from pathlib import Path

app = typer.Typer()

@app.command()
def run(
    config_path: Path = typer.Option(
        Path("config.toml"), 
        "--config", "-c",
        help="Path to config file"
    ),
):
    """Run the quality-calculator service."""

    asyncio.run(_run(config_path))

@app.command()
def help(
    config_path: Path = typer.Option(
        Path("config.toml"), 
        "--config", "-c",
        help="Path to config file"
    ),
):
    """Quality calculator service help"""

    typer.echo("Help for quality-calculator")

async def _run(config_path: Path):
    typer.echo(f"Hello from quality-calculator service")

if __name__ == "__main__":
    app()
