package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/main-pipeline/internal/pipeline"
	"github.com/pelletier/go-toml/v2"
)

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

func (d Duration) Or(defaultValue time.Duration) time.Duration {
	if d.Duration <= 0 {
		return defaultValue
	}
	return d.Duration
}

type Config struct {
	Temporal TemporalConfig `toml:"temporal"`
	Metrics  MetricsConfig  `toml:"metrics"`
	Queues   QueuesConfig   `toml:"queues"`
	Activity ActivityConfig `toml:"activity"`
	Trigger  TriggerConfig  `toml:"trigger"`
}

type TemporalConfig struct {
	HostPort         string `toml:"host_port"`
	Namespace        string `toml:"namespace"`
	TaskQueue        string `toml:"task_queue"`
	ConnectAttempts  int    `toml:"connect_attempts"`
	WorkflowIDPrefix string `toml:"workflow_id_prefix"`
}

type MetricsConfig struct {
	ListenAddress string `toml:"listen_address"`
}

type QueuesConfig struct {
	FloorParser     string `toml:"floor_parser"`
	Layout          string `toml:"layout"`
	DeviceSelection string `toml:"device_selection"`
}

type ActivityConfig struct {
	StartToCloseTimeout Duration `toml:"start_to_close_timeout"`
	HeartbeatTimeout    Duration `toml:"heartbeat_timeout"`
	MaximumAttempts     int32    `toml:"maximum_attempts"`
	InitialInterval     Duration `toml:"initial_interval"`
	BackoffCoefficient  float64  `toml:"backoff_coefficient"`
	MaximumInterval     Duration `toml:"maximum_interval"`
}

type TriggerConfig struct {
	RequestPath string `toml:"request_path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := defaultConfig()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func defaultConfig() Config {
	return Config{
		Temporal: TemporalConfig{
			HostPort:         "localhost:7233",
			Namespace:        "default",
			TaskQueue:        "main-pipeline-orchestration",
			ConnectAttempts:  30,
			WorkflowIDPrefix: "main-pipeline",
		},
		Metrics: MetricsConfig{
			ListenAddress: ":2112",
		},
		Queues: QueuesConfig{
			FloorParser:     "main-pipeline-floor-parser",
			Layout:          "main-pipeline-layout",
			DeviceSelection: "main-pipeline-device-selection",
		},
		Activity: ActivityConfig{
			StartToCloseTimeout: Duration{Duration: 15 * time.Minute},
			HeartbeatTimeout:    Duration{Duration: 30 * time.Second},
			MaximumAttempts:     3,
			InitialInterval:     Duration{Duration: 2 * time.Second},
			BackoffCoefficient:  2.0,
			MaximumInterval:     Duration{Duration: 1 * time.Minute},
		},
		Trigger: TriggerConfig{
			RequestPath: "",
		},
	}
}

func (c *Config) applyDefaults() {
	if c.Temporal.ConnectAttempts <= 0 {
		c.Temporal.ConnectAttempts = 30
	}
	if c.Activity.MaximumAttempts <= 0 {
		c.Activity.MaximumAttempts = 3
	}
	if c.Activity.BackoffCoefficient <= 0 {
		c.Activity.BackoffCoefficient = 2.0
	}
}

func (c *Config) validate() error {
	if c.Temporal.TaskQueue == "" {
		return fmt.Errorf("temporal.task_queue must be configured")
	}
	if c.Queues.FloorParser == "" || c.Queues.Layout == "" || c.Queues.DeviceSelection == "" {
		return fmt.Errorf("all worker queues must be configured")
	}
	return nil
}

func LoadRequest(path string) (*pipeline.Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read request: %w", err)
	}

	var request pipeline.Request
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}
	if request.RequestID == "" {
		request.RequestID = "main-pipeline-request"
	}
	return &request, nil
}

func (c *Config) WorkflowInput(request *pipeline.Request) pipeline.WorkflowInput {
	return pipeline.WorkflowInput{
		Request: *request,
		TaskQueues: pipeline.TaskQueues{
			Orchestration:   c.Temporal.TaskQueue,
			FloorParser:     c.Queues.FloorParser,
			Layout:          c.Queues.Layout,
			DeviceSelection: c.Queues.DeviceSelection,
		},
		Activity: pipeline.ActivitySettings{
			StartToCloseTimeout: c.Activity.StartToCloseTimeout.Or(15 * time.Minute),
			HeartbeatTimeout:    c.Activity.HeartbeatTimeout.Or(30 * time.Second),
			Retry: pipeline.RetrySettings{
				MaximumAttempts:    c.Activity.MaximumAttempts,
				InitialInterval:    c.Activity.InitialInterval.Or(2 * time.Second),
				BackoffCoefficient: c.Activity.BackoffCoefficient,
				MaximumInterval:    c.Activity.MaximumInterval.Or(time.Minute),
			},
		},
	}
}
