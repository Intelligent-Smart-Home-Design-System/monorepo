from __future__ import annotations

from typing import Protocol

from quality_calculator.domain.models import DeviceRecord


class QualityRepository(Protocol):
    async def get_pending_devices(
        self, limit: int, after_id: int = 0, recompute_all: bool = False
    ) -> list[DeviceRecord]:
        """
        Возвращает до `limit` устройств с id > after_id, упорядоченных по id.
        По умолчанию только те, у кого quality IS NULL (свежепостроенные).
        recompute_all=True — игнорировать quality (полный пересчёт).
        Keyset-пагинация по id гарантирует продвижение даже для устройств,
        которым качество посчитать нельзя (нет сигналов) — без бесконечного цикла.
        """
        ...

    async def save_quality(self, device_id: int, quality: float) -> None:
        """Записывает посчитанное качество в devices.quality."""
        ...
