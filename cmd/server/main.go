package main

import (
	"context"
	"fmt"
	_ "go-integration-service-rabbitmq/docs"
	"go-integration-service-rabbitmq/internal/handlers"
	"go-integration-service-rabbitmq/internal/rabbitmq"
	"go-integration-service-rabbitmq/internal/storage"
	"log"
	"net/http"
	"os"
	"time"

	"go-integration-service-rabbitmq/internal/database"

	httpSwagger "github.com/swaggo/http-swagger"
)

func run() error {
	url := os.Getenv("RABBITMQ_URL")
	commitSecret := os.Getenv("COMMIT_SECRET")
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if url == "" {
		url = "amqp://guest:guest@localhost:5672/"
	}

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
		return fmt.Errorf("failed to declare queue rabbitmq: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	db, err := database.NewPostgres(ctx, databaseURL)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer db.Close()

	postgresStore := storage.NewPostgresStorage(db)
	sessionStore := storage.NewMemoryStorage()

	publisher := rabbitmq.NewPublisher(ch)

	handler := handlers.NewHandler(postgresStore, postgresStore, sessionStore, publisher)

	mux := http.NewServeMux()

	mux.HandleFunc("/task", handler.AuthMiddleware(handler.CreateTask))
	mux.HandleFunc("/status/", handler.AuthMiddleware(handler.GetStatus))
	mux.HandleFunc("/result/", handler.AuthMiddleware(handler.GetResult))
	mux.HandleFunc("/register", handler.Register)
	mux.HandleFunc("/login", handler.Login)
	mux.HandleFunc("/commit", handler.CommitMiddleware(handler.Commit, commitSecret))
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	return http.ListenAndServe(":8000", mux)
}

// @title Golang integration service rabbitmq swagger
// @version 1.0
// @description Swagger api for golang service
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your token as: Bearer <token>
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
