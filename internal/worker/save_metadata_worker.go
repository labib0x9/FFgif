package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/labib0x9/ffgif/internal/app/job"
	jobdomain "github.com/labib0x9/ffgif/internal/domain/job"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SaveVideoWorker struct {
	client     queue.Queue
	srv        job.Service
	maxRetries int
}

func NewSaveVideoWorker(client queue.Queue, srv job.Service) *SaveVideoWorker {
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
				return fmt.Errorf("consumer closed")
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
		case errors.Is(jobdomain.ErrInvalidUserID, err):
			{
				slog.Error("invalid user id", "user_id", msg.UserID)
				d.Nack(false, false)
				return
			}
		default:
			{
				if msg.Retries < w.maxRetries {
					msg.Retries++
					d.Nack(false, true)
					slog.Error("Create retry", "error", err)
					return
				}
				slog.Error("Create error", "error", err)
				d.Nack(false, false)
				return
			}
		}
	}

	err = d.Ack(false)
	if err != nil {
		slog.Error("ack failed", "error", err)
		return
	}

	slog.Info("video metadata saved", "key", msg.Key)
}
