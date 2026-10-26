package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	PublishTask(body []byte) error
}

type AMQPPublisher struct {
	mu   sync.Mutex
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel

	done chan struct{}
	wg   sync.WaitGroup
}

func NewPublisher(url string) (*AMQPPublisher, error) {
	conn, err := NewConnection(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	_, err = DeclareQueue(ch)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	p := &AMQPPublisher{
		url:  url,
		conn: conn,
		ch:   ch,
		done: make(chan struct{}),
	}

	p.wg.Add(1)
	go p.watchConnection()

	return p, nil
}

func (p *AMQPPublisher) PublishTask(body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch == nil || p.ch.IsClosed() {
		return fmt.Errorf("RabbitMQ publisher is unavailable")
	}

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

func (p *AMQPPublisher) reconnect() error {
	conn, err := NewConnection(p.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if _, err := DeclareQueue(ch); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	p.mu.Lock()

	oldConn := p.conn
	oldCh := p.ch

	p.conn = conn
	p.ch = ch

	p.mu.Unlock()

	if oldCh != nil {
		oldCh.Close()
	}
	if oldConn != nil {
		oldConn.Close()
	}

	return nil
}

func (p *AMQPPublisher) watchConnection() {
	defer p.wg.Done()

	for {
		p.mu.Lock()

		conn := p.conn
		ch := p.ch

		connClosed := conn.NotifyClose(make(chan *amqp.Error, 1))
		chClosed := ch.NotifyClose(make(chan *amqp.Error, 1))

		p.mu.Unlock()

		select {
		case <-p.done:
			return
		case <-connClosed:
			log.Println("RabbitMQ connection closed, reconnecting...")
		case <-chClosed:
			log.Println("RabbitMQ channel closed, reconnecting...")
		}

		// не начнем переподключение, если Publisher закрывается
		select {
		case <-p.done:
			return
		default:
		}

		p.mu.Lock()
		oldConn := p.conn
		oldCh := p.ch
		p.conn = nil
		p.ch = nil
		p.mu.Unlock()

		if oldCh != nil {
			oldCh.Close()
		}
		if oldConn != nil {
			oldConn.Close()
		}

		for {
			select {
			case <-p.done:
				return
			default:
			}

			if err := p.reconnect(); err == nil {
				log.Println("RabbitMQ reconnected successfully")
				break
			} else {
				log.Printf("RabbitMQ reconnect failed: %v", err)
			}

			timer := time.NewTimer(2 * time.Second)

			select {
			case <-p.done:
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}

func (p *AMQPPublisher) Close() error {
	close(p.done)

	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch != nil {
		p.ch.Close()
		p.ch = nil
	}

	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}

	return nil
}
