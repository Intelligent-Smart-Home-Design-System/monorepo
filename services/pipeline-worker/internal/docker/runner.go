package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/pipeline"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog"
	"go.temporal.io/sdk/activity"
)

type Settings struct {
	Host                  string
	NetworkName           string
	MonitoringNetworkName string
	ContainerPrefix       string
	AutoRemove            bool
	ConfigRoot            string
}

type Runner struct {
	client                *client.Client
	networkName           string
	monitoringNetworkName string
	containerPrefix       string
	autoRemove            bool
	configRoot            string
	logger                zerolog.Logger
}

var invalidContainerChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

func NewRunner(settings Settings, logger zerolog.Logger) (*Runner, error) {
	options := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if settings.Host != "" {
		options = append(options, client.WithHost(settings.Host))
	}

	dockerClient, err := client.NewClientWithOpts(options...)
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	return &Runner{
		client:                dockerClient,
		networkName:           settings.NetworkName,
		monitoringNetworkName: settings.MonitoringNetworkName,
		containerPrefix:       settings.ContainerPrefix,
		autoRemove:            settings.AutoRemove,
		configRoot:            settings.ConfigRoot,
		logger:                logger,
	}, nil
}

func (r *Runner) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *Runner) Run(ctx context.Context, params pipeline.RunContainerParams) (*pipeline.RunContainerResult, error) {
	resolvedCommand, containerConfigPath, err := r.resolveCommand(params)
	if err != nil {
		return nil, err
	}

	r.logger.Info().
		Str("job", params.Name).
		Str("image", params.Image).
		Strs("command", resolvedCommand).
		Msg("Starting job container")

	imageInspect, _, err := r.client.ImageInspectWithRaw(ctx, params.Image)
	if err != nil {
		return nil, fmt.Errorf("inspect image %q: %w", params.Image, err)
	}
	r.logger.Info().
		Str("job", params.Name).
		Str("image", params.Image).
		Str("image_id", shortImageID(imageInspect.ID)).
		Msg("Job container image ready")

	envValues, err := r.resolveEnv(params.EnvMapping, params.Name)
	if err != nil {
		return nil, err
	}
	envValues = appendScraperBrowserEnv(envValues, params)
	envValues = appendOTelEnv(envValues, r.monitoringNetworkName)

	containerName := buildContainerName(r.containerPrefix, params.Name)
	labels := map[string]string{
		"pipeline_job": params.Name,
		"service":      "pipeline-worker",
	}

	hostConfig := &container.HostConfig{}
	if binds, err := buildVolumeBinds(params.Volumes, params.Name); err != nil {
		return nil, err
	} else if len(binds) > 0 {
		hostConfig.Binds = binds
	}
	if shm := parseShmSize(params.ShmSize); shm > 0 {
		hostConfig.ShmSize = shm
	}
	networkingConfig := r.containerNetworking()
	if networkingConfig == nil && r.networkName != "" {
		hostConfig.NetworkMode = container.NetworkMode(r.networkName)
	}

	created, err := r.client.ContainerCreate(
		ctx,
		&container.Config{
			Image:  params.Image,
			Cmd:    resolvedCommand,
			Env:    envValues,
			Labels: labels,
		},
		hostConfig,
		networkingConfig,
		nil,
		containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	r.logger.Info().
		Str("job", params.Name).
		Str("image", params.Image).
		Str("container_id", created.ID).
		Str("container_name", containerName).
		Msg("Job container created")

	if r.autoRemove {
		defer func() {
			r.logger.Info().
				Str("job", params.Name).
				Str("container_id", created.ID).
				Msg("Removing job container")
			removeErr := r.client.ContainerRemove(context.Background(), created.ID, container.RemoveOptions{Force: true})
			if removeErr != nil {
				r.logger.Warn().Err(removeErr).Str("container_id", created.ID).Msg("Failed to remove job container")
				return
			}
			r.logger.Info().
				Str("job", params.Name).
				Str("container_id", created.ID).
				Msg("Job container removed")
		}()
	}

	if containerConfigPath != "" {
		if err := r.copyConfigDir(ctx, created.ID, params.ConfigPath); err != nil {
			return nil, fmt.Errorf("copy config into container: %w", err)
		}
		r.logger.Info().
			Str("job", params.Name).
			Str("container_id", created.ID).
			Str("config_path", containerConfigPath).
			Msg("Job config copied into container")
	}

	startedAt := time.Now().UTC()
	if err := r.client.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	r.logger.Info().
		Str("job", params.Name).
		Str("container_id", created.ID).
		Msg("Job container started")

	var statusCode int64
	for {
		// heartbeat so temporal knows we're alive
		activity.RecordHeartbeat(ctx, created.ID)

		// check if temporal cancelled us
		if ctx.Err() != nil {
			_ = r.client.ContainerStop(context.Background(), created.ID, container.StopOptions{})
			return nil, ctx.Err()
		}

		inspect, err := r.client.ContainerInspect(ctx, created.ID)
		if err != nil {
			return nil, fmt.Errorf("inspect container: %w", err)
		}

		if !inspect.State.Running {
			statusCode = int64(inspect.State.ExitCode)
			break
		}

		time.Sleep(5 * time.Second)
	}

	logs, logErr := r.containerLogs(ctx, created.ID)
	if logErr != nil {
		r.logger.Warn().Err(logErr).Str("container_id", created.ID).Msg("Failed to read container logs")
	}

	completedAt := time.Now().UTC()
	result := &pipeline.RunContainerResult{
		Name:        params.Name,
		Image:       params.Image,
		Command:     resolvedCommand,
		ContainerID: created.ID,
		ExitCode:    statusCode,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		Logs:        logs,
	}

	r.logger.Info().
		Str("job", params.Name).
		Str("image", params.Image).
		Int64("exit_code", statusCode).
		Str("container_id", created.ID).
		Dur("duration", completedAt.Sub(startedAt)).
		Msg("Job container finished")

	emitJobContainerLogs(r.logger, params.Name, logs)

	if statusCode != 0 {
		return result, fmt.Errorf("container %s exited with code %d", params.Name, statusCode)
	}

	return result, nil
}

func (r *Runner) resolveCommand(params pipeline.RunContainerParams) ([]string, string, error) {
	if len(params.Command) == 0 {
		return nil, "", fmt.Errorf("job %q has empty command", params.Name)
	}

	containerConfigPath := ""
	if params.ConfigPath != "" {
		hostPath := params.ConfigPath
		if !filepath.IsAbs(hostPath) {
			hostPath = filepath.Join(r.configRoot, filepath.FromSlash(params.ConfigPath))
		}
		containerConfigPath = path.Join(pipeline.DefaultConfigDir, filepath.Base(hostPath))
	}

	resolved := make([]string, len(params.Command))
	for index, part := range params.Command {
		value := strings.ReplaceAll(part, pipeline.ConfigDirPlaceholder, pipeline.DefaultConfigDir)
		value = strings.ReplaceAll(value, pipeline.ConfigPathPlaceholder, containerConfigPath)
		resolved[index] = value
	}

	return resolved, containerConfigPath, nil
}

func (r *Runner) containerNetworking() *network.NetworkingConfig {
	if r.networkName == "" || r.monitoringNetworkName == "" {
		return nil
	}
	if strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) == "" {
		return nil
	}

	return &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			r.networkName:           {},
			r.monitoringNetworkName: {},
		},
	}
}

