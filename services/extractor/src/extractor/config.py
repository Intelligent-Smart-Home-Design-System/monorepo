from pydantic_settings import BaseSettings, SettingsConfigDict
from pydantic import BaseModel, JsonValue
import tomli
from pathlib import Path
from typing import Dict


class YandexCloudModelsConfig(BaseModel):
    folder: str
    api_key: str = ""
    llm_model: str
    temperature: float = 0


class DatabaseConfig(BaseModel):
    host: str = "localhost"
    port: int = 5432
    name: str
    user: str
    password: str = ""

    @property
    def dsn(self) -> str:
        return f"postgresql://{self.user}:{self.password}@{self.host}:{self.port}/{self.name}"


class LoggingConfig(BaseModel):
    level: str = "INFO"
    format: str = "json"


class TaxonomyConfig(BaseModel):
    path: str = "taxonomy_schema.json"


class ExtractionConfig(BaseModel):
    hints: dict[str, str]
    hint_templates: dict[str, str]


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="EXTRACTOR_",
        env_nested_delimiter="__",
    )

    database: DatabaseConfig
    logging: LoggingConfig = LoggingConfig()
    yandex_cloud: YandexCloudModelsConfig
    taxonomy: TaxonomyConfig
    extraction: ExtractionConfig
    batch_size: int = 10

    @classmethod
    def from_toml(cls, path: Path = Path("config.toml")) -> Settings:
        with open(path, "rb") as f:
            data = tomli.load(f)
        return cls(**data)
