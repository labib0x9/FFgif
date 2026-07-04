package rabbitmq

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/labib0x9/ffgif/config"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	EmailQueue         = "email.queue"
	ProcessQueue       = "process.queue"
	SaveQueue          = "video.save.queue"
	SaveRetryQueue     = "save.retry.queue"
	UploadProcessQueue = "process.upload.queue"
)

type rabbitMQ struct {
	conn       *amqp.Connection
	consumerCh sync.Map // for each consumer a dedicated channel
}

func NewRabbitMQ(cnf *config.RabbitMq) queue.Queue {
	url := fmt.Sprintf("amqp://%s:%s@%s/", cnf.User, cnf.Pass, cnf.Addr)
	conn, err := amqp.Dial(url)
	if err != nil {
		panic(fmt.Errorf("rabbitmq dial: %w, url=%s", err, url))
	}

	r := rabbitMQ{
		conn: conn,
	}

	slog.Info("rabbitMq connection complete")
	return &r
}

func Setup(q queue.Queue, cnf *config.Minio) error {
	r, ok := q.(*rabbitMQ)
	if !ok {
		return fmt.Errorf("type not matched")
	}
	ch, err := r.conn.Channel()
	if err != nil {
		return fmt.Errorf("setup channel: %w", err)
	}
	defer ch.Close()

	if err := r.declareEmailQueueDead(ch); err != nil {
		return err
	}

	if err := r.declareEmailQueue(ch); err != nil {
		return err
	}

	if err := r.declareProcessQueueDead(ch); err != nil {
		return err
	}

	if err := r.declareProcessQueue(ch); err != nil {
		return err
	}

	if err := r.declareSaveQueueDead(ch); err != nil {
		return err
	}

	if err := r.declareSaveQueue(ch); err != nil {
		return err
	}

	if err := r.declareSaveRetryQueueDead(ch); err != nil {
		return err
	}

	if err := r.declareSaveRetryQueue(ch); err != nil {
		return err
	}

	if err := r.declareUploadProcessQueueDead(ch); err != nil {
		return err
	}

	if err := r.declareUploadProcessQueue(ch, cnf); err != nil {
		return err
	}

	slog.Info("RabbitMq setup complete")
	return nil
}

func (r *rabbitMQ) declareEmailQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		EmailQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": getDeadQueue(EmailQueue),
		},
	)
	return err
}

func (r *rabbitMQ) declareEmailQueueDead(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		getDeadQueue(EmailQueue),
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (r *rabbitMQ) declareProcessQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		ProcessQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": getDeadQueue(ProcessQueue),
		},
	)
	return err
}

func (r *rabbitMQ) declareProcessQueueDead(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		getDeadQueue(ProcessQueue),
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (r *rabbitMQ) declareSaveQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		SaveQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": getDeadQueue(SaveQueue),
		},
	)
	return err
}

func (r *rabbitMQ) declareSaveQueueDead(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		getDeadQueue(SaveQueue),
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (r *rabbitMQ) declareSaveRetryQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
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
	return err
}

func (r *rabbitMQ) declareSaveRetryQueueDead(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		getDeadQueue(SaveRetryQueue),
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (r *rabbitMQ) declareUploadProcessQueue(ch *amqp.Channel, cnf *config.Minio) error {
	if err := ch.ExchangeDeclare(
		cnf.ExchangeQueue,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return err
	}
	q, err := ch.QueueDeclare(
		UploadProcessQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": getDeadQueue(UploadProcessQueue),
		},
	)
	if err != nil {
		return err
	}
	return ch.QueueBind(q.Name, "", cnf.ExchangeQueue, false, nil)
}

func (r *rabbitMQ) declareUploadProcessQueueDead(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		getDeadQueue(UploadProcessQueue),
		true,
		false,
		false,
		false,
		nil,
	)
	return err
}

func (r *rabbitMQ) channel() (*amqp.Channel, error) {
	return r.conn.Channel()
}

func (r *rabbitMQ) CloseConsumerChannel(name string) error {
	val, ok := r.consumerCh.Load(name)
	if !ok {
		return fmt.Errorf("error on fetching consumer")
	}
	ch := val.(*amqp.Channel)

	if err := ch.Close(); err != nil {
		return err
	}
	r.consumerCh.Delete(name)
	return nil
}

func (r *rabbitMQ) Close() error {
	r.consumerCh.Range(func(key, value any) bool {
		ch := value.(*amqp.Channel)
		if ch != nil {
			ch.Close()
		}
		return true // continue
	})

	if r.conn != nil && !r.conn.IsClosed() {
		return r.conn.Close()
	}
	return nil
}

func getDeadQueue(queue string) string {
	return queue + ".dead"
}
