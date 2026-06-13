package rabbitmq

import (
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	EmailQueue     = "email.queue"
	ProcessQueue   = "process.queue"
	SaveQueue      = "video.save.queue"
	SaveRetryQueue = "save.retry.queue"
)

type RabbitMQ struct {
	conn *amqp.Connection
}

func NewRabbitMQ() *RabbitMQ {
	url := "amqp://guest:guest@localhost:5672/" // will come from .env
	conn, err := amqp.Dial(url)
	if err != nil {
		panic(fmt.Errorf("rabbitmq dial: %w", err))
		return nil
	}

	r := &RabbitMQ{
		conn: conn,
	}

	if err := r.setup(); err != nil {
		conn.Close()
		panic(err)
		return nil
	}

	slog.Info("rabbitmq connected")

	return r
}

func (r *RabbitMQ) setup() error {
	ch, err := r.conn.Channel()
	if err != nil {
		return fmt.Errorf("setup channel: %w", err)
	}
	defer ch.Close()

	// dead letter queues

	_, err = ch.QueueDeclare(
		"email.queue.dead",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare email dlq: %w", err)
	}

	_, err = ch.QueueDeclare(
		"process.queue.dead",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare process dlq: %w", err)
	}

	_, err = ch.QueueDeclare(
		"save.queue.dead",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare save dlq: %w", err)
	}

	// main queues

	_, err = ch.QueueDeclare(
		EmailQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": "email.queue.dead",
		},
	)
	if err != nil {
		return fmt.Errorf("declare email queue: %w", err)
	}

	_, err = ch.QueueDeclare(
		ProcessQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": "process.queue.dead",
		},
	)
	if err != nil {
		return fmt.Errorf("declare process queue: %w", err)
	}

	_, err = ch.QueueDeclare(
		SaveQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": "save.queue.dead",
		},
	)
	if err != nil {
		return fmt.Errorf("declare save queue: %w", err)
	}

	_, err = ch.QueueDeclare(
		SaveRetryQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-message-ttl":             int32(1000),
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": SaveQueue,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *RabbitMQ) Channel() (*amqp.Channel, error) {
	return r.conn.Channel()
}

func (r *RabbitMQ) Close() error {
	if r.conn != nil && !r.conn.IsClosed() {
		return r.conn.Close()
	}

	return nil
}
