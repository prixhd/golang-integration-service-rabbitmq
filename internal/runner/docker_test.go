package runner

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunCodePython(t *testing.T) {
	cli, err := NewDockerClient()

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

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

func TestRunCodeGCC(t *testing.T) {
	cli, err := NewDockerClient()
	code := `#include <stdio.h>
		int main() {
			printf("Hello from GCC!\n");
			return 0;
		}`

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	result, err := RunCode(
		ctx,
		cli,
		code,
		"gcc",
	)

	if err != nil {
		t.Fatalf("failed to run code: %v", err)
	}

	if !strings.Contains(result, "Hello from GCC!") {
		t.Errorf("unexpected result: %q", result)
	}

	t.Log("C code execution completed successfully")
}

func TestRunCodeClang(t *testing.T) {
	cli, err := NewDockerClient()
	code := `#include <stdio.h>

			int main() {
				printf("Hello from Clang!\n");
				return 0;
			}`

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	result, err := RunCode(
		ctx,
		cli,
		code,
		"clang",
	)

	if err != nil {
		t.Fatalf("failed to run code: %v", err)
	}

	if !strings.Contains(result, "Hello from Clang!") {
		t.Errorf("unexpected result: %q", result)
	}

	t.Log("C code execution completed successfully")
}

func TestRunCodeGPlusPlus(t *testing.T) {
	cli, err := NewDockerClient()
	code := `#include <iostream>

			int main() {
				std::cout << "Hello from C++!" << std::endl;
				return 0;
			}`

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	result, err := RunCode(
		ctx,
		cli,
		code,
		"g++",
	)

	if err != nil {
		t.Fatalf("failed to run code: %v", err)
	}

	if !strings.Contains(result, "Hello from C++!") {
		t.Errorf("unexpected result: %q", result)
	}

	t.Log("C++ code execution completed successfully")
}

func TestRunCodeClangPlusPlus(t *testing.T) {
	cli, err := NewDockerClient()
	code := `#include <iostream>

			int main() {
				std::cout << "Hello from C++!" << std::endl;
				return 0;
			}`

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	result, err := RunCode(
		ctx,
		cli,
		code,
		"clang++",
	)

	if err != nil {
		t.Fatalf("failed to run code: %v", err)
	}

	if !strings.Contains(result, "Hello from C++!") {
		t.Errorf("unexpected result: %q", result)
	}

	t.Log("C++ code execution completed successfully")
}

func TestRunCodeGCCCompilationError(t *testing.T) {
	cli, err := NewDockerClient()
	code := `int main() {
				unknown_variable;
				return 0;
			}`

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	result, err := RunCode(
		ctx,
		cli,
		code,
		"gcc",
	)

	if err != nil {
		t.Fatalf("failed to run code: %v", err)
	}

	if !strings.Contains(result, "unknown_variable") {
		t.Errorf("unexpected result: %q", result)
	}

	t.Log("GCC compilation error handled successfully")
}

func TestRunCodePythonError(t *testing.T) {
	cli, err := NewDockerClient()
	code := `print(10 / 0)`

	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	defer cli.Close()

	if err := CheckDocker(cli); err != nil {
		t.Skipf("Docker Engine is unavailable: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	defer cancel()

	result, err := RunCode(
		ctx,
		cli,
		code,
		"python3",
	)

	if err != nil {
		t.Fatalf("failed to run code: %v", err)
	}

	if !strings.Contains(result, "ZeroDivisionError") {
		t.Errorf("unexpected result: %q", result)
	}

	t.Log("Python runtime error handled successfully")
}
