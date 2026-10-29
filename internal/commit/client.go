package commit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type CommitRequest struct {
	TaskID string `json:"task_id"`
	Result string `json:"result"`
}

type HTTPStatusError struct {
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf(
		"commit failed with HTTP status %d",
		e.StatusCode,
	)
}

func SendResult(
	ctx context.Context,
	url string,
	secret string,
	taskID string,
	result string,
) error {
	commit := CommitRequest{
		TaskID: taskID,
		Result: result,
	}

	body, err := json.Marshal(commit)

	if err != nil {
		return fmt.Errorf("failed to convert: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)

	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return fmt.Errorf("failed to send commit request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HTTPStatusError{
			StatusCode: resp.StatusCode,
		}
	}

	return nil
}

func SendResultWithRetry(
	ctx context.Context,
	url string,
	secret string,
	taskID string,
	result string,
) error {
	const maxAttempts = 3

	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := SendResult(ctx, url, secret, taskID, result)

		if err == nil {
			return nil
		}

		lastErr = err

		var statusErr *HTTPStatusError

		if errors.As(err, &statusErr) {
			status := statusErr.StatusCode

			if status != http.StatusTooManyRequests &&
				(status < 500 || status >= 600) {
				return err
			}
		}

		if attempt == maxAttempts {
			break
		}

		delay := time.Duration(attempt) * time.Second

		timer := time.NewTimer(delay)

		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}

	return fmt.Errorf(
		"failed to send result after %d attempts: %w",
		maxAttempts,
		lastErr,
	)
}
