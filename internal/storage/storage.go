package storage

import "go-integration-service-rabbitmq/internal/models"

type TaskStorage interface {
	Save(task models.Task) error
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
