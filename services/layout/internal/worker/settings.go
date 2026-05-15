package worker

import (
	"os"
	"strconv"
	"strings"
)

type Settings struct {
	ServiceName             string
	TemporalAddress         string
	TemporalNamespace       string
	TemporalTaskQueue       string
	MetricsListenAddress    string
	MaxConcurrentActivities int
	ComputeConcurrency      int64
	TracksConfigPath        string
	DevicesConfigPath       string
	TracingEnabled          bool
	OTLPEndpoint            string
	OTLPInsecure            bool
}

func LoadSettings() Settings {
	return Settings{
		ServiceName:             getEnv("SERVICE_NAME", "layout-worker"),
		TemporalAddress:         getEnv("TEMPORAL_ADDRESS", "localhost:7233"),
		TemporalNamespace:       getEnv("TEMPORAL_NAMESPACE", "default"),
		TemporalTaskQueue:       getEnv("TEMPORAL_TASK_QUEUE", "main-pipeline-layout"),
		MetricsListenAddress:    getEnv("METRICS_LISTEN_ADDRESS", ":2114"),
		MaxConcurrentActivities: getIntEnv("MAX_CONCURRENT_ACTIVITIES", 32),
		ComputeConcurrency:      int64(getIntEnv("COMPUTE_CONCURRENCY", 4)),
		TracksConfigPath:        getEnv("TRACKS_CONFIG_PATH", "internal/configs/tracks.json"),
		DevicesConfigPath:       getEnv("DEVICES_CONFIG_PATH", "internal/configs/devices.json"),
		TracingEnabled:          getBoolEnv("TRACING_ENABLED", true),
		OTLPEndpoint:            strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		OTLPInsecure:            getBoolEnv("OTEL_EXPORTER_OTLP_INSECURE", true),
	}
}

func getEnv(key string, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getBoolEnv(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
