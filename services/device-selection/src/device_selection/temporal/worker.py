from __future__ import annotations

import asyncio
import os

import asyncpg
import structlog
from prometheus_client import start_http_server
from temporalio.client import Client
from temporalio.worker import Worker

from device_selection.config import Settings
from device_selection.temporal.activities import (
    ActivityState,
    init_activity_state,
    select_devices,
    select_devices_json,
)


async def run_worker(settings: Settings) -> None:
    log = structlog.get_logger()
    start_http_server(int(os.getenv("METRICS_PORT", "2115")))

    pool = await asyncpg.create_pool(settings.database.dsn)
    log.info("database pool created")

    init_activity_state(ActivityState(pool=pool, settings=settings))

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
        workflows=[],                     # no workflows — we are an activity provider
        activities=[select_devices, select_devices_json],
    ):
        log.info("worker started", task_queue=settings.temporal.task_queue)
        await asyncio.Event().wait()      # run until SIGINT / SIGTERM
