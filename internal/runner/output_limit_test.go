package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOutputLimit(t *testing.T) {
	cli, err := NewDockerClient()
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	code := `print("A" * 200000)`

	result, err := RunCode(ctx, cli, code, "python3")
	if err != nil {
		t.Fatalf("RunCode failed: %v", err)
	}

	if !strings.Contains(result, "Output limit exceeded") {
		t.Fatalf("expected output limit diagnostic, got length %d", len(result))
	}

	if len(result) > maxOutputBytes+100 {
		t.Fatalf("result unexpectedly large: %d bytes", len(result))
	}
}
