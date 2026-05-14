package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Intelligent-Smart-Home-Design-System/monorepo/services/pipeline-worker/internal/pipeline"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/rs/zerolog"
)

type Settings struct {
	Host            string
	NetworkName     string
	ContainerPrefix string
	AutoRemove      bool
	ConfigRoot      string
}

type Runner struct {
	client          *client.Client
	networkName     string
	containerPrefix string
	autoRemove      bool
	configRoot      string
	logger          zerolog.Logger
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
		client:          dockerClient,
		networkName:     settings.NetworkName,
		containerPrefix: settings.ContainerPrefix,
		autoRemove:      settings.AutoRemove,
		configRoot:      settings.ConfigRoot,
		logger:          logger,
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

	if _, _, err := r.client.ImageInspectWithRaw(ctx, params.Image); err != nil {
		return nil, fmt.Errorf("inspect image %q: %w", params.Image, err)
	}

	envValues, err := r.resolveEnv(params)
	if err != nil {
		return nil, err
	}

	containerName := buildContainerName(r.containerPrefix, params.Name)
	labels := map[string]string{
		"pipeline_job": params.Name,
		"service":      "pipeline-worker",
	}

	hostConfig := &container.HostConfig{}
	if r.networkName != "" {
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
		&network.NetworkingConfig{},
		nil,
		containerName,
	)
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	if r.autoRemove {
		defer func() {
			removeErr := r.client.ContainerRemove(context.Background(), created.ID, container.RemoveOptions{Force: true})
			if removeErr != nil {
				r.logger.Warn().Err(removeErr).Str("container_id", created.ID).Msg("Failed to remove job container")
			}
		}()
	}

	if containerConfigPath != "" {
		if err := r.copyConfigDir(ctx, created.ID, params.ConfigPath); err != nil {
			return nil, fmt.Errorf("copy config into container: %w", err)
		}
	}

	startedAt := time.Now().UTC()
	if err := r.client.ContainerStart(ctx, created.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("start container: %w", err)
	}

	waitResponse, waitErrors := r.client.ContainerWait(ctx, created.ID, container.WaitConditionNotRunning)
	var statusCode int64
	select {
	case err := <-waitErrors:
		if err != nil {
			return nil, fmt.Errorf("wait for container: %w", err)
		}
	case response := <-waitResponse:
		statusCode = response.StatusCode
		if response.Error != nil && response.Error.Message != "" {
			return nil, fmt.Errorf("container exited with wait error: %s", response.Error.Message)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
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

func (r *Runner) resolveEnv(params pipeline.RunContainerParams) ([]string, error) {
	keys := make([]string, 0, len(params.EnvMapping))
	for key := range params.EnvMapping {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	values := make([]string, 0, len(keys))
	for _, targetKey := range keys {
		sourceKey := params.EnvMapping[targetKey]
		sourceValue, ok := os.LookupEnv(sourceKey)
		if !ok {
			return nil, fmt.Errorf("required env %q for job %q is not set", sourceKey, params.Name)
		}
		values = append(values, fmt.Sprintf("%s=%s", targetKey, sourceValue))
	}

	return values, nil
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
	return strings.TrimSpace(combined), nil
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

func buildContainerName(prefix string, jobName string) string {
	sanitized := invalidContainerChars.ReplaceAllString(strings.ToLower(strings.TrimSpace(jobName)), "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		sanitized = "job"
	}
	return fmt.Sprintf("%s-%s-%d", prefix, sanitized, time.Now().Unix())
}
