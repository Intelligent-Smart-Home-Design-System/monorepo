from __future__ import annotations

from dataclasses import dataclass

import structlog
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from prometheus_client import Counter, Gauge, Histogram, PlatformCollector, ProcessCollector, Registry, start_http_server

from device_selection.config import Settings


@dataclass
class Metrics:
    registry: Registry
    runs_total: Counter
    duration_seconds: Histogram
    concurrent_runs: Gauge


def configure_logging(settings: Settings) -> None:
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.JSONRenderer()
            if settings.logging.format == "json"
            else structlog.dev.ConsoleRenderer(),
        ]
    )


def configure_tracing(settings: Settings, service_name: str) -> None:
    observability = settings.observability
    if not observability.tracing_enabled or not observability.otlp_endpoint:
        return

    provider = TracerProvider(
        resource=Resource.create(
            {
                "service.name": service_name,
                "deployment.environment": "development",
                "service.version": "dev",
            }
        )
    )
    exporter = OTLPSpanExporter(
        endpoint=observability.otlp_endpoint,
        insecure=observability.otlp_insecure,
    )
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)


def configure_metrics(settings: Settings, prefix: str) -> Metrics:
    registry = Registry()
    ProcessCollector(registry=registry)
    PlatformCollector(registry=registry)

    metrics = Metrics(
        registry=registry,
        runs_total=Counter(
            f"{prefix}_activity_runs_total",
            "Total number of activity runs grouped by status.",
            ("activity", "status"),
            registry=registry,
        ),
        duration_seconds=Histogram(
            f"{prefix}_activity_duration_seconds",
            "Duration of activity runs.",
            ("activity",),
            registry=registry,
        ),
        concurrent_runs=Gauge(
            f"{prefix}_activity_concurrent_runs",
            "Number of currently executing activities.",
            ("activity",),
            registry=registry,
        ),
    )

    start_http_server(
        settings.observability.metrics_port,
        addr=settings.observability.metrics_host,
        registry=registry,
    )
    return metrics
