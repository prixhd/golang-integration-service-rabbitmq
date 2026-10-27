package storage

import (
	"errors"
	"go-integration-service-rabbitmq/internal/models"
)

var ErrUserExists = errors.New("user already exists")
var ErrNotFound = errors.New("not found")

type OutboxEvent struct {
	ID      int64
	TaskID  string
	Payload []byte
}

type TaskStorage interface {
	Save(task models.Task) error
	SaveWithOutbox(task models.Task, message models.TaskMessage) error
	Get(id string) (models.Task, error)
}

type UserStorage interface {
	SaveUser(user models.User) error
	GetUserByLogin(login string) (models.User, error)
}

type SessionStorage interface {
	SaveSession(session models.Session) error
	GetSession(sessionID string) (models.Session, error)
}

type OutboxStorage interface {
	GetPendingEvents(limit int) ([]OutboxEvent, error)
	MarkPublished(id int64) error
}
