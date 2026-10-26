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
	"time"

	"github.com/docker/docker/client"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func executeWithRetry(
	cli *client.Client,
	task models.TaskMessage,
) (string, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)

		result, err := runner.RunCode(
			ctx,
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

		time.Sleep(time.Duration(attempt) * time.Second)
	}

	return "", fmt.Errorf("unexpected retry loop termination")
}

func run() error {
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

	for msg := range msgs {
		var taskMessage models.TaskMessage

		err := json.Unmarshal(msg.Body, &taskMessage)
		if err != nil {
			log.Printf("failed to decode task message: %v", err)
			if nackErr := msg.Nack(false, false); nackErr != nil {
				return fmt.Errorf("failed to nack invalid message: %w", nackErr)
			}
			continue
		}

		start := time.Now()
		result, err := executeWithRetry(cli, taskMessage)

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

		commitCtx, commitCancel := context.WithTimeout(context.Background(), 10*time.Second)

		err = commit.SendResultWithRetry(
			commitCtx,
			commitURL,
			commitSecret,
			taskMessage.TaskID,
			result,
		)

		commitCancel()

		if err != nil {
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

	return fmt.Errorf("RabbitMQ deliveries channel closed unexpectedly")
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
