from pydantic_settings import BaseSettings
from pydantic import BaseModel
import tomli
from pathlib import Path


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
    format: str = "json"


class TaxonomyConfig(BaseModel):
    path: str = "taxonomy_schema.json"


class Settings(BaseSettings):
    database: DatabaseConfig
    taxonomy: TaxonomyConfig
    logging: LoggingConfig = LoggingConfig()

    @classmethod
    def from_toml(cls, path: Path = Path("config.toml")) -> Settings:
        with open(path, "rb") as f:
            data = tomli.load(f)
        return cls(**data)
