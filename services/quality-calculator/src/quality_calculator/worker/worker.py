from __future__ import annotations

import structlog

from quality_calculator.evaluator import QualityEvaluator
from quality_calculator.ports.repository import QualityRepository


class Worker:
    def __init__(
        self,
        evaluator: QualityEvaluator,
        repository: QualityRepository,
        batch_size: int = 200,
        recompute_all: bool = False,
    ):
        self._evaluator = evaluator
        self._repository = repository
        self._batch_size = batch_size
        self._recompute_all = recompute_all

    async def run(self) -> dict[str, int]:
        log = structlog.get_logger()
        after_id = 0
        total = 0
        scored = 0
        skipped = 0  # устройства без сигналов — quality посчитать нельзя

        log.info("quality_scoring_started", recompute_all=self._recompute_all)

        while True:
            batch = await self._repository.get_pending_devices(
                limit=self._batch_size, after_id=after_id, recompute_all=self._recompute_all
            )
            if not batch:
                break

            for device in batch:
                total += 1
                result = self._evaluator.evaluate_device(device.to_eval_record())
                quality = result["Q_total"]
                if quality is None:
                    skipped += 1
                    continue
                try:
                    await self._repository.save_quality(device.id, quality)
                    scored += 1
                except Exception as e:  # noqa: BLE001 — не валим батч из-за одной записи
                    log.error("save_failed", device_id=device.id, error=str(e))

            after_id = batch[-1].id  # выборка упорядочена по id
            log.info("batch_done", processed=total, scored=scored, skipped=skipped)

        log.info("quality_scoring_finished", total=total, scored=scored, skipped=skipped)
        return {"total": total, "scored": scored, "skipped": skipped}
