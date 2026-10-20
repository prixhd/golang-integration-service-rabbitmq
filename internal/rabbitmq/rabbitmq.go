package rabbitmq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	PublishTask(body []byte) error
}

type AMQPPublisher struct {
	ch *amqp.Channel
}

func NewPublisher(ch *amqp.Channel) *AMQPPublisher {
	return &AMQPPublisher{
		ch: ch,
	}
}

func (p *AMQPPublisher) PublishTask(body []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := p.ch.PublishWithContext(
		ctx,
		"",
		"tasks",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})

	return err
}

func NewConnection(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url)

	if err != nil {
		return nil, err
	}

	return conn, nil
}

func DeclareQueue(ch *amqp.Channel) (amqp.Queue, error) {
	q, err := ch.QueueDeclare(
		"tasks",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return amqp.Queue{}, err
	}

	return q, nil

}
