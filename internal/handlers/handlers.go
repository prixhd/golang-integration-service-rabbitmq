package handlers

import (
	"encoding/json"
	"go-integration-service-rabbitmq/internal/models"
	"go-integration-service-rabbitmq/internal/storage"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	tasks    storage.TaskStorage
	users    storage.UserStorage
	sessions storage.SessionStorage
}

type CreateTaskRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateTaskResponse struct {
	TaskID string `json:"task_id"`
}

type GetTaskResponse struct {
	Status string `json:"status"`
}

type GetResultResponse struct {
	Result string `json:"result"`
}

func NewHandler(
	tasks storage.TaskStorage,
	users storage.UserStorage,
	sessions storage.SessionStorage,
) *Handler {
	return &Handler{
		tasks:    tasks,
		users:    users,
		sessions: sessions,
	}
}

// CreateTask godoc
// @Summary Create a new task
// @Description Accepts code for asynchronous processing
// @Param task body CreateTaskRequest false "Task data"
// @Tags tasks
// @Accept json
// @Produce json
// @Success 201 {object} CreateTaskResponse
// @Failure 400 {string} string "Invalid JSON"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /task [post]
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

	if err := h.tasks.Save(newTask); err != nil {
		http.Error(w, "Failed to save task", http.StatusInternalServerError)
		return
	}

	go func() {
		time.Sleep(3 * time.Second)

		newTask.Status = models.StatusReady
		newTask.Result = "Mock execution completed"

		if err := h.tasks.Save(newTask); err != nil {
			log.Printf("failed to update task %s: %v", newTask.ID, err)
		}
	}()

	response := CreateTaskResponse{
		TaskID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

// GetStatus godoc
// @Summary Retrieves status based on given ID
// @Produce json
// @Tags tasks
// @Description Returns the current processing status of a task
// @Param id path string true "Status ID"
// @Success 200 {object} GetTaskResponse
// @Failure 400 {string} string "Task ID is required"
// @Failure 404 {string} string "This task not found"
// @Failure 405 {string} string "This method not allowed"
// @Router /status/{id} [get]
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		http.Error(w, "This method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/status/")
	if id == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	task, err := h.tasks.Get(id)

	if err != nil {
		http.Error(w, "This task not found", http.StatusNotFound)
		return
	}

	response := GetTaskResponse{
		Status: task.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}

}

// GetResult godoc
// @Summary Retrieves result based on given ID
// @Produce json
// @Tags tasks
// @Description Returns result of a task
// @Param id path string true "Result ID"
// @Success 200 {object} GetResultResponse
// @Failure 400 {string} string "Task ID is required"
// @Failure 404 {string} string "This task not found"
// @Failure 405 {string} string "This method not allowed"
// @Router /result/{id} [get]
func (h *Handler) GetResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "This method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/result/")
	if id == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	task, err := h.tasks.Get(id)

	if err != nil {
		http.Error(w, "This task not found", http.StatusNotFound)
		return
	}

	response := GetResultResponse{
		Result: task.Result,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var reg RegisterRequest

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := json.NewDecoder(r.Body).Decode(&reg)

	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if reg.Username == "" || reg.Password == "" {
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(reg.Password),
		bcrypt.DefaultCost,
	)

	if err != nil {
		http.Error(w, "Error with generate hash from password", http.StatusInternalServerError)
		return
	}

	passwordHash := string(hash)
	id := uuid.NewString()

	user := models.User{
		ID:           id,
		Login:        reg.Username,
		PasswordHash: passwordHash,
	}

	if err := h.users.SaveUser(user); err != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

}
