module github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline

go 1.25.0

require (
	github.com/pelletier/go-toml/v2 v2.2.4
	github.com/prometheus/client_golang v1.20.5
	github.com/rs/zerolog v1.35.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	go.temporal.io/sdk v1.25.1
)
