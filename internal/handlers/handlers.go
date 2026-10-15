package handlers

import "go-integration-service-rabbitmq/internal/storage"

type Handler struct {
	storage storage.TaskStorage
}

func NewHandler(s storage.TaskStorage) *Handler {
	return &Handler{
		storage: s,
	}
}
