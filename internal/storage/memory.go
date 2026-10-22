package storage

import (
	"fmt"
	"go-integration-service-rabbitmq/internal/models"
	"sync"
)

type MemoryStorage struct {
	tasks    map[string]models.Task
	users    map[string]models.User
	sessions map[string]models.Session
	mu       sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tasks:    make(map[string]models.Task),
		users:    make(map[string]models.User),
		sessions: make(map[string]models.Session),
	}
}

func (s *MemoryStorage) Save(task models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
	return nil
}

func (s *MemoryStorage) Get(id string) (models.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]

	if !ok {
		return models.Task{}, fmt.Errorf("task %s not found", id)
	}

	return task, nil
}

func (s *MemoryStorage) SaveUser(user models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.users[user.Login]

	if exists {
		return fmt.Errorf("user %s: %w", user.Login, ErrUserExists)
	}

	s.users[user.Login] = user
	return nil
}

func (s *MemoryStorage) GetUserByLogin(login string) (models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[login]

	if !exists {
		return models.User{}, fmt.Errorf("user %s not found", login)
	}

	return user, nil
}

func (s *MemoryStorage) SaveSession(session models.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.sessions[session.SessionID]

	if exists {
		return fmt.Errorf("session %s already exists", session.SessionID)
	}

	s.sessions[session.SessionID] = session
	return nil
}

func (s *MemoryStorage) GetSession(sessionID string) (models.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]

	if !exists {
		return models.Session{}, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}
