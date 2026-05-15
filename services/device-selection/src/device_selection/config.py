from __future__ import annotations

from pathlib import Path
from typing import Optional

import tomli
from pydantic import BaseModel


class DatabaseConfig(BaseModel):
    enabled: bool = True
    host: str = "localhost"
    port: int = 5432
    name: str = "catalog"
    user: str = "catalog"
    password: str = "catalog"

    @property
    def dsn(self) -> str:
        return f"postgresql://{self.user}:{self.password}@{self.host}:{self.port}/{self.name}"


class TemporalConfig(BaseModel):
    host: str = "localhost"
    port: int = 7233
    namespace: str = "default"
    task_queue: str = "device-selection"

    @property
    def address(self) -> str:
        return f"{self.host}:{self.port}"


class QualityConfig(BaseModel):
    calculate: bool = True
    min_reviews: int = 10
    global_avg_rating: float = 4.0
    rating_floor: float = 4.0


class LoggingConfig(BaseModel):
    format: str = "console"


class WorkerConfig(BaseModel):
    max_concurrent_activities: int = 32
    compute_concurrency: int = 4


class ObservabilityConfig(BaseModel):
    metrics_host: str = "0.0.0.0"
    metrics_port: int = 2115
    tracing_enabled: bool = True
    otlp_endpoint: Optional[str] = None
    otlp_insecure: bool = True


class SolverConfig(BaseModel):
    max_bridge_ecosystems: int = 5
    max_hub_types: int = 4
    max_candidates_per_type: Optional[int] = None
    default_time_budget_seconds: float = 120.0


class Settings(BaseModel):
    database: DatabaseConfig
    temporal: TemporalConfig = TemporalConfig()
    quality: QualityConfig = QualityConfig()
    logging: LoggingConfig = LoggingConfig()
    worker: WorkerConfig = WorkerConfig()
    observability: ObservabilityConfig = ObservabilityConfig()
    solver: SolverConfig = SolverConfig()
    catalog_ttl_seconds: float = 86400.0 # 1 day

    @classmethod
    def from_toml(cls, path: Path) -> Settings:
        with open(path, "rb") as f:
            data = tomli.load(f)
        return cls.model_validate(data)
