from __future__ import annotations

import asyncio
import json
import time
from dataclasses import dataclass
from pathlib import Path

import structlog
from opentelemetry import trace
from temporalio import activity

from internal.pipeline import parse_floor_path
from internal.worker.contracts import ParseFloorInput, ParseFloorOutput
from internal.worker.observability import Metrics


@dataclass
class ActivityState:
    semaphore: asyncio.Semaphore
    metrics: Metrics
    service_name: str


_state: ActivityState | None = None


def init_activity_state(state: ActivityState) -> None:
    global _state
    _state = state


def _get_state() -> ActivityState:
    if _state is None:
        raise RuntimeError("Activity state is not initialized")
    return _state


def _logger_for_activity(request_id: str) -> structlog.BoundLogger:
    info = activity.info()
    return structlog.get_logger().bind(
        request_id=request_id,
        workflow_id=info.workflow_id,
        workflow_run_id=info.workflow_run_id,
        activity_id=info.activity_id,
        activity_type=info.activity_type,
        task_queue=info.task_queue,
    )


@activity.defn(name="floor_parser.parse_floor")
async def parse_floor_activity(payload: ParseFloorInput) -> ParseFloorOutput:
    state = _get_state()
    logger = _logger_for_activity(payload.request_id)
    tracer = trace.get_tracer(state.service_name)

    start = time.perf_counter()
    state.metrics.concurrent_runs.inc()
    try:
        async with state.semaphore:
            with tracer.start_as_current_span("floor_parser.parse_floor") as span:
                span.set_attribute("request.id", payload.request_id)
                span.set_attribute("source.path", payload.source_path)
                span_context = span.get_span_context()
                logger = logger.bind(
                    trace_id=f"{span_context.trace_id:032x}",
                    span_id=f"{span_context.span_id:016x}",
                )

                logger.info("floor-parser activity started", source_path=payload.source_path, output_path=payload.output_path)
                activity.heartbeat("started")

                source_path = Path(payload.source_path)
                output_path = Path(payload.output_path)
                output_path.parent.mkdir(parents=True, exist_ok=True)

                floor_json = parse_floor_path(source_path, source_name=source_path.name)
                output_path.write_text(json.dumps(floor_json, ensure_ascii=False, indent=2), encoding="utf-8")

                activity.heartbeat("completed")
                result = ParseFloorOutput(
                    request_id=payload.request_id,
                    output_path=str(output_path),
                    wall_count=len(floor_json.get("walls", [])),
                    door_count=len(floor_json.get("doors", [])),
                    window_count=len(floor_json.get("windows", [])),
                    warning_count=len(floor_json.get("warnings", [])),
                )
                duration = time.perf_counter() - start
                state.metrics.runs_total.labels(status="success").inc()
                state.metrics.duration_seconds.observe(duration)
                logger.info(
                    "floor-parser activity completed",
                    duration_seconds=duration,
                    wall_count=result.wall_count,
                    door_count=result.door_count,
                    window_count=result.window_count,
                )
                return result
    except Exception:
        duration = time.perf_counter() - start
        state.metrics.runs_total.labels(status="failure").inc()
        state.metrics.duration_seconds.observe(duration)
        logger.exception("floor-parser activity failed", duration_seconds=duration)
        raise
    finally:
        state.metrics.concurrent_runs.dec()