func appendOTelEnv(values []string, monitoringNetworkName string) []string {
	if monitoringNetworkName == "" {
		return values
	}
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return values
	}

	values = append(values, "OTEL_EXPORTER_OTLP_ENDPOINT="+endpoint)
	if insecure, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_INSECURE"); ok {
		values = append(values, "OTEL_EXPORTER_OTLP_INSECURE="+insecure)
	} else {
		values = append(values, "OTEL_EXPORTER_OTLP_INSECURE=true")
	}
	return values
}

func (r *Runner) resolveEnv(envMapping map[string]string, jobName string) ([]string, error) {
	keys := make([]string, 0, len(envMapping))
	for key := range envMapping {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	values := make([]string, 0, len(keys))
	for _, targetKey := range keys {
		sourceKey := envMapping[targetKey]
		sourceValue, ok := os.LookupEnv(sourceKey)
		if !ok {
			return nil, fmt.Errorf("required env %q for job %q is not set", sourceKey, jobName)
		}
		values = append(values, fmt.Sprintf("%s=%s", targetKey, sourceValue))
	}

	return values, nil
}

func shortImageID(imageID string) string {
	if len(imageID) <= 12 {
		return imageID
	}
	return imageID[:12]
}

func (r *Runner) copyConfigDir(ctx context.Context, containerID string, configPath string) error {
	hostPath := configPath
	if !filepath.IsAbs(hostPath) {
		hostPath = filepath.Join(r.configRoot, filepath.FromSlash(configPath))
	}

	sourceDir := filepath.Dir(hostPath)
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("stat config dir %q: %w", sourceDir, err)
	}

	archive, err := tarDirectory(sourceDir, strings.TrimPrefix(pipeline.DefaultConfigDir, "/"))
	if err != nil {
		return err
	}

	return r.client.CopyToContainer(ctx, containerID, "/", archive, container.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
}

func (r *Runner) containerLogs(ctx context.Context, containerID string) (string, error) {
	reader, err := r.client.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, reader); err != nil {
		return "", err
	}

	combined := strings.TrimSpace(strings.Join([]string{
		strings.TrimSpace(stdout.String()),
		strings.TrimSpace(stderr.String()),
	}, "\n"))

	truncate := func(logs string, maxBytes int) string {
		if len(logs) <= maxBytes {
			return logs
		}
		// keep the tail
		return "...[truncated]...\n" + logs[len(logs)-maxBytes:]
	}

	return truncate(strings.TrimSpace(combined), 100_000), nil
}

