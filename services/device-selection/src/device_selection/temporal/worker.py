from __future__ import annotations

import asyncio

import asyncpg
import structlog
from temporalio.client import Client
from temporalio.worker import Worker

from device_selection.config import Settings
from device_selection.temporal.activities import (
    ActivityState,
    init_activity_state,
    select_devices,
    select_devices_from_file,
)
from device_selection.temporal.observability import (
    configure_logging,
    configure_metrics,
    configure_tracing,
)


async def run_worker(settings: Settings) -> None:
    configure_logging(settings)
    configure_tracing(settings, "device-selection-worker")
    metrics = configure_metrics(settings, "device_selection")
    log = structlog.get_logger()

    pool: asyncpg.Pool | None = None
    if settings.database.enabled:
        pool = await asyncpg.create_pool(settings.database.dsn)
        log.info("database pool created")
    else:
        log.info("database pool disabled")

    init_activity_state(
        ActivityState(
            pool=pool,
            settings=settings,
            metrics=metrics,
            semaphore=asyncio.Semaphore(settings.worker.compute_concurrency),
            service_name="device-selection-worker",
        )
    )

    client = await Client.connect(
        settings.temporal.address,
        namespace=settings.temporal.namespace,
    )
    log.info(
        "temporal client connected",
        address=settings.temporal.address,
        namespace=settings.temporal.namespace,
    )

    async with Worker(
        client,
        task_queue=settings.temporal.task_queue,
        workflows=[],
        activities=[select_devices, select_devices_from_file],
        max_concurrent_activities=settings.worker.max_concurrent_activities,
    ):
        log.info(
            "worker started",
            task_queue=settings.temporal.task_queue,
            max_concurrent_activities=settings.worker.max_concurrent_activities,
            compute_concurrency=settings.worker.compute_concurrency,
            metrics_port=settings.observability.metrics_port,
        )
        await asyncio.Event().wait()
