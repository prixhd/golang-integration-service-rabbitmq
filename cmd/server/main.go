package main

import (
	"go-integration-service-rabbitmq/internal/handlers"
	"go-integration-service-rabbitmq/internal/storage"
	"log"
	"net/http"
)

func main() {
	store := storage.NewMemoryStorage()

	handler := handlers.NewHandler(store)

	mux := http.NewServeMux()

	mux.HandleFunc("/task", handler.CreateTask)
	mux.HandleFunc("/status/", handler.GetStatus)
	mux.HandleFunc("/result/", handler.GetResult)

	log.Fatal(http.ListenAndServe(":8000", mux))
}