func tarDirectory(sourceDir string, destinationRoot string) (io.ReadCloser, error) {
	buffer := bytes.NewBuffer(nil)
	writer := tar.NewWriter(buffer)

	err := filepath.Walk(sourceDir, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(sourceDir, current)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = path.Join(destinationRoot, filepath.ToSlash(relativePath))

		if err := writer.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(current)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(writer, file); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		_ = writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(buffer.Bytes())), nil
}

func emitJobContainerLogs(logger zerolog.Logger, jobName string, logs string) {
	logs = strings.TrimSpace(logs)
	if logs == "" {
		return
	}

	const maxLines = 200
	allLines := strings.Split(logs, "\n")
	lines := allLines
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
		logger.Info().
			Str("pipeline_job", jobName).
			Int("log_lines_truncated_from", len(allLines)).
			Msg("pipeline job container logs truncated")
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		event := zerolog.Dict()
		event = event.Str("pipeline_job", jobName).Str("raw_line", line)

		var fields map[string]interface{}
		if err := json.Unmarshal([]byte(line), &fields); err == nil {
			if msg, ok := fields["event"].(string); ok && msg != "" {
				event = event.Str("job_event", msg)
			}
			for key, value := range fields {
				if key == "event" || key == "level" || key == "timestamp" {
					continue
				}
				event = event.Interface(key, value)
			}
			logger.Info().Dict("job_log", event).Msg("pipeline job container log")
			continue
		}

		logger.Info().Dict("job_log", event).Msg("pipeline job container log")
	}
}

func buildContainerName(prefix string, jobName string) string {
	sanitized := invalidContainerChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(jobName)), "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		sanitized = "job"
	}
	return fmt.Sprintf("%s-%s-%d", prefix, sanitized, time.Now().Unix())
}

func buildVolumeBinds(volumes []pipeline.VolumeMount, jobName string) ([]string, error) {
	if len(volumes) == 0 {
		return nil, nil
	}
	binds := make([]string, 0, len(volumes))
	for _, v := range volumes {
		source := expandEnvPlaceholders(v.Source)
		target := strings.TrimSpace(v.Target)
		if source == "" {
			return nil, fmt.Errorf("job %q volume source is empty (set DNS_CHROME_PROFILE_HOST or use a named volume)", jobName)
		}
		if target == "" {
			return nil, fmt.Errorf("job %q volume target is required", jobName)
		}
		mountType := strings.ToLower(strings.TrimSpace(v.Type))
		if mountType == "" {
			if strings.Contains(source, "/") || strings.Contains(source, `\`) || strings.Contains(source, ":") {
				mountType = "bind"
			} else {
				mountType = "volume"
			}
		}
		var bind string
		switch mountType {
		case "bind":
			bind = fmt.Sprintf("%s:%s", source, target)
		case "volume":
			bind = fmt.Sprintf("%s:%s", source, target)
		default:
			return nil, fmt.Errorf("job %q volume type %q is unsupported", jobName, mountType)
		}
		if v.ReadOnly {
			bind += ":ro"
		}
		binds = append(binds, bind)
	}
	return binds, nil
}

func expandEnvPlaceholders(value string) string {
	return os.ExpandEnv(value)
}

func parseShmSize(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if strings.HasSuffix(strings.ToLower(raw), "g") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(strings.ToLower(raw), "g"), 64)
		if err != nil {
			return 0
		}
		return int64(n * 1024 * 1024 * 1024)
	}
	if strings.HasSuffix(strings.ToLower(raw), "m") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(strings.ToLower(raw), "m"), 64)
		if err != nil {
			return 0
		}
		return int64(n * 1024 * 1024)
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// appendScraperBrowserEnv gives each pipeline scraper job its own Chrome user-data dir
// so concurrent or back-to-back containers never fight over SingletonLock on a shared volume.
func appendScraperBrowserEnv(values []string, params pipeline.RunContainerParams) []string {
	if !strings.Contains(params.Image, "scraper") {
		return values
	}
	for _, v := range values {
		if strings.HasPrefix(v, "DNS_BROWSER_PROFILE=") {
			return values
		}
	}
	if !strings.Contains(params.Name, "scrape") {
		return values
	}
	dir := fmt.Sprintf("/tmp/pipeline-dns-chrome-%s-%d", strings.TrimPrefix(params.Name, "scraper-"), time.Now().UnixNano())
	return append(values, "DNS_BROWSER_PROFILE="+dir)
}
