package main

import (
	"encoding/json"
	"fmt"
	"go-integration-service-rabbitmq/internal/models"
	"go-integration-service-rabbitmq/internal/rabbitmq"
	"log"
	"os"
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

	for msg := range msgs {
		var taskMessage models.TaskMessage

		err := json.Unmarshal(msg.Body, &taskMessage)
		if err != nil {
			log.Printf("failed to decode task message: %v", err)
			continue
		}

		log.Printf("Task id: %s", taskMessage.TaskID)
		log.Printf("Translator: %s", taskMessage.Translator)
		log.Printf("Code: %s", taskMessage.Code)

	}

	log.Println("CodeProcessor connected to RabbitMQ")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
