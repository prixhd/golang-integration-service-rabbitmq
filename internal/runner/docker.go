package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/client"
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
		return fmt.Errorf("docker engine dont answer: %w", err)
	}

	return nil
}
