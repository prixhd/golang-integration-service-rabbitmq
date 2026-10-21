package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go-integration-service-rabbitmq/internal/models"
	"go-integration-service-rabbitmq/internal/rabbitmq"
	"go-integration-service-rabbitmq/internal/runner"
	"log"
	"os"
	"time"
)

func run() error {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
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
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		result, err := runner.RunCode(
			ctx,
			cli,
			taskMessage.Code,
			taskMessage.Translator,
		)

		cancel()

		if err != nil {
			log.Printf("failed to run code: %v", err)
			continue
		}

		log.Printf("Task %s result: %s", taskMessage.TaskID, result)

	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
