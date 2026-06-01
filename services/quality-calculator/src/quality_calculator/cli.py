from __future__ import annotations

import json
import asyncio
from datetime import datetime
from pathlib import Path

import structlog
import typer

from quality_calculator.config import Settings
from quality_calculator.evaluator import QualityEvaluator
from quality_calculator.evaluation.evaluate import run_strategy

app = typer.Typer()

DEVICE_TYPES_PATH = Path("../../shared/schemas/devices/device_types.json")
TRAITS_PATH = Path("config/evaluation_traits.json")
STRATEGIES_DIR = Path("config/strategies")
SPEC_STRATEGIES_DIR = Path("config/strategies/specs")
RUBRIC_PATH = Path("config/ground_truth_rubric.json")


def _load_evaluator(strategy_name: str) -> tuple[QualityEvaluator, dict]:
    tech = json.loads(DEVICE_TYPES_PATH.read_text(encoding="utf-8"))
    traits = json.loads(TRAITS_PATH.read_text(encoding="utf-8"))
    cfg = json.loads((STRATEGIES_DIR / f"{strategy_name}.json").read_text(encoding="utf-8"))
    w = cfg["weights"]
    evaluator = QualityEvaluator(
        tech_schema=tech,
        eval_strategy=traits,
        weights=(w["reputation"], w["specs"], w["ecosystem"]),
        reputation_mode=cfg.get("reputation_mode", "bayesian"),
    )
    return evaluator, cfg


def _setup_logging(settings: Settings) -> None:
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
    config_path: Path = typer.Option(Path("config.toml"), "--config", "-c", help="Path to config file"),
    strategy: str = typer.Option(None, "--strategy", "-s", help="Переопределить стратегию из config.toml"),
):
    """Прочитать устройства золотого слоя, посчитать quality и записать в devices.quality."""
    settings = Settings.from_toml(config_path)
    _setup_logging(settings)
    asyncio.run(_run(settings, strategy))


async def _run(settings: Settings, strategy_override: str | None):
    # Импортируем здесь, чтобы команды эвалюации работали без установленного asyncpg.
    from quality_calculator.adapters.postgres_repository import PostgresQualityRepository
    from quality_calculator.worker.worker import Worker

    log = structlog.get_logger()
    strategy_name = strategy_override or settings.scoring.strategy
    evaluator, _cfg = _load_evaluator(strategy_name)

    repo = await PostgresQualityRepository.create(settings.database, log)
    try:
        worker = Worker(
            evaluator=evaluator,
            repository=repo,
            batch_size=settings.scoring.batch_size,
            recompute_all=settings.scoring.recompute_all,
        )
        stats = await worker.run()
        typer.echo(
            f"strategy='{strategy_name}'  обработано={stats['total']}  "
            f"записано quality={stats['scored']}  без сигналов={stats['skipped']}"
        )
    finally:
        await repo.close()


@app.command()
def run_evaluation(
    catalog_path: Path = typer.Option(Path("evaluation/catalog.json"), "--catalog", "-i"),
    output_dir: Path = typer.Option(Path("evaluation/results"), "--output", "-o"),
    strategy: str = typer.Option(
        None, "--strategy", "-s",
        help="Одна стратегия. Если не задана — прогоняются все из config/strategies.",
    ),
    min_n: int = typer.Option(10, "--min-n", help="Минимум устройств в категории для агрегатов."),
):
    """Оценить качество модели(-ей) на тестовом каталоге и записать метрики в JSON."""
    catalog = json.loads(catalog_path.read_text(encoding="utf-8"))
    output_dir.mkdir(parents=True, exist_ok=True)

    if strategy:
        names = [strategy]
    else:
        names = sorted(p.stem for p in STRATEGIES_DIR.glob("*.json"))

    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    table_rows = []
    full_output = {"timestamp": timestamp, "catalog": str(catalog_path), "strategies": []}

    for name in names:
        evaluator, cfg = _load_evaluator(name)
        metrics, per_device = run_strategy(
            catalog, evaluator, name, cfg["weights"], cfg.get("reputation_mode", "bayesian")
        )
        summary = metrics.summary(min_n=min_n)
        table_rows.append(summary)
        full_output["strategies"].append(
            {
                "summary": summary,
                "per_category": [c.__dict__ for c in metrics.per_category],
                "description": cfg.get("description", ""),
            }
        )
        typer.echo(
            f"{name:<18} spearman={_fmt(summary['weighted_spearman'])} "
            f"kendall={_fmt(summary['weighted_kendall'])} "
            f"prec@10={_fmt(summary['weighted_precision_at_10'])} "
            f"specsCov={_fmt(summary['weighted_specs_coverage'])} "
            f"scored={summary['scored_devices']}"
        )

    out_path = output_dir / f"{timestamp}_evaluation.json"
    out_path.write_text(json.dumps(full_output, ensure_ascii=False, indent=2), encoding="utf-8")
    typer.echo(f"\nResults: {out_path}")


