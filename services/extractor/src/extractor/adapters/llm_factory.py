import os
from typing import Literal

import outlines
from openai import AsyncOpenAI
from outlines.models.base import AsyncModel

from extractor.config import LLMConfig

Provider = Literal["yandex_cloud", "openai_compatible"]


def _resolve_api_key(llm: LLMConfig) -> str:
    if llm.api_key:
        return llm.api_key
    if llm.provider == "yandex_cloud":
        api_key = os.environ.get("YANDEX_CLOUD_API_KEY", "")
        if not api_key:
            raise ValueError("YANDEX_CLOUD_API_KEY env var not set")
        return api_key
    return os.environ.get("OPENAI_API_KEY", "ollama")


def make_openai_client(llm: LLMConfig) -> AsyncOpenAI:
    api_key = _resolve_api_key(llm)

    if llm.provider == "yandex_cloud":
        if not llm.folder:
            raise ValueError("llm.folder is required for yandex_cloud provider")
        return AsyncOpenAI(
            api_key=api_key,
            base_url=llm.base_url,
            project=llm.folder,
            default_headers={"Authorization": f"Api-Key {api_key}"},
        )

    return AsyncOpenAI(
        api_key=api_key,
        base_url=llm.base_url,
    )


def make_outlines_model(llm: LLMConfig) -> AsyncModel:
    client = make_openai_client(llm)
    if llm.provider == "yandex_cloud":
        model_name = f"gpt://{llm.folder}/{llm.model}"
    else:
        model_name = llm.model
    return outlines.from_openai(client, model_name)
