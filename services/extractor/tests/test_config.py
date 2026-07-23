import os
from pathlib import Path

import pytest

from extractor.config import LLMSource, Settings


BASE_TOML = """
llm_source = "{llm_source}"

[taxonomy]
path = "taxonomy_schema.json"

[extraction]
[extraction.hints]
model = "hint"
[extraction.hint_templates]
ecosystem = "tpl"

[database]
host = "localhost"
port = 5432
name = "catalog"
user = "catalog"
"""


def write_config(tmp_path: Path, llm_source: str, extra: str = "") -> Path:
    path = tmp_path / "config.toml"
    path.write_text(BASE_TOML.format(llm_source=llm_source) + extra, encoding="utf-8")
    return path


def test_local_llm_source(tmp_path: Path):
    path = write_config(
        tmp_path,
        "local",
        """
[local_llm]
base_url = "http://host.docker.internal:11434/v1"
model = "qwen2.5:7b"
api_key = "ollama"
""",
    )

    settings = Settings.from_toml(path)

    assert settings.llm_source == LLMSource.LOCAL
    assert settings.llm is not None
    assert settings.llm.provider.value == "openai_compatible"
    assert settings.llm.base_url.endswith("/v1")
    assert settings.yandex_cloud is None


def test_local_rejects_yandex_section(tmp_path: Path):
    path = write_config(
        tmp_path,
        "local",
        """
[local_llm]
base_url = "http://localhost:11434/v1"
model = "qwen2.5:7b"

[yandex_cloud]
folder = "abc"
llm_model = "gpt"
""",
    )

    with pytest.raises(ValueError, match="must not be set"):
        Settings.from_toml(path)


def test_yandex_llm_source(tmp_path: Path, monkeypatch: pytest.MonkeyPatch):
    path = write_config(
        tmp_path,
        "yandex_cloud",
        """
[yandex_cloud]
folder = "b1ghdbj0nn88kkalmtlj"
llm_model = "gpt-oss-120b/latest"
""",
    )
    monkeypatch.setenv("YANDEX_CLOUD_API_KEY", "test-key")

    settings = Settings.from_toml(path)

    assert settings.llm_source == LLMSource.YANDEX_CLOUD
    assert settings.llm is not None
    assert settings.llm.provider.value == "yandex_cloud"
    assert settings.llm.api_key == "test-key"
    assert settings.local_llm is None


def test_yandex_requires_api_key(tmp_path: Path, monkeypatch: pytest.MonkeyPatch):
    path = write_config(
        tmp_path,
        "yandex_cloud",
        """
[yandex_cloud]
folder = "abc"
llm_model = "gpt"
""",
    )
    monkeypatch.delenv("YANDEX_CLOUD_API_KEY", raising=False)

    with pytest.raises(ValueError, match="YANDEX_CLOUD_API_KEY"):
        Settings.from_toml(path)


def test_yandex_ignores_extractor_prefixed_yandex_env(tmp_path: Path, monkeypatch: pytest.MonkeyPatch):
    path = write_config(
        tmp_path,
        "local",
        """
[local_llm]
base_url = "http://localhost:11434/v1"
model = "qwen2.5:7b"
""",
    )
    monkeypatch.setenv("EXTRACTOR_YANDEX_CLOUD__API_KEY", "should-not-break-validation")

    settings = Settings.from_toml(path)
    assert settings.llm_source == LLMSource.LOCAL


def test_database_password_from_env(tmp_path: Path):
    path = write_config(
        tmp_path,
        "local",
        """
[local_llm]
base_url = "http://localhost:11434/v1"
model = "qwen2.5:7b"
""",
    )

    old = os.environ.get("EXTRACTOR_DATABASE__PASSWORD")
    os.environ["EXTRACTOR_DATABASE__PASSWORD"] = "secret"
    try:
        settings = Settings.from_toml(path)
        assert settings.database.password == "secret"
    finally:
        if old is None:
            os.environ.pop("EXTRACTOR_DATABASE__PASSWORD", None)
        else:
            os.environ["EXTRACTOR_DATABASE__PASSWORD"] = old
