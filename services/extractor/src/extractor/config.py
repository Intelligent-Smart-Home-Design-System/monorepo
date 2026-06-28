import os
from enum import Enum
from pathlib import Path

import tomli
from pydantic import BaseModel, Field, model_validator


class LLMSource(str, Enum):
    LOCAL = "local"
    YANDEX_CLOUD = "yandex_cloud"


class LLMProvider(str, Enum):
    YANDEX_CLOUD = "yandex_cloud"
    OPENAI_COMPATIBLE = "openai_compatible"


class LocalLLMConfig(BaseModel):
    base_url: str
    model: str
    api_key: str = "ollama"
    temperature: float = 0


class YandexCloudConfig(BaseModel):
    folder: str
    llm_model: str
    temperature: float = 0


class LLMConfig(BaseModel):
    provider: LLMProvider
    model: str
    base_url: str
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


class Settings(BaseModel):
    llm_source: LLMSource
    local_llm: LocalLLMConfig | None = None
    yandex_cloud: YandexCloudConfig | None = None
    llm: LLMConfig | None = Field(default=None, exclude=True)

    database: DatabaseConfig
    logging: LoggingConfig = LoggingConfig()
    taxonomy: TaxonomyConfig
    extraction: ExtractionConfig
    pre_llm_gate: PreLLMGateConfig = PreLLMGateConfig()
    batch_size: int = 10

    @model_validator(mode="after")
    def validate_llm_source(self) -> "Settings":
        if self.llm_source == LLMSource.LOCAL:
            if self.local_llm is None:
                raise ValueError("[local_llm] is required when llm_source = 'local'")
            if self.yandex_cloud is not None:
                raise ValueError("[yandex_cloud] must not be set when llm_source = 'local'")
            self.llm = LLMConfig(
                provider=LLMProvider.OPENAI_COMPATIBLE,
                model=self.local_llm.model,
                base_url=self.local_llm.base_url,
                api_key=self.local_llm.api_key,
                temperature=self.local_llm.temperature,
            )
            return self

        if self.yandex_cloud is None:
            raise ValueError("[yandex_cloud] is required when llm_source = 'yandex_cloud'")
        if self.local_llm is not None:
            raise ValueError("[local_llm] must not be set when llm_source = 'yandex_cloud'")

        api_key = os.environ.get("YANDEX_CLOUD_API_KEY", "").strip()
        if not api_key:
            raise ValueError(
                "YANDEX_CLOUD_API_KEY environment variable is required when llm_source = 'yandex_cloud'"
            )

        self.llm = LLMConfig(
            provider=LLMProvider.YANDEX_CLOUD,
            model=self.yandex_cloud.llm_model,
            folder=self.yandex_cloud.folder,
            base_url="https://ai.api.cloud.yandex.net/v1",
            api_key=api_key,
            temperature=self.yandex_cloud.temperature,
        )
        return self

    @classmethod
    def from_toml(cls, path: Path = Path("config.toml")) -> "Settings":
        with open(path, "rb") as f:
            data = tomli.load(f)

        settings = cls.model_validate(data)

        password = os.environ.get("EXTRACTOR_DATABASE__PASSWORD")
        if password is not None:
            settings.database.password = password

        return settings
