FROM python:3.14-slim
WORKDIR /app

COPY pyproject.toml ./
COPY README.md ./
COPY src ./src
COPY evaluation ./evaluation
COPY config.toml ./config.toml
COPY config.main-pipeline.toml ./config.main-pipeline.toml

RUN pip install --no-cache-dir .

ENTRYPOINT ["device-selection", "run", "--config", "config.main-pipeline.toml"]
