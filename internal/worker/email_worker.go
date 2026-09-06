package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/labib0x9/ffgif/internal/port/mailer"
	"github.com/labib0x9/ffgif/internal/port/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type EmailWorker struct {
	client     queue.Queue
	mailer     mailer.EmailSender
	maxRetries int
}

func NewEmailWorker(client queue.Queue, mailer mailer.EmailSender) *EmailWorker {
	return &EmailWorker{
		client:     client,
		mailer:     mailer,
		maxRetries: 3,
	}
}

func (w *EmailWorker) Run(ctx context.Context, name string, concurrency int) error {
	msgs, err := w.client.ConsumeEmail(ctx, name, concurrency)
	if err != nil {
		return err
	}
	defer w.client.CloseConsumerChannel(name)
	slog.Info("Email worker started", "concurrency", concurrency)

	sem := make(chan struct{}, concurrency)
	for {
		select {

		case <-ctx.Done():
			slog.Info("Email worker shutting down")
			return nil

		case d, ok := <-msgs:
			if !ok {
				return queue.ErrConsumerChannelClosed
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

	var msg queue.EmailMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		slog.Error("invalid email message", "error", err)
		d.Nack(false, false)
		return
	}

	slog.Info("processing email", "type", msg.Name, "email", msg.To)

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
		slog.Error("unknown email job type", "type", msg.Name, "email", msg.To)
		d.Nack(false, false)
		return
	}

	if err != nil {
		slog.Error("email sending failed", "error", err, "type", msg.Name, "email", msg.To)
		err := d.Nack(false, false)
		if err != nil {
			slog.Error("nack dead-letter failed", "error", err, "email", msg.To)
		}

		return
	}

	err = d.Ack(false)
	if err != nil {
		slog.Error("ack failed", "error", err, "email", msg.To)
		return
	}

	slog.Info("email processed successfully", "email", msg.To)
}
