package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/pipeline"
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
	Temporal TemporalConfig         `toml:"temporal"`
	Docker   DockerConfig           `toml:"docker"`
	Metrics  MetricsConfig          `toml:"metrics"`
	Activity ActivityConfig         `toml:"activity"`
	Pipeline PipelineConfig         `toml:"pipeline"`
	Jobs     []pipeline.JobDefinition `toml:"jobs"`

	configPath string
}

type TemporalConfig struct {
	HostPort         string `toml:"host_port"`
	Namespace        string `toml:"namespace"`
	TaskQueue        string `toml:"task_queue"`
	ScheduleID       string `toml:"schedule_id"`
	ScheduleCron     string `toml:"schedule_cron"`
	ScheduleTimezone string `toml:"schedule_timezone"`
	ScheduleEnabled  bool   `toml:"schedule_enabled"`
	ConnectAttempts  int    `toml:"connect_attempts"`
	WorkflowIDPrefix string `toml:"workflow_id_prefix"`
}

type DockerConfig struct {
	Host            string `toml:"host"`
	NetworkName     string `toml:"network_name"`
	ContainerPrefix string `toml:"container_prefix"`
	AutoRemove      bool   `toml:"auto_remove"`
}

type MetricsConfig struct {
	ListenAddress string `toml:"listen_address"`
}

type ActivityConfig struct {
	StartToCloseTimeout Duration `toml:"start_to_close_timeout"`
	HeartbeatTimeout    Duration `toml:"heartbeat_timeout"`
	MaximumAttempts     int32    `toml:"maximum_attempts"`
	InitialInterval     Duration `toml:"initial_interval"`
	BackoffCoefficient  float64  `toml:"backoff_coefficient"`
	MaximumInterval     Duration `toml:"maximum_interval"`
}

type PipelineConfig struct {
	ConfigRoot string   `toml:"config_root"`
	JobNames   []string `toml:"job_names"`
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

	cfg.configPath = path
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
			TaskQueue:        "catalog-pipeline",
			ScheduleID:       "catalog-pipeline-daily",
			ScheduleCron:     "0 11 * * *",
			ScheduleTimezone: "UTC",
			ScheduleEnabled:  true,
			ConnectAttempts:  30,
			WorkflowIDPrefix: "catalog-pipeline",
		},
		Docker: DockerConfig{
			Host:            "unix:///var/run/docker.sock",
			NetworkName:     "catalog-pipeline-network",
			ContainerPrefix: "catalog-pipeline",
			AutoRemove:      true,
		},
		Metrics: MetricsConfig{
			ListenAddress: ":2112",
		},
		Activity: ActivityConfig{
			StartToCloseTimeout: Duration{Duration: 2 * time.Hour},
			HeartbeatTimeout:    Duration{Duration: 30 * time.Second},
			MaximumAttempts:     3,
			InitialInterval:     Duration{Duration: time.Minute},
			BackoffCoefficient:  2.0,
			MaximumInterval:     Duration{Duration: 15 * time.Minute},
		},
		Pipeline: PipelineConfig{
			ConfigRoot: "./jobs",
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
	if len(c.Jobs) == 0 {
		return fmt.Errorf("at least one [[jobs]] entry must be configured")
	}

	seen := make(map[string]struct{}, len(c.Jobs))
	for _, job := range c.Jobs {
		if job.Name == "" {
			return fmt.Errorf("job name is required")
		}
		if _, exists := seen[job.Name]; exists {
			return fmt.Errorf("duplicate job name %q", job.Name)
		}
		seen[job.Name] = struct{}{}
		if job.Image == "" {
			return fmt.Errorf("job %q must define image", job.Name)
		}
		if len(job.Command) == 0 {
			return fmt.Errorf("job %q must define command", job.Name)
		}
	}

	for _, name := range c.JobNames() {
		if _, exists := seen[name]; !exists {
			return fmt.Errorf("pipeline.job_names references unknown job %q", name)
		}
	}

	return nil
}

func (c *Config) ConfigRoot() string {
	root := c.Pipeline.ConfigRoot
	if filepath.IsAbs(root) {
		return filepath.Clean(root)
	}
	return filepath.Clean(filepath.Join(filepath.Dir(c.configPath), root))
}

func (c *Config) JobNames() []string {
	if len(c.Pipeline.JobNames) > 0 {
		return append([]string(nil), c.Pipeline.JobNames...)
	}

	names := make([]string, 0, len(c.Jobs))
	for _, job := range c.Jobs {
		names = append(names, job.Name)
	}
	return names
}

func (c *Config) WorkflowInput() pipeline.WorkflowInput {
	ordered := make([]pipeline.JobDefinition, 0, len(c.JobNames()))
	jobsByName := make(map[string]pipeline.JobDefinition, len(c.Jobs))
	for _, job := range c.Jobs {
		jobsByName[job.Name] = job
	}
	for _, name := range c.JobNames() {
		ordered = append(ordered, jobsByName[name])
	}

	return pipeline.WorkflowInput{
		Jobs: ordered,
		Activity: pipeline.ActivitySettings{
			StartToCloseTimeout: c.Activity.StartToCloseTimeout.Or(2 * time.Hour),
			HeartbeatTimeout:    c.Activity.HeartbeatTimeout.Or(30 * time.Second),
			Retry: pipeline.RetrySettings{
				MaximumAttempts:    c.Activity.MaximumAttempts,
				InitialInterval:    c.Activity.InitialInterval.Or(time.Minute),
				BackoffCoefficient: c.Activity.BackoffCoefficient,
				MaximumInterval:    c.Activity.MaximumInterval.Or(15 * time.Minute),
			},
		},
	}
}
