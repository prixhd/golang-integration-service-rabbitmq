package storage

import (
	"fmt"
	"go-integration-service-rabbitmq/internal/models"
	"sync"
)

type MemoryStorage struct {
	tasks map[uint64]models.Task
	mu    sync.RWMutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tasks: make(map[uint64]models.Task),
	}
}

func (s *MemoryStorage) Save(task models.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
	return nil
}

func (s *MemoryStorage) Get(id uint64) (models.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[id]

	if !ok {
		return models.Task{}, fmt.Errorf("task %d not found", id)
	}

	return task, nil
}
