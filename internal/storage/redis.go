package storage

import (
	"context"
	"errors"
	"fmt"
	"go-integration-service-rabbitmq/internal/models"

	"github.com/redis/go-redis/v9"
)

type RedisStorage struct {
	client *redis.Client
}

func NewRedisStorage(client *redis.Client) *RedisStorage {
	return &RedisStorage{client: client}
}

func (s *RedisStorage) SaveSession(session models.Session) error {
	err := s.client.Set(
		context.Background(),
		"session:"+session.SessionID,
		session.UserID,
		0,
	).Err()

	if err != nil {
		return fmt.Errorf("fail to set session: %w", err)
	}

	return nil
}

func (s *RedisStorage) GetSession(sessionID string) (models.Session, error) {

	userID, err := s.client.Get(
		context.Background(),
		"session:"+sessionID,
	).Result()

	if errors.Is(err, redis.Nil) {
		return models.Session{}, fmt.Errorf("session %s not found: %w", sessionID, ErrNotFound)
	}

	if err != nil {
		return models.Session{}, fmt.Errorf("select session by id: %w", err)
	}

	return models.Session{
		UserID:    userID,
		SessionID: sessionID,
	}, nil
}
