package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"go-integration-service-rabbitmq/internal/models"
	"go-integration-service-rabbitmq/internal/rabbitmq"
	"go-integration-service-rabbitmq/internal/storage"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	tasks     storage.TaskStorage
	users     storage.UserStorage
	sessions  storage.SessionStorage
	publisher rabbitmq.Publisher
}

type CreateTaskRequest struct {
	Code       string `json:"code"`
	Translator string `json:"translator"`
}

type CommitRequest struct {
	TaskID string `json:"task_id"`
	Result string `json:"result"`
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

type LoginResponse struct {
	Token string `json:"token"`
}

func NewHandler(
	tasks storage.TaskStorage,
	users storage.UserStorage,
	sessions storage.SessionStorage,
	publisher rabbitmq.Publisher,
) *Handler {
	return &Handler{
		tasks:     tasks,
		users:     users,
		sessions:  sessions,
		publisher: publisher,
	}
}

func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		authToken := strings.TrimPrefix(authHeader, "Bearer ")

		if authToken == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if _, err := h.sessions.GetSession(authToken); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func (h *Handler) CommitMiddleware(next http.HandlerFunc, secret string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		commitHeader := r.Header.Get("Authorization")

		if subtle.ConstantTimeCompare([]byte(commitHeader), []byte("Bearer "+secret)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// CreateTask godoc
// @Summary Create a new task
// @Description Accepts code for asynchronous processing
// @Param task body CreateTaskRequest true "Task data"
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Failure 401 {string} string "Unauthorized"
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

	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if task.Code == "" || task.Translator == "" {
		http.Error(w, "code and translator are required", http.StatusBadRequest)
		return
	}

	id := uuid.NewString()

	newTask := models.Task{
		ID:         id,
		Code:       task.Code,
		Translator: task.Translator,
		Status:     models.StatusInProgress,
		Result:     "",
	}

	if err := h.tasks.Save(newTask); err != nil {
		http.Error(w, "Failed to save task", http.StatusInternalServerError)
		return
	}

	message := models.TaskMessage{
		TaskID:     id,
		Translator: task.Translator,
		Code:       task.Code,
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		http.Error(w, "failed to convert json to []byte", http.StatusInternalServerError)
		return
	}

	if err := h.publisher.PublishTask(messageData); err != nil {
		http.Error(w, "failed public task to rabbitmq", http.StatusInternalServerError)
		return
	}

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
// @Security BearerAuth
// @Failure 401 {string} string "Unauthorized"
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
// @Security BearerAuth
// @Failure 401 {string} string "Unauthorized"
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

// @Summary Register a new user
// @Description Creates a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User credentials"
// @Success 201 {string} string "Created"
// @Failure 400 {string} string "Bad Request"
// @Failure 405 {string} string "Method not allowed"
// @Failure 409 {string} string "User already exists"
// @Failure 500 {string} string "Internal Server Error"
// @Router /register [post]
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
		if errors.Is(err, storage.ErrUserExists) {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}

		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

}

// Login godoc
// @Summary Login user
// @Description Login user to account
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "User credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {string} string "Username and password are required"
// @Failure 401 {string} string "Invalid credentials"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal Server Error"
// @Router /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
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

	user, err := h.users.GetUserByLogin(reg.Username)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(reg.Password),
	)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token := uuid.NewString()

	session := models.Session{
		UserID:    user.ID,
		SessionID: token,
	}

	if err := h.sessions.SaveSession(session); err != nil {
		http.Error(w, "Session already exists", http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

// Commit godoc
// @Summary Save task execution result
// @Description Internal endpoint used by CodeProcessor to save task results
// @Tags internal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CommitRequest true "Task execution result"
// @Success 200 {string} string "OK"
// @Failure 400 {string} string "Invalid request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Task not found"
// @Failure 405 {string} string "Method not allowed"
// @Failure 500 {string} string "Internal server error"
// @Router /commit [post]
func (h *Handler) Commit(w http.ResponseWriter, r *http.Request) {
	var commit CommitRequest

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := json.NewDecoder(r.Body).Decode(&commit)

	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if commit.TaskID == "" {
		http.Error(w, "task_id is required", http.StatusBadRequest)
		return
	}

	task, err := h.tasks.Get(commit.TaskID)

	if err != nil {
		http.Error(w, "task_id not found", http.StatusNotFound)
		return
	}

	task.Status = models.StatusReady
	task.Result = commit.Result

	err = h.tasks.Save(task)

	if err != nil {
		http.Error(w, "failed to save task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
