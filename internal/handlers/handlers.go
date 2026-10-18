package handlers

import (
	"encoding/json"
	"go-integration-service-rabbitmq/internal/models"
	"go-integration-service-rabbitmq/internal/storage"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type Handler struct {
	storage storage.TaskStorage
}

type CreateTaskRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type CreateTaskResponse struct {
	TaskID string `json:"task_id"`
}

func NewHandler(s storage.TaskStorage) *Handler {
	return &Handler{
		storage: s,
	}
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var task CreateTaskRequest

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := json.NewDecoder(r.Body).Decode(&task)

	if err != nil && err != io.EOF {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	id := uuid.NewString()

	newTask := models.Task{
		ID:       id,
		Code:     task.Code,
		Language: task.Language,
		Status:   models.StatusInProgress,
		Result:   "",
	}

	if err := h.storage.Save(newTask); err != nil {
		http.Error(w, "Failed to save task", http.StatusInternalServerError)
		return
	}

	response := CreateTaskResponse{
		TaskID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(response)
}
