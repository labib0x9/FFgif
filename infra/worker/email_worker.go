// package worker

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log/slog"

// 	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
// 	amqp "github.com/rabbitmq/amqp091-go"
// )

// type EmailSender interface {
// 	SendVerificationToken(email string, token string) error
// 	SendResetPassword(email string, token string) error
// 	SendResetNotification(email string) error
// }

// type EmailWorker struct {
// 	client     *rabbitmq.RabbitMQ
// 	mailer     EmailSender
// 	maxRetries int
// }

// func NewEmailWorker(client *rabbitmq.RabbitMQ, mailer EmailSender) *EmailWorker {
// 	return &EmailWorker{
// 		client:     client,
// 		mailer:     mailer,
// 		maxRetries: 3,
// 	}
// }

// func (w *EmailWorker) Run(ctx context.Context, concurrency int) error {
// 	if err := w.client.Channel.Qos(concurrency, 0, false); err != nil {
// 		slog.Error("EmailWorker run Qos", "err", err)
// 		return fmt.Errorf("qos: %w", err)
// 	}

// 	msgs, err := w.client.Channel.Consume(
// 		"email.queue",
// 		"email-worker",
// 		false,
// 		false, false, false, nil,
// 	)
// 	if err != nil {
// 		slog.Error("EmailWorker run consume", "err", err)
// 		return fmt.Errorf("consume: %w", err)
// 	}

// 	slog.Info("Msg Sender Run:", "worker", concurrency)

// 	sem := make(chan struct{}, concurrency)

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return nil
// 		case d, ok := <-msgs:
// 			if !ok {
// 				return fmt.Errorf("channel closed")
// 			}
// 			sem <- struct{}{}
// 			go func(d amqp.Delivery) {
// 				defer func() { <-sem }()
// 				w.handle(ctx, d)
// 			}(d)
// 		}
// 	}
// }

// func (w *EmailWorker) handle(ctx context.Context, d amqp.Delivery) {
// 	var msg rabbitmq.EmailMessage
// 	if err := json.Unmarshal(d.Body, &msg); err != nil {
// 		d.Nack(false, false)
// 		return
// 	}

// 	slog.Info("Msg Sender:", "service", msg.Name, "email", msg.To)

// 	var err error
// 	switch msg.Name {
// 	case "signup":
// 		err = w.mailer.SendVerificationToken(msg.To, msg.Token)
// 	case "forgot-password":
// 		err = w.mailer.SendResetPassword(msg.To, msg.Token)
// 	case "resend-verify":
// 		err = w.mailer.SendVerificationToken(msg.To, msg.Token)
// 	case "reset-password":
// 		err = w.mailer.SendResetNotification(msg.To)
// 	}

// 	if err != nil {
// 		retries := retryCount(d)

// 		if retries < w.maxRetries {
// 			d.Nack(false, true)
// 		} else {
// 			d.Nack(false, false)
// 		}
// 		return
// 	}

// 	d.Ack(false)
// }

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EmailSender interface {
	SendVerificationToken(email string, token string) error
	SendResetPassword(email string, token string) error
	SendResetNotification(email string) error
}

type EmailWorker struct {
	client     *rabbitmq.RabbitMQ
	mailer     EmailSender
	maxRetries int
}

func NewEmailWorker(
	client *rabbitmq.RabbitMQ,
	mailer EmailSender,
) *EmailWorker {
	return &EmailWorker{
		client:     client,
		mailer:     mailer,
		maxRetries: 3,
	}
}

func (w *EmailWorker) Run(
	ctx context.Context,
	concurrency int,
) error {

	// dedicated consumer channel
	ch, err := w.client.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	// important:
	// prevent one worker from taking too many jobs
	err = ch.Qos(concurrency, 0, false)
	if err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	msgs, err := ch.Consume(
		rabbitmq.EmailQueue,
		"email-worker",
		false, // auto ack
		false, // exclusive
		false, // no local
		false, // no wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	slog.Info(
		"email worker started",
		"concurrency", concurrency,
	)

	sem := make(chan struct{}, concurrency)

	for {
		select {

		case <-ctx.Done():
			slog.Info("email worker shutting down")
			return nil

		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("consumer channel closed")
			}

			sem <- struct{}{}

			go func(d amqp.Delivery) {
				defer func() {
					<-sem
				}()

				w.handle(d)

			}(d)
		}
	}
}

func (w *EmailWorker) handle(d amqp.Delivery) {

	var msg rabbitmq.EmailMessage

	if err := json.Unmarshal(d.Body, &msg); err != nil {

		slog.Error(
			"invalid email message",
			"error", err,
		)

		d.Nack(false, false)
		return
	}

	slog.Info(
		"processing email",
		"type", msg.Name,
		"email", msg.To,
	)

	var err error

	switch msg.Name {

	case "signup":
		err = w.mailer.SendVerificationToken(
			msg.To,
			msg.Token,
		)

	case "forgot-password":
		err = w.mailer.SendResetPassword(
			msg.To,
			msg.Token,
		)

	case "resend-verify":
		err = w.mailer.SendVerificationToken(
			msg.To,
			msg.Token,
		)

	case "reset-password":
		err = w.mailer.SendResetNotification(
			msg.To,
		)

	default:

		slog.Error(
			"unknown email job type",
			"type", msg.Name,
		)

		d.Nack(false, false)
		return
	}

	if err != nil {

		retries := retryCount(d)

		slog.Error(
			"email job failed",
			"error", err,
			"retries", retries,
		)

		// retry
		if retries < w.maxRetries {

			err := d.Nack(false, true)
			if err != nil {
				slog.Error(
					"nack retry failed",
					"error", err,
				)
			}

			return
		}

		// dead letter
		err := d.Nack(false, false)
		if err != nil {
			slog.Error(
				"nack dead-letter failed",
				"error", err,
			)
		}

		return
	}

	err = d.Ack(false)
	if err != nil {

		slog.Error(
			"ack failed",
			"error", err,
		)

		return
	}

	slog.Info(
		"email processed successfully",
		"email", msg.To,
	)
}
