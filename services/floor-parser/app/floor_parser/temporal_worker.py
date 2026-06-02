from __future__ import annotations

import asyncio
import logging
import os
from typing import Any

from prometheus_client import Counter, start_http_server
from temporalio import activity
from temporalio.client import Client
from temporalio.worker import Worker

log = logging.getLogger("floor-parser-worker")
PARSED_TOTAL = Counter("floor_parser_activity_total", "Total parsed floor JSON requests", ["status"])


@activity.defn(name="parse_floor_json")
async def parse_floor_json(inp: dict[str, Any]) -> dict[str, Any]:
    floor_plan = inp.get("floor_plan")
    if not isinstance(floor_plan, dict):
        PARSED_TOTAL.labels(status="failure").inc()
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
    activity.logger.info("floor plan json accepted request_id=%s", inp.get("request_id"))
    return {"floor_plan": result}


async def main() -> None:
    logging.basicConfig(level=os.getenv("LOG_LEVEL", "INFO"))
    temporal_address = os.getenv("TEMPORAL_ADDRESS", "localhost:7233")
    namespace = os.getenv("TEMPORAL_NAMESPACE", "default")
    task_queue = os.getenv("TEMPORAL_TASK_QUEUE", "floor-parser")
    metrics_port = int(os.getenv("METRICS_PORT", "2113"))

    start_http_server(metrics_port)
    client = await Client.connect(temporal_address, namespace=namespace)
    log.info("worker started task_queue=%s temporal=%s", task_queue, temporal_address)

    async with Worker(client, task_queue=task_queue, workflows=[], activities=[parse_floor_json]):
        await asyncio.Event().wait()


if __name__ == "__main__":
    asyncio.run(main())
