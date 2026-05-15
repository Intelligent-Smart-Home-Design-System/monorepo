from __future__ import annotations

import os
from dataclasses import dataclass

import structlog
from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from prometheus_client import Counter, Gauge, Histogram, ProcessCollector, Registry, PlatformCollector, start_http_server


@dataclass
class WorkerSettings:
    service_name: str
    temporal_address: str
    temporal_namespace: str
    temporal_task_queue: str
    metrics_host: str
    metrics_port: int
    max_concurrent_activities: int
    compute_concurrency: int
    otlp_endpoint: str
    otlp_insecure: bool
    tracing_enabled: bool
    log_format: str

    @classmethod
    def from_env(cls) -> "WorkerSettings":
        return cls(
            service_name=os.getenv("SERVICE_NAME", "floor-parser-worker"),
            temporal_address=os.getenv("TEMPORAL_ADDRESS", "localhost:7233"),
            temporal_namespace=os.getenv("TEMPORAL_NAMESPACE", "default"),
            temporal_task_queue=os.getenv("TEMPORAL_TASK_QUEUE", "main-pipeline-floor-parser"),
            metrics_host=os.getenv("METRICS_HOST", "0.0.0.0"),
            metrics_port=int(os.getenv("METRICS_PORT", "2113")),
            max_concurrent_activities=int(os.getenv("MAX_CONCURRENT_ACTIVITIES", "32")),
            compute_concurrency=int(os.getenv("COMPUTE_CONCURRENCY", "4")),
            otlp_endpoint=os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
            otlp_insecure=os.getenv("OTEL_EXPORTER_OTLP_INSECURE", "true").lower() != "false",
            tracing_enabled=os.getenv("TRACING_ENABLED", "true").lower() != "false",
            log_format=os.getenv("LOG_FORMAT", "json"),
        )


@dataclass
class Metrics:
    registry: Registry
    runs_total: Counter
    duration_seconds: Histogram
    concurrent_runs: Gauge


def configure_logging(settings: WorkerSettings) -> None:
    renderer = (
        structlog.processors.JSONRenderer()
        if settings.log_format == "json"
        else structlog.dev.ConsoleRenderer()
    )
    structlog.configure(
        processors=[
            structlog.processors.TimeStamper(fmt="iso"),
            renderer,
        ]
    )


def configure_tracing(settings: WorkerSettings) -> None:
    if not settings.tracing_enabled or not settings.otlp_endpoint:
        return

    provider = TracerProvider(
        resource=Resource.create(
            {
                "service.name": settings.service_name,
                "deployment.environment": os.getenv("APP_ENV", "development"),
                "service.version": os.getenv("SERVICE_VERSION", "dev"),
            }
        )
    )
    exporter = OTLPSpanExporter(
        endpoint=settings.otlp_endpoint,
        insecure=settings.otlp_insecure,
    )
    provider.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(provider)


def configure_metrics(settings: WorkerSettings) -> Metrics:
    registry = Registry()
    ProcessCollector(registry=registry)
    PlatformCollector(registry=registry)

    metrics = Metrics(
        registry=registry,
        runs_total=Counter(
            "floor_parser_activity_runs_total",
            "Total number of floor-parser activity runs grouped by status.",
            ("status",),
            registry=registry,
        ),
        duration_seconds=Histogram(
            "floor_parser_activity_duration_seconds",
            "Duration of floor-parser activities.",
            registry=registry,
        ),
        concurrent_runs=Gauge(
            "floor_parser_activity_concurrent_runs",
            "Number of currently executing floor-parser activities.",
            registry=registry,
        ),
    )

    start_http_server(settings.metrics_port, addr=settings.metrics_host, registry=registry)
    return metrics
