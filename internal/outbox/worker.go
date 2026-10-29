package outbox

import (
	"context"
	"log"
	"time"

	"go-integration-service-rabbitmq/internal/rabbitmq"
	"go-integration-service-rabbitmq/internal/storage"
)

type Worker struct {
	store     storage.OutboxStorage
	publisher rabbitmq.Publisher
}

func NewWorker(
	store storage.OutboxStorage,
	publisher rabbitmq.Publisher,
) *Worker {
	return &Worker{
		store:     store,
		publisher: publisher,
	}
}

func (w *Worker) processBatch() {
	events, err := w.store.GetPendingEvents(10)

	if err != nil {
		log.Printf("failed to get pending outbox events: %v", err)
		return
	}

	for _, event := range events {
		if err := w.publisher.PublishTask(event.Payload); err != nil {
			log.Printf(
				"failed to publish outbox event %d: %v",
				event.ID,
				err,
			)
			return
		}

		if err := w.store.MarkPublished(event.ID); err != nil {
			log.Printf(
				"failed to mark outbox event %d as published: %v",
				event.ID,
				err,
			)
			return
		}
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			w.processBatch()
		}
	}
}
