from __future__ import annotations

from pathlib import Path

import tomli
from pydantic import BaseModel
from pydantic_settings import BaseSettings, SettingsConfigDict


class DatabaseConfig(BaseModel):
    host: str = "localhost"
    port: int = 5432
    name: str
    user: str
    password: str

    @property
    def dsn(self) -> str:
        return f"postgresql://{self.user}:{self.password}@{self.host}:{self.port}/{self.name}"


class LoggingConfig(BaseModel):
    level: str = "INFO"
    format: str = "json"  # "json" | "console"


class ScoringConfig(BaseModel):
    # Имя стратегии из config/strategies/<name>.json
    strategy: str = "full"
    # Размер батча устройств за одну выборку из БД
    batch_size: int = 200
    # Если true — пересчитать quality для всех устройств, а не только с quality IS NULL
    # (полезно после смены модели/весов; иначе перестроенные устройства подхватываются по NULL).
    recompute_all: bool = False
    # Пути к схемам/правилам (по умолчанию — раскладка монорепо относительно services/quality-calculator)
    device_types_path: str = "../../shared/schemas/devices/device_types.json"
    traits_path: str = "config/evaluation_traits.json"
    strategies_dir: str = "config/strategies"


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="QUALITY_CALCULATOR_",
        env_nested_delimiter="__",
    )

    database: DatabaseConfig
    logging: LoggingConfig = LoggingConfig()
    scoring: ScoringConfig = ScoringConfig()

    @classmethod
    def from_toml(cls, path: Path = Path("config.toml")) -> "Settings":
        with open(path, "rb") as f:
            data = tomli.load(f)
        return cls(**data)
