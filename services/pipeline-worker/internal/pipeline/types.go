package pipeline

import "time"

const (
	ConfigPathPlaceholder = "{{config_path}}"
	ConfigDirPlaceholder  = "{{config_dir}}"
	DefaultConfigDir      = "/pipeline/config"
)

type JobDefinition struct {
	Name       string            `toml:"name" json:"name"`
	Image      string            `toml:"image" json:"image"`
	Command    []string          `toml:"command" json:"command"`
	ConfigPath string            `toml:"config_path" json:"config_path"`
	EnvMapping map[string]string `toml:"env_mapping" json:"env_mapping"`
}

type RetrySettings struct {
	MaximumAttempts    int32         `json:"maximum_attempts"`
	InitialInterval    time.Duration `json:"initial_interval"`
	BackoffCoefficient float64       `json:"backoff_coefficient"`
	MaximumInterval    time.Duration `json:"maximum_interval"`
}

type ActivitySettings struct {
	StartToCloseTimeout time.Duration `json:"start_to_close_timeout"`
	HeartbeatTimeout    time.Duration `json:"heartbeat_timeout"`
	Retry               RetrySettings `json:"retry"`
}

type WorkflowInput struct {
	Jobs     []JobDefinition  `json:"jobs"`
	Activity ActivitySettings `json:"activity"`
}

type RunContainerParams struct {
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Command    []string          `json:"command"`
	ConfigPath string            `json:"config_path"`
	EnvMapping map[string]string `json:"env_mapping"`
}

type RunContainerResult struct {
	Name        string    `json:"name"`
	Image       string    `json:"image"`
	Command     []string  `json:"command"`
	ContainerID string    `json:"container_id"`
	ExitCode    int64     `json:"exit_code"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Logs        string    `json:"logs"`
}

type WorkflowResult struct {
	StartedAt   time.Time            `json:"started_at"`
	CompletedAt time.Time            `json:"completed_at"`
	Jobs        []RunContainerResult `json:"jobs"`
}
