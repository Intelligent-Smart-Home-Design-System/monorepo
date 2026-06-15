FROM python:3.14-slim
WORKDIR /app
COPY pyproject.toml uv.lock README.md ./
COPY src ./src
RUN pip install --no-cache-dir uv && uv sync --no-dev
COPY . .
ENV PATH="/app/.venv/bin:$PATH"
EXPOSE 2115
CMD ["device-selection", "run", "--config", "/app/config.toml"]
