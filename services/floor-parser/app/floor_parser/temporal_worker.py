from __future__ import annotations

import asyncio
import os
from typing import Any

import structlog
from prometheus_client import Counter, start_http_server
from temporalio import activity
from temporalio.client import Client
from temporalio.worker import Worker

from internal.logging_config import setup_logging

log = structlog.get_logger("floor-parser-worker")
PARSED_TOTAL = Counter("floor_parser_activity_total", "Total parsed floor JSON requests", ["status"])


@activity.defn(name="parse_floor_json")
async def parse_floor_json(inp: dict[str, Any]) -> dict[str, Any]:
    request_id = inp.get("request_id", "unknown")
    log.info("floor plan json accepted", request_id=request_id)

    floor_plan = inp.get("floor_plan")
    if not isinstance(floor_plan, dict):
        PARSED_TOTAL.labels(status="failure").inc()
        log.error("invalid floor_plan type", request_id=request_id, floor_plan_type=type(floor_plan).__name__)
        raise ValueError("floor_plan must be an object")

    result = dict(floor_plan)
    result.setdefault("schema_version", "1.0")
    result.setdefault("meta", {})
    result.setdefault("walls", [])
    result.setdefault("doors", [])
    result.setdefault("windows", [])
    result.setdefault("rooms", [])
    result.setdefault("warnings", [])

    PARSED_TOTAL.labels(status="success").inc()
    log.info("floor plan parsed successfully", request_id=request_id, rooms=len(result["rooms"]))
    return {"floor_plan": result}


async def main() -> None:
    setup_logging(service="floor-parser-worker")
    temporal_address = os.getenv("TEMPORAL_ADDRESS", "localhost:7233")
    namespace = os.getenv("TEMPORAL_NAMESPACE", "default")
    task_queue = os.getenv("TEMPORAL_TASK_QUEUE", "floor-parser")
    metrics_port = int(os.getenv("METRICS_PORT", "2113"))

    start_http_server(metrics_port)
    client = await Client.connect(temporal_address, namespace=namespace)
    log.info("worker started", task_queue=task_queue, temporal_address=temporal_address, namespace=namespace)

    async with Worker(client, task_queue=task_queue, workflows=[], activities=[parse_floor_json]):
        await asyncio.Event().wait()


if __name__ == "__main__":
    asyncio.run(main())
