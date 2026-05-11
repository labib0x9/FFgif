package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// func (r *RabbitMQ) PublishEmail(ctx context.Context, msg EmailMessage) error {
// 	return r.publish(ctx, "email.queue", msg)
// }

// func (r *RabbitMQ) PublishVideo(ctx context.Context, msg VideoMessage) error {
// 	return r.publish(ctx, "process.queue", msg)
// }

// func (r *RabbitMQ) publish(ctx context.Context, queue string, payload any) error {
// 	ch, err := r.getChannel()
// 	if err != nil {
// 		slog.Error("publish getChannel()", "error", err)
// 		return fmt.Errorf("get channel: %w", err)
// 	}

// 	body, err := json.Marshal(payload)
// 	if err != nil {
// 		slog.Error("publish marshal()", "error", err)
// 		return fmt.Errorf("marshal: %w", err)
// 	}

// 	slog.Info("publish", "queue", queue)

// 	err = ch.PublishWithContext(ctx,
// 		"",
// 		queue,
// 		false,
// 		false,
// 		amqp.Publishing{
// 			ContentType:  "application/json",
// 			DeliveryMode: amqp.Persistent,
// 			Timestamp:    time.Now(),
// 			Body:         body,
// 		},
// 	)
// 	if err != nil {
// 		slog.Error("publish publishWithContext()", "error", err)
// 	}
// 	return err
// }

func (r *RabbitMQ) PublishEmail(ctx context.Context, msg EmailMessage) error {
	return r.publish(ctx, EmailQueue, msg)
}

func (r *RabbitMQ) PublishVideo(ctx context.Context, msg VideoMessage) error {
	return r.publish(ctx, ProcessQueue, msg)
}

func (r *RabbitMQ) PublishSaveVideo(ctx context.Context, msg SaveVideoMessage) error {
	return r.publish(ctx, SaveQueue, msg)
}

func (r *RabbitMQ) publish(ctx context.Context, queue string, payload any) error {

	// create fresh channel per publish
	// safer than shared channel

	ch, err := r.conn.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(
		ctx,
		"", // default exchange
		queue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)

	if err != nil {
		return fmt.Errorf("publish message: %w", err)
	}

	slog.Info(
		"message published",
		"queue", queue,
	)

	return nil
}

func (r *RabbitMQ) PublishRetrySaveVideo(ctx context.Context, msg SaveVideoMessage) error {
	return r.publish(
		ctx,
		SaveRetryQueue,
		msg,
	)
}
