package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-integration-service-rabbitmq/internal/commit"
	"go-integration-service-rabbitmq/internal/metrics"
	"go-integration-service-rabbitmq/internal/models"
	"go-integration-service-rabbitmq/internal/rabbitmq"
	"go-integration-service-rabbitmq/internal/runner"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/google/uuid"

	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func executeWithRetry(
	ctx context.Context,
	cli *client.Client,
	task models.TaskMessage,
) (string, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		taskCtx, cancel := context.WithTimeout(
			ctx,
			10*time.Second,
		)

		result, err := runner.RunCode(
			taskCtx,
			cli,
			task.Code,
			task.Translator,
		)

		cancel()

		if err == nil {
			return result, nil
		}

		if !errors.Is(err, runner.ErrContainerPreparation) {
			return result, err
		}

		if attempt == maxAttempts {
			return result, err
		}

		log.Printf(
			"Task %s: container preparation failed, retry %d/%d: %v",
			task.TaskID,
			attempt,
			maxAttempts,
			err,
		)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}

	return "", fmt.Errorf("unexpected retry loop termination")
}

func commitUntilSuccess(
	ctx context.Context,
	url string,
	secret string,
	taskID string,
	result string,
) error {
	for {
		requestCtx, cancel := context.WithTimeout(
			ctx,
			10*time.Second,
		)

		err := commit.SendResultWithRetry(
			requestCtx,
			url,
			secret,
			taskID,
			result,
		)

		cancel()

		if err == nil {
			return nil
		}

		var statusErr *commit.HTTPStatusError

		if errors.As(err, &statusErr) {
			status := statusErr.StatusCode

			if status != http.StatusTooManyRequests &&
				(status < 500 || status >= 600) {
				return fmt.Errorf(
					"permanent commit error for task %s: %w",
					taskID,
					err,
				)
			}
		}

		log.Printf(
			"Task %s: commit failed, retrying in 5 seconds: %v",
			taskID,
			err,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func run(ctx context.Context) error {
	// prometeus registr metrics
	metrics.RegisterProcessorMetrics()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":9091", metricsMux); err != nil {
			log.Printf("metrics server error: %v", err)
		}
	}()

	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

	commitURL := os.Getenv("COMMIT_URL")
	if commitURL == "" {
		commitURL = "http://localhost:8000/commit"
	}

	commitSecret := os.Getenv("COMMIT_SECRET")
	if commitSecret == "" {
		return fmt.Errorf("COMMIT_SECRET is required")
	}

	conn, err := rabbitmq.NewConnection(url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to create rabbitmq channel: %w", err)
	}
	defer ch.Close()

	_, err = rabbitmq.DeclareQueue(ch)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	cli, err := runner.NewDockerClient()

	if err != nil {
		return fmt.Errorf("failed to create new docker client: %w", err)
	}
	defer cli.Close()

	if err := runner.CheckDocker(cli); err != nil {
		return fmt.Errorf("failed to check docker client: %w", err)
	}

	cleanupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := runner.CleanupOrphanContainers(cleanupCtx, cli); err != nil {
		return fmt.Errorf("failed to cleanup orphan containers: %w", err)
	}

	log.Println("Docker Engine connected successfully")

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("failed to configure RabbitMQ QoS: %w", err)
	}

	msgs, err := ch.Consume(
		"tasks",
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	log.Println("CodeProcessor connected to RabbitMQ")

	for {
		var msg amqp.Delivery

		select {
		case <-ctx.Done():
			log.Println("CodeProcessor shutting down")
			return nil

		case delivery, ok := <-msgs:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("RabbitMQ deliveries channel closed unexpectedly")
			}
			msg = delivery
		}

		var taskMessage models.TaskMessage

		err := json.Unmarshal(msg.Body, &taskMessage)
		if err != nil {
			log.Printf("failed to decode task message: %v", err)
			if nackErr := msg.Nack(false, false); nackErr != nil {
				return fmt.Errorf("failed to nack invalid message: %w", nackErr)
			}
			continue
		}

		if taskMessage.TaskID == "" ||
			taskMessage.Code == "" ||
			taskMessage.Translator == "" {

			log.Printf("invalid task message: missing required fields")

			if err := msg.Nack(false, false); err != nil {
				return fmt.Errorf("failed to nack invalid task: %w", err)
			}

			continue
		}

		if _, err := uuid.Parse(taskMessage.TaskID); err != nil {
			log.Printf("invalid task ID: %q", taskMessage.TaskID)

			if nackErr := msg.Nack(false, false); nackErr != nil {
				return fmt.Errorf("failed to nack invalid task ID: %w", nackErr)
			}

			continue
		}

		start := time.Now()
		result, err := executeWithRetry(ctx, cli, taskMessage)
		if ctx.Err() != nil {
			log.Printf("Processor is shutting down, task %s will be redelivered", taskMessage.TaskID)
			return nil
		}

		// prometeus CodeProcessor histogram
		duration := time.Since(start).Seconds()
		metrics.TaskDuration.WithLabelValues(taskMessage.Translator).Observe(duration)

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				result = "Execution time limit exceed (10 sec)"
				log.Printf(
					"Task %s exceed execution time limit",
					taskMessage.TaskID,
				)
			} else if errors.Is(err, runner.ErrUnsupportedTranslator) {
				result = err.Error()
			} else if errors.Is(err, runner.ErrContainerPreparation) {
				result = "Execution failed: unable to prepare Docker container after 3 attempts"

				log.Printf(
					"Task %s failed after container preparation retries: %v",
					taskMessage.TaskID,
					err,
				)
			} else {
				result = "Execution failed: internal code runner error"

				log.Printf(
					"Task %s failed during execution: %v",
					taskMessage.TaskID,
					err,
				)
			}
		}

		log.Printf("Task %s result: %s", taskMessage.TaskID, result)

		err = commitUntilSuccess(
			ctx,
			commitURL,
			commitSecret,
			taskMessage.TaskID,
			result,
		)

		if ctx.Err() != nil {
			log.Printf(
				"Processor is shutting down, task %s will be redelivered",
				taskMessage.TaskID,
			)
			return nil
		}

		if err != nil {
			var statusErr *commit.HTTPStatusError

			if errors.As(err, &statusErr) {
				if statusErr.StatusCode == http.StatusUnauthorized {
					return fmt.Errorf(
						"commit authorization failed for task %s: %w",
						taskMessage.TaskID,
						err,
					)
				}

				if statusErr.StatusCode >= 400 &&
					statusErr.StatusCode < 500 &&
					statusErr.StatusCode != http.StatusTooManyRequests {

					log.Printf(
						"Task %s: permanent commit error, moving to DLQ: %v",
						taskMessage.TaskID,
						err,
					)

					if nackErr := msg.Nack(false, false); nackErr != nil {
						return fmt.Errorf(
							"failed to nack task %s: %w",
							taskMessage.TaskID,
							nackErr,
						)
					}

					continue
				}
			}

			return fmt.Errorf(
				"failed to commit task %s: %w",
				taskMessage.TaskID,
				err,
			)
		}

		err = msg.Ack(false)

		if err != nil {
			return fmt.Errorf("failed to ack task %s: %w", taskMessage.TaskID, err)
		}

		// prometeus CodeProcessor counter
		metrics.TasksTotal.
			WithLabelValues(taskMessage.Translator).
			Inc()

	}

}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}
