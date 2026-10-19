package main

import (
	_ "go-integration-service-rabbitmq/docs"
	"go-integration-service-rabbitmq/internal/handlers"
	"go-integration-service-rabbitmq/internal/storage"
	"log"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Golang integration service rabbitmq swagger
// @version 1.0
// @description Swagger api for golang service
func main() {
	store := storage.NewMemoryStorage()

	handler := handlers.NewHandler(store, store, store)

	mux := http.NewServeMux()

	mux.HandleFunc("/task", handler.CreateTask)
	mux.HandleFunc("/status/", handler.GetStatus)
	mux.HandleFunc("/result/", handler.GetResult)
	mux.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	log.Fatal(http.ListenAndServe(":8000", mux))
}
