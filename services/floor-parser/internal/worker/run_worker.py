from __future__ import annotations

import asyncio

import structlog
from temporalio.client import Client
from temporalio.worker import Worker

from internal.worker.activities import ActivityState, init_activity_state, parse_floor_activity
from internal.worker.observability import WorkerSettings, configure_logging, configure_metrics, configure_tracing


async def run_worker(settings: WorkerSettings) -> None:
    configure_logging(settings)
    configure_tracing(settings)
    metrics = configure_metrics(settings)
    logger = structlog.get_logger()

    init_activity_state(
        ActivityState(
            semaphore=asyncio.Semaphore(settings.compute_concurrency),
            metrics=metrics,
            service_name=settings.service_name,
        )
    )

    client = await Client.connect(
        settings.temporal_address,
        namespace=settings.temporal_namespace,
    )
    logger.info(
        "temporal client connected",
        address=settings.temporal_address,
        namespace=settings.temporal_namespace,
    )

    async with Worker(
        client,
        task_queue=settings.temporal_task_queue,
        workflows=[],
        activities=[parse_floor_activity],
        max_concurrent_activities=settings.max_concurrent_activities,
    ):
        logger.info(
            "floor-parser worker started",
            task_queue=settings.temporal_task_queue,
            max_concurrent_activities=settings.max_concurrent_activities,
            compute_concurrency=settings.compute_concurrency,
            metrics_port=settings.metrics_port,
        )
        await asyncio.Event().wait()
