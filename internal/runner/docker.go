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

func runExec(
	ctx context.Context,
	cli *client.Client,
	containerID string,
	command []string,
) (string, int, error) {
	execResponse, err := cli.ContainerExecCreate(
		ctx,
		containerID,
		container.ExecOptions{
			Cmd:          command,
			AttachStdout: true,
			AttachStderr: true,
		},
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create exec: %w", err)
	}

	execAttach, err := cli.ContainerExecAttach(
		ctx,
		execResponse.ID,
		container.ExecAttachOptions{},
	)

	if err != nil {
		return "", 0, fmt.Errorf("failed to attach exec: %w", err)
	}

	defer execAttach.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	_, err = stdcopy.StdCopy(&stdout, &stderr, execAttach.Reader)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read exec output: %w", err)
	}

	inspect, err := cli.ContainerExecInspect(ctx, execResponse.ID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to inspect exec: %w", err)
	}

	return stdout.String() + stderr.String(), inspect.ExitCode, nil
}

func RunCode(
	ctx context.Context,
	cli *client.Client,
	code string,
	translator string,
) (string, error) {
	var filename string
	var command []string
	var compileCommand []string

	switch translator {
	case "python3":
		filename = "main.py"
		command = []string{"python3", "/workspace/main.py"}
	case "gcc":
		filename = "main.c"
		compileCommand = []string{
			"gcc",
			"/workspace/main.c",
			"-o",
			"/build/main",
		}
		command = []string{"/build/main"}
	case "clang":
		filename = "main.c"
		compileCommand = []string{
			"clang",
			"/workspace/main.c",
			"-o",
			"/build/main",
		}
		command = []string{"/build/main"}
	case "g++":
		filename = "main.cpp"
		compileCommand = []string{
			"g++",
			"/workspace/main.cpp",
			"-o",
			"/build/main",
		}
		command = []string{"/build/main"}
	case "clang++":
		filename = "main.cpp"
		compileCommand = []string{
			"clang++",
			"/workspace/main.cpp",
			"-o",
			"/build/main",
		}
		command = []string{"/build/main"}
	default:
		return "", fmt.Errorf("unsupported translator: %s", translator)
	}

	config := &container.Config{
		Image: "code-runner:latest",
		Cmd:   []string{"sleep", "infinity"},
		User:  "65534:65534",
	}
	pidsLimit := int64(32)

	hostConfig := &container.HostConfig{
		NetworkMode:    "none",
		ReadonlyRootfs: false,
		CapDrop:        []string{"ALL"},
		SecurityOpt:    []string{"no-new-privileges"},
		Resources: container.Resources{
			Memory:    128 * 1024 * 1024,
			NanoCPUs:  500_000_000,
			PidsLimit: &pidsLimit,
		},
		Tmpfs: map[string]string{
			"/tmp":   "rw,noexec,nosuid,nodev,size=16m",
			"/build": "rw,exec,nosuid,nodev,size=32m,mode=1777",
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

	archive, err := createCodeArchive(filename, code)
	if err != nil {
		return "", fmt.Errorf("failed to create code archive: %w", err)
	}

	err = cli.CopyToContainer(
		ctx,
		response.ID,
		"/workspace",
		archive,
		container.CopyToContainerOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to copy code into container: %w", err)
	}

	if compileCommand != nil {
		compileResult, exitCode, err := runExec(ctx, cli, response.ID, compileCommand)

		if err != nil {
			return "", fmt.Errorf("failed to run compiler: %w", err)
		}

		if exitCode != 0 {
			return compileResult, nil
		}

		log.Printf("Compilation result: %s", compileResult)
	}

	result, _, err := runExec(ctx, cli, response.ID, command)
	if err != nil {
		return "", fmt.Errorf(
			"failed to execute code: %w; program output: %s",
			err,
			result,
		)
	}

	return result, nil
}
