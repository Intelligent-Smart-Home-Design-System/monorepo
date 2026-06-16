# Образ для запуска quality-calculator как джобы пайплайна (Temporal pipeline-worker).
#
# ВАЖНО: контекст сборки — КОРЕНЬ монорепо, а не папка сервиса.
# Сервису нужен shared/schemas/devices/device_types.json, который лежит вне
# services/quality-calculator. См. строку сборки в services/pipeline-worker/Makefile:
#   docker build -t quality-calculator:latest -f ../quality-calculator/Dockerfile ../..

FROM python:3.14-slim AS builder
COPY --from=ghcr.io/astral-sh/uv:latest /uv /uvx /bin/

ENV UV_PYTHON_DOWNLOADS=0

WORKDIR /app/services/quality-calculator

RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=services/quality-calculator/uv.lock,target=uv.lock \
    --mount=type=bind,source=services/quality-calculator/pyproject.toml,target=pyproject.toml \
    uv sync --locked --no-install-project --no-editable

COPY services/quality-calculator /app/services/quality-calculator
# Единственный файл из shared, который читает run (девайс-схема каталога).
COPY shared/schemas/devices/device_types.json /app/shared/schemas/devices/device_types.json

RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --locked --no-editable

FROM python:3.14-slim

COPY --from=builder /app /app

# WORKDIR подобран так, чтобы относительные пути в cli.py разрешались как в репо:
#   config/...                                  -> /app/services/quality-calculator/config/...
#   ../../shared/schemas/devices/device_types.json -> /app/shared/schemas/devices/device_types.json
WORKDIR /app/services/quality-calculator

# Команду переопределяет pipeline.toml; CMD — для ручного запуска образа.
CMD ["/app/services/quality-calculator/.venv/bin/quality-calculator", "run", "--config", "config.toml"]
