package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/labib0x9/ffgif/internal/app/job"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoWorker struct {
	client     queue.Queue
	srv        job.Service
	maxRetries int
}

func NewVideoWorker(srv job.Service, client queue.Queue) *VideoWorker {
	return &VideoWorker{
		srv:        srv,
		client:     client,
		maxRetries: 2,
	}
}

func (w *VideoWorker) Run(ctx context.Context, name string, concurrency int) error {
	msgs, err := w.client.ConsumeVideo(ctx, name, concurrency)
	if err != nil {
		return err
	}
	defer w.client.CloseConsumerChannel(name)

	slog.Info("Video Processing worker started", "concurrency", concurrency)
	sem := make(chan struct{}, concurrency)
	for {
		select {
		case <-ctx.Done():
			slog.Info("video worker shutting down")
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
				w.handle(ctx, d)
			}(d)
		}
	}
}

func (w *VideoWorker) handle(ctx context.Context, d amqp.Delivery) {
	var msg queue.VideoMessage
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		slog.Error("invalid video message", "error", err)
		d.Nack(false, false)
		return
	}

	slog.Info("processing video", "key", msg.Key, "userID", msg.UserID, "JobId", msg.JobId)

	err = w.srv.Process(ctx, msg)
	if err != nil {
		slog.Error("video processing failed", "error", err, "retries", msg.Retries, "JobId", msg.JobId)

		if msg.Retries < w.maxRetries {
			msg.Retries++
			err := d.Nack(false, true)
			if err != nil {
				slog.Error("nack retry failed", "error", err)
			}
			return
		}

		err := d.Nack(false, false)
		if err != nil {
			slog.Error("nack dead-letter failed", "error", err)
		}
		return
	}

	err = d.Ack(false)
	if err != nil {
		slog.Error("ack failed", "error", err)
		return
	}

	slog.Info("video processed successfully", "JobId", msg.JobId)
}

func retryCount(d amqp.Delivery) int {
	deaths, ok := d.Headers["x-death"].([]interface{})
	if !ok || len(deaths) == 0 {
		return 0
	}
	entry, ok := deaths[0].(amqp.Table)
	if !ok {
		return 0
	}
	count, _ := entry["count"].(int64)
	return int(count)
}
