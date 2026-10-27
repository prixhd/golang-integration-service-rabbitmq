package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"go-integration-service-rabbitmq/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) SaveUser(user models.User) error {
	_, err := s.db.Exec(`INSERT INTO users (id, login, password_hash)
						VALUES ($1, $2, $3)`,
		user.ID,
		user.Login,
		user.PasswordHash,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("insert user: %w", ErrUserExists)
		}

		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (s *PostgresStorage) GetUserByLogin(login string) (models.User, error) {
	var user models.User

	err := s.db.QueryRow(
		`SELECT id, login, password_hash FROM users WHERE login = $1`,
		login,
	).Scan(&user.ID, &user.Login, &user.PasswordHash)

	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, fmt.Errorf("user %s not found: %w", login, ErrNotFound)
	} else if err != nil {
		return models.User{}, fmt.Errorf("select user by login: %w", err)
	}

	return user, nil
}

func (s *PostgresStorage) Save(task models.Task) error {
	_, err := s.db.Exec(`INSERT INTO tasks (id, code, translator, status, result)
							VALUES ($1, $2, $3, $4, $5)
							ON CONFLICT (id) DO UPDATE SET
								code = EXCLUDED.code,
								translator = EXCLUDED.translator,
								status = EXCLUDED.status,
								result = EXCLUDED.result;`,
		task.ID,
		task.Code,
		task.Translator,
		task.Status,
		task.Result,
	)

	if err != nil {
		return fmt.Errorf("insert task: %w", err)
	}

	return nil
}

func (s *PostgresStorage) Get(id string) (models.Task, error) {
	var task models.Task

	err := s.db.QueryRow(
		`SELECT id, code, translator, status, result
			FROM tasks
			WHERE id = $1;`,
		id,
	).Scan(&task.ID, &task.Code, &task.Translator, &task.Status, &task.Result)

	if errors.Is(err, sql.ErrNoRows) {
		return models.Task{}, fmt.Errorf("task %s not found: %w", id, ErrNotFound)
	} else if err != nil {
		return models.Task{}, fmt.Errorf("select task by id: %w", err)
	}

	return task, nil
}

func (s *PostgresStorage) SaveWithOutbox(
	task models.Task,
	message models.TaskMessage,
) error {

	payload, err := json.Marshal(message)

	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}

	tx, err := s.db.BeginTx(context.Background(), nil)

	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO tasks (id, code, translator, status, result)
		VALUES ($1, $2, $3, $4, $5)`,
		task.ID,
		task.Code,
		task.Translator,
		task.Status,
		task.Result,
	)
	if err != nil {
		return fmt.Errorf("failed to insert task: %w", err)
	}

	_, err = tx.Exec(
		`INSERT INTO outbox (task_id, payload)
		VALUES ($1, $2)`,
		task.ID,
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetPendingEvents(limit int) ([]OutboxEvent, error) {
	rows, err := s.db.Query(
		`SELECT id, task_id, payload
         FROM outbox
         WHERE published_at IS NULL
         ORDER BY id
         LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending outbox events: %w", err)
	}
	defer rows.Close()

	events := []OutboxEvent{}

	for rows.Next() {
		var event OutboxEvent

		if err := rows.Scan(
			&event.ID,
			&event.TaskID,
			&event.Payload,
		); err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate outbox events: %w", err)
	}

	return events, nil
}

func (s *PostgresStorage) MarkPublished(id int64) error {
	_, err := s.db.Exec(
		`UPDATE outbox
         SET published_at = NOW()
         WHERE id = $1`,
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to mark outbox event as published: %w", err)
	}

	return nil
}
