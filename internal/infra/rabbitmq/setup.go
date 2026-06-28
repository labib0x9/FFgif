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
	EmailQueue              = "email.queue"
	ProcessQueue            = "process.queue"
	SaveQueue               = "video.save.queue"
	SaveRetryQueue          = "save.retry.queue"
	UploadNotificationQueue = "notify.upload.queue"
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

	if err := r.setup(); err != nil {
		conn.Close()
		panic(err)
	}

	slog.Info("rabbitMq connection complete")

	return &r
}

func (r *rabbitMQ) setup() error {
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

	if err := r.declareUploadNotificationQueueDead(ch); err != nil {
		return err
	}

	if err := r.declareUploadNotificationQueue(ch); err != nil {
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

func (r *rabbitMQ) declareUploadNotificationQueue(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		UploadNotificationQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": getDeadQueue(UploadNotificationQueue),
		},
	)
	return err
}

func (r *rabbitMQ) declareUploadNotificationQueueDead(ch *amqp.Channel) error {
	_, err := ch.QueueDeclare(
		getDeadQueue(UploadNotificationQueue),
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
