// package rabbitmq

// import (
// 	"fmt"
// 	"sync"

// 	amqp "github.com/rabbitmq/amqp091-go"
// )

// type RabbitMQ struct {
// 	Conn    *amqp.Connection
// 	Channel *amqp.Channel
// 	// Queue   amqp.Queue
// 	mu sync.Mutex
// }

// func NewRabbitMQ() *RabbitMQ {
// 	url := "amqp://guest:guest@localhost:5672/" // will come from .env

// 	conn, err := amqp.Dial(url)
// 	if err != nil {
// 		panic(err)
// 		// return nil
// 	}

// 	channel, err := conn.Channel()
// 	if err != nil {
// 		panic(err)
// 		// return nil
// 	}

// 	// if _, err := channel.QueueDeclare("email.queue.dead", true, false, false, false, nil); err != nil {
// 	// 	panic(err)
// 	// }

// 	// if _, err := channel.QueueDeclare("process.queue.dead", true, false, false, false, nil); err != nil {
// 	// 	panic(err)
// 	// }

// 	// if _, err := channel.QueueDeclare(
// 	// 	"email.queue",
// 	// 	true,
// 	// 	false,
// 	// 	false,
// 	// 	false,
// 	// 	amqp.Table{
// 	// 		"x-dead-letter-exchange":    "",
// 	// 		"x-dead-letter-routing-key": "email.queue.dead",
// 	// 		"x-message-ttl":             int32(300000),
// 	// 	},
// 	// ); err != nil {
// 	// 	panic(err)
// 	// }

// 	// if _, err = channel.QueueDeclare(
// 	// 	"process.queue",
// 	// 	true,
// 	// 	false,
// 	// 	false,
// 	// 	false,
// 	// 	amqp.Table{
// 	// 		"x-dead-letter-exchange":    "",
// 	// 		"x-dead-letter-routing-key": "process.queue.dead",
// 	// 	},
// 	// ); err != nil {
// 	// 	panic(err)
// 	// }

// 	if err := declareQueues(channel); err != nil {
// 		panic(err)
// 	}

// 	return &RabbitMQ{
// 		Conn:    conn,
// 		Channel: channel,
// 		// Queue:   queue,
// 	}
// }

// func (r *RabbitMQ) getChannel() (*amqp.Channel, error) {
// 	r.mu.Lock()
// 	defer r.mu.Unlock()

// 	// If channel is still open, reuse it
// 	if r.Channel != nil && !r.Channel.IsClosed() {
// 		return r.Channel, nil
// 	}

// 	// Check connection is still alive
// 	if r.Conn == nil || r.Conn.IsClosed() {
// 		return nil, fmt.Errorf("rabbitmq connection is closed")
// 	}

// 	// Reopen the channel
// 	ch, err := r.Conn.Channel()
// 	if err != nil {
// 		return nil, fmt.Errorf("reopen channel: %w", err)
// 	}

// 	// Re-declare queues on the new channel
// 	if err := declareQueues(ch); err != nil {
// 		ch.Close()
// 		return nil, err
// 	}

// 	r.Channel = ch
// 	return r.Channel, nil
// }

// func declareQueues(ch *amqp.Channel) error {
// 	for _, dlq := range []string{"email.queue.dead", "process.queue.dead"} {
// 		if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
// 			return fmt.Errorf("declare %s: %w", dlq, err)
// 		}
// 	}

// 	if _, err := ch.QueueDeclare(
// 		"email.queue", true, false, false, false,
// 		amqp.Table{
// 			"x-dead-letter-exchange":    "",
// 			"x-dead-letter-routing-key": "email.queue.dead",
// 			"x-message-ttl":             int32(300000),
// 		},
// 	); err != nil {
// 		return fmt.Errorf("declare email.queue: %w", err)
// 	}

// 	if _, err := ch.QueueDeclare(
// 		"process.queue", true, false, false, false,
// 		amqp.Table{
// 			"x-dead-letter-exchange":    "",
// 			"x-dead-letter-routing-key": "process.queue.dead",
// 		},
// 	); err != nil {
// 		return fmt.Errorf("declare process.queue: %w", err)
// 	}

// 	return nil
// }

// // func (r *RabbitMQ) Publish(message model.EmailMessage) error {
// // 	rawMsg, err := json.Marshal(message)
// // 	if err != nil {
// // 		return err
// // 	}
// // 	err = r.Channel.Publish(
// // 		"",
// // 		r.Queue.Name,
// // 		false,
// // 		false,
// // 		amqp.Publishing{
// // 			ContentType: "text/plain",
// // 			Body:        rawMsg,
// // 		},
// // 	)
// // 	return err
// // }

// func (r *RabbitMQ) Close() {
// 	if r.Channel != nil {
// 		r.Channel.Close()
// 	}
// 	if r.Conn != nil {
// 		r.Conn.Close()
// 	}
// }

package rabbitmq

import (
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	EmailQueue   = "email.queue"
	ProcessQueue = "process.queue"
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
