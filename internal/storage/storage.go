package storage

import "go-integration-service-rabbitmq/internal/models"

type TaskStorage interface {
	Save(task models.Task) error
	Get(id uint64) (models.Task, error)
}
