package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/labib0x9/ffgif/internal/app/media"
	jobdomain "github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SaveVideoWorker struct {
	client     queue.Queue
	srv        media.Service
	maxRetries int
}

func NewSaveVideoWorker(client queue.Queue, srv media.Service) *SaveVideoWorker {
	return &SaveVideoWorker{
		client:     client,
		srv:        srv,
		maxRetries: 3,
	}
}

func (w *SaveVideoWorker) Run(ctx context.Context, name string, concurrency int) error {
	msgs, err := w.client.ConsumeSave(ctx, name, concurrency)
	if err != nil {
		return err
	}
	defer w.client.CloseConsumerChannel(name)

	slog.Info("Video saver worker started")

	sem := make(chan struct{}, concurrency)
	for {
		select {
		case <-ctx.Done():
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
				w.handle(ctx, d)
			}(d)
		}
	}
}

func (w *SaveVideoWorker) handle(ctx context.Context, d amqp.Delivery) {
	var msg queue.SaveVideoMessage
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		slog.Error("invalid save video message", "error", err)
		d.Nack(false, false)
		return
	}

	err = w.srv.SaveMetadata(ctx, msg)
	if err != nil {
		switch {
		case errors.Is(err, jobdomain.ErrInvalidUserID):
			slog.Error("invalid user id", "user_id", msg.UserID, "key", msg.Key)
			d.Nack(false, false)
			return
		default:
			slog.Error("save video metadata failed", "error", err, "key", msg.Key, "user_id", msg.UserID)
			d.Nack(false, false)
			return
		}
	}

	err = d.Ack(false)
	if err != nil {
		slog.Error("ack failed", "error", err, "key", msg.Key)
		return
	}

	slog.Info("video metadata saved", "key", msg.Key, "user_id", msg.UserID)
}
