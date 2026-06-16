from __future__ import annotations

import os
import time
from typing import Any, Callable

import structlog
from opentelemetry import trace
from opentelemetry._logs import SeverityNumber
from opentelemetry.exporter.otlp.proto.http._log_exporter import OTLPLogExporter
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk._logs import LogRecord as OTELLogRecord
from opentelemetry.sdk._logs import LoggerProvider
from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor

_LEVEL_TO_SEVERITY: dict[str, SeverityNumber] = {
    "trace": SeverityNumber.TRACE,
    "debug": SeverityNumber.DEBUG,
    "info": SeverityNumber.INFO,
    "warning": SeverityNumber.WARN,
    "warn": SeverityNumber.WARN,
    "error": SeverityNumber.ERROR,
    "critical": SeverityNumber.FATAL,
    "fatal": SeverityNumber.FATAL,
}


class _OTELLogProcessor:
    """Structlog processor that forwards each log record to the OTLP log pipeline."""

    def __init__(self, logger_provider: LoggerProvider) -> None:
        self._logger = logger_provider.get_logger("floor-parser")

    def __call__(self, logger: Any, method: str, event_dict: dict[str, Any]) -> dict[str, Any]:
        span_ctx = trace.get_current_span().get_span_context()
        level = str(event_dict.get("level", method or "info")).lower()

        attrs = {
            k: str(v)
            for k, v in event_dict.items()
            if k not in ("event", "level", "timestamp")
        }

        self._logger.emit(
            OTELLogRecord(
                timestamp=time.time_ns(),
                observed_timestamp=time.time_ns(),
                trace_id=span_ctx.trace_id,
                span_id=span_ctx.span_id,
                trace_flags=span_ctx.trace_flags,
                severity_text=level.upper(),
                severity_number=_LEVEL_TO_SEVERITY.get(level, SeverityNumber.INFO),
                body=str(event_dict.get("event", "")),
                resource=None,
                attributes=attrs,
            )
        )
        return event_dict


def _http_endpoint() -> str | None:
    ep = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "").strip()
    if not ep:
        return None
    if not ep.startswith(("http://", "https://")):
        ep = "http://" + ep
    return ep


def setup_telemetry(service: str) -> Callable[[], None]:
    """Initialise OTLP traces and logs.  Must be called after setup_logging()
    and before the first structlog message is emitted.  Returns a shutdown fn."""
    endpoint = _http_endpoint()
    if not endpoint:
        return lambda: None

    resource = Resource.create({"service.name": service})

    tracer_provider = TracerProvider(resource=resource)
    tracer_provider.add_span_processor(
        BatchSpanProcessor(OTLPSpanExporter(endpoint=f"{endpoint}/v1/traces"))
    )
    trace.set_tracer_provider(tracer_provider)

    logger_provider = LoggerProvider(resource=resource)
    logger_provider.add_log_record_processor(
        BatchLogRecordProcessor(OTLPLogExporter(endpoint=f"{endpoint}/v1/logs"))
    )

    # Insert OTLP log processor into the structlog chain just before wrap_for_formatter.
    processors = list(structlog.get_config()["processors"])
    processors.insert(len(processors) - 1, _OTELLogProcessor(logger_provider))
    structlog.configure(processors=processors)

    def _shutdown() -> None:
        tracer_provider.shutdown()
        logger_provider.shutdown()

    return _shutdown
