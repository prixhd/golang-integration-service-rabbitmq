package commit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type CommitRequest struct {
	TaskID string `json:"task_id"`
	Result string `json:"result"`
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
		return fmt.Errorf("commit failed with status: %s", resp.Status)
	}

	return nil
}
