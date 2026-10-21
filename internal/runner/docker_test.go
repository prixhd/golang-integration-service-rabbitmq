package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunCodePython(t *testing.T) {
	cli, err := NewDockerClient()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	result, err := RunCode(
		ctx,
		cli,
		`print("Hello from Docker!")`,
		"python3",
	)

	if err != nil {
		t.Fatalf("failed to run code: %v", err)
	}

	if !strings.Contains(result, "Hello from Docker!") {
		t.Errorf("unexpected result: %q", result)
	}

	t.Log("Python execution completed successfully")
}
