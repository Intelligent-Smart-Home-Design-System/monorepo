from __future__ import annotations

import asyncio

from internal.worker.observability import WorkerSettings
from internal.worker.run_worker import run_worker


def main() -> None:
    asyncio.run(run_worker(WorkerSettings.from_env()))


if __name__ == "__main__":
    main()