def _fmt(x) -> str:
    return f"{x:.3f}" if isinstance(x, (int, float)) else "  -  "


def _list_spec_strategies() -> list[str]:
    return sorted(p.stem for p in SPEC_STRATEGIES_DIR.glob("*.json"))


def _select_spec_strategy_interactive(names: list[str]) -> list[str]:
    """Интерактивное меню в терминале: вывести список и дать выбрать одну стратегию или все."""
    typer.echo("Выберите спек-стратегию для прогона против эталона:\n")
    for i, name in enumerate(names, 1):
        typer.echo(f"  [{i}] {name}")
    typer.echo(f"  [{len(names) + 1}] <все>\n")
    while True:
        raw = typer.prompt("Номер").strip()
        if not raw.isdigit():
            typer.echo("Введите номер из списка.")
            continue
        idx = int(raw)
        if 1 <= idx <= len(names):
            return [names[idx - 1]]
        if idx == len(names) + 1:
            return names
        typer.echo("Номер вне диапазона.")


@app.command("tune-specs")
def tune_specs(
    catalog_path: Path = typer.Option(Path("evaluation/catalog.json"), "--catalog", "-i"),
    output_dir: Path = typer.Option(Path("evaluation/results"), "--output", "-o"),
    strategy: str = typer.Option(
        None, "--strategy", "-s",
        help="Спек-стратегия из config/strategies/specs. Если не задана — интерактивный выбор.",
    ),
    min_n: int = typer.Option(10, "--min-n", help="Минимум размеченных устройств в категории для F1."),
):
    """
    Тестирует спек-стратегии (варианты весов внутри N(S)) против эталона
    bad/good/excellent и печатает таблицу лидеров по weighted macro-F1.
    """
    from quality_calculator.evaluation.spec_tuning import evaluate_spec_strategy

    tech = json.loads(DEVICE_TYPES_PATH.read_text(encoding="utf-8"))
    rubric = json.loads(RUBRIC_PATH.read_text(encoding="utf-8"))
    catalog = json.loads(catalog_path.read_text(encoding="utf-8"))

    available = _list_spec_strategies()
    if not available:
        typer.echo("Нет спек-стратегий в config/strategies/specs.")
        raise typer.Exit(code=1)

    if strategy:
        names = [strategy]
    else:
        import sys
        # В неинтерактивном окружении (CI, пайп) — берём все, чтобы не зависнуть на prompt.
        names = _select_spec_strategy_interactive(available) if sys.stdin.isatty() else available

    results = []
    for name in names:
        spec_strategy = json.loads((SPEC_STRATEGIES_DIR / f"{name}.json").read_text(encoding="utf-8"))
        res = evaluate_spec_strategy(catalog, tech, spec_strategy, rubric, name, min_n=min_n)
        results.append(res)

    results.sort(key=lambda r: (r["weighted_macro_f1"] is not None, r["weighted_macro_f1"]), reverse=True)

    typer.echo("\n=== Лидерборд спек-стратегий (по эталону, метрика — N(S)) ===")
    for r in results:
        typer.echo(
            f"  {r['strategy']:<20} macroF1={_fmt(r['weighted_macro_f1'])} "
            f"размечено={r['labeled_devices']:<4} оценено={r['evaluated_devices']}"
        )
    best = results[0]
    typer.echo(f"\nЛучшая спек-стратегия: {best['strategy']} (macroF1={_fmt(best['weighted_macro_f1'])})")

    output_dir.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    out_path = output_dir / f"{timestamp}_spec_tuning.json"
    out_path.write_text(
        json.dumps({"timestamp": timestamp, "best": best["strategy"], "results": results},
                   ensure_ascii=False, indent=2),
        encoding="utf-8",
    )
    typer.echo(f"Подробности: {out_path}")


if __name__ == "__main__":
    app()
