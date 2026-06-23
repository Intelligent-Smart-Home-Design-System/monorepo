from enum import Enum
from pathlib import Path

import tomli
from pydantic import BaseModel, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict


class LLMProvider(str, Enum):
    YANDEX_CLOUD = "yandex_cloud"
    OPENAI_COMPATIBLE = "openai_compatible"


class YandexCloudModelsConfig(BaseModel):
    folder: str
    api_key: str = ""
    llm_model: str
    temperature: float = 0


class LLMConfig(BaseModel):
    provider: LLMProvider = LLMProvider.YANDEX_CLOUD
    model: str
    base_url: str = "https://ai.api.cloud.yandex.net/v1"
    api_key: str = ""
    folder: str = ""
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


class PreLLMGateConfig(BaseModel):
    """
    Pre-LLM gate settings.

    See monorepo/docs/catalog-pipeline-architecture.md for the full check pipeline.
    """

    no_dups_check: bool = False
    """Skip content-hash and catalog-coverage checks (CLI: --no-dups-check)."""


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="EXTRACTOR_",
        env_nested_delimiter="__",
    )

    database: DatabaseConfig
    logging: LoggingConfig = LoggingConfig()
    llm: LLMConfig | None = None
    yandex_cloud: YandexCloudModelsConfig | None = None
    taxonomy: TaxonomyConfig
    extraction: ExtractionConfig
    pre_llm_gate: PreLLMGateConfig = PreLLMGateConfig()
    batch_size: int = 10

    @model_validator(mode="after")
    def resolve_llm(self) -> "Settings":
        if self.llm is not None:
            return self
        if self.yandex_cloud is None:
            raise ValueError("Either [llm] or [yandex_cloud] must be configured in config.toml")
        self.llm = LLMConfig(
            provider=LLMProvider.YANDEX_CLOUD,
            model=self.yandex_cloud.llm_model,
            folder=self.yandex_cloud.folder,
            api_key=self.yandex_cloud.api_key,
            temperature=self.yandex_cloud.temperature,
        )
        return self

    @classmethod
    def from_toml(cls, path: Path = Path("config.toml")) -> "Settings":
        with open(path, "rb") as f:
            data = tomli.load(f)
        return cls(**data)
