package commit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendResultWithRetryUnauthorized(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusUnauthorized)
		},
	))
	defer server.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err := SendResultWithRetry(
		ctx,
		server.URL,
		"wrong-secret",
		"test-task-id",
		"test-result",
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestSendResultWithRetry(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			attempts++

			if attempts < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.WriteHeader(http.StatusOK)
		},
	))
	defer server.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	err := SendResultWithRetry(
		ctx,
		server.URL,
		"test-secret",
		"test-task-id",
		"Hello from test",
	)

	if err != nil {
		t.Fatalf("expected successful commit, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}
