package runner

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func NewDockerClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)

	if err != nil {
		return nil, err
	}

	return cli, nil
}

func CheckDocker(cli *client.Client) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker engine is unavailable: %w", err)
	}

	return nil
}

func RunCode(
	ctx context.Context,
	cli *client.Client,
	code string,
	translator string,
) (string, error) {
	if translator != "python3" {
		return "", fmt.Errorf("unsupported translator: %s", translator)
	}

	config := &container.Config{
		Image: "code-runner:latest",
		Cmd:   []string{"python3", "-c", code},
		User:  "65534:65534",
	}
	pidsLimit := int64(32)

	hostConfig := &container.HostConfig{
		NetworkMode:    "none",
		ReadonlyRootfs: true,
		CapDrop:        []string{"ALL"},
		SecurityOpt:    []string{"no-new-privileges"},
		Resources: container.Resources{
			Memory:    128 * 1024 * 1024,
			NanoCPUs:  500_000_000,
			PidsLimit: &pidsLimit,
		},
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,nodev,size=16m",
		},
	}

	response, err := cli.ContainerCreate(
		ctx,
		config,
		hostConfig,
		nil,
		nil,
		"",
	)

	if err != nil {
		return "", fmt.Errorf("error with container create: %w", err)
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := cli.ContainerRemove(
			cleanupCtx,
			response.ID,
			container.RemoveOptions{Force: true},
		); err != nil {
			log.Printf("failed to remove container %s: %v", response.ID, err)
		}
	}()

	err = cli.ContainerStart(ctx, response.ID, container.StartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to start container : %w", err)
	}

	statusCh, errCh := cli.ContainerWait(
		ctx,
		response.ID,
		container.WaitConditionNotRunning,
	)

	select {
	case status := <-statusCh:
		log.Printf("Container exited with code: %d", status.StatusCode)

	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("failed to wait for container: %w", err)
		}

	case <-ctx.Done():
		return "", fmt.Errorf("container execution timeout: %w", ctx.Err())
	}

	logs, err := cli.ContainerLogs(
		ctx,
		response.ID,
		container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		},
	)

	if err != nil {
		return "", fmt.Errorf("failed to take logs from docker container: %w", err)
	}

	defer logs.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	_, err = stdcopy.StdCopy(&stdout, &stderr, logs)

	if err != nil {
		return "", fmt.Errorf("failed to take logs and logs error: %w", err)
	}

	return stdout.String() + stderr.String(), nil
}
