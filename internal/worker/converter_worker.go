package worker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/labib0x9/ffgif/internal/app/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoWorker struct {
	client     queue.Queue
	srv        media.Service
	maxRetries int
}

func NewVideoWorker(srv media.Service, client queue.Queue) *VideoWorker {
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

func (w *VideoWorker) handle(ctx context.Context, d amqp.Delivery) {
	var msg queue.VideoMessage
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		slog.Error("invalid video message", "error", err)
		d.Nack(false, false)
		return
	}

	slog.Info("converting video", "key", msg.Key, "user_id", msg.UserID, "job_id", msg.JobId)

	err = w.srv.Process(ctx, msg)
	if err != nil {
		slog.Error("video conversion failed", "error", err, "job_id", msg.JobId)
		// if retryCount(d.Headers) < int64(w.maxRetries) {
		// 	_ = d.Nack(false, true)
		// } else {
		_ = d.Nack(false, false)
		// }
		return
	}

	err = d.Ack(false)
	if err != nil {
		slog.Error("ack failed", "error", err, "job_id", msg.JobId)
		return
	}

	slog.Info("video processed successfully", "job_id", msg.JobId)
}

func retryCount(headers amqp.Table) int64 {
	if headers == nil {
		return 0
	}
	raw, ok := headers["x-death"]
	if !ok || raw == nil {
		return 0
	}
	var firstEntry any
	switch v := raw.(type) {
	case []any:
		if len(v) > 0 {
			firstEntry = v[0]
		}
	case []amqp.Table:
		if len(v) > 0 {
			firstEntry = v[0]
		}
	case []map[string]any:
		if len(v) > 0 {
			firstEntry = v[0]
		}
	}

	if firstEntry == nil {
		return 0
	}

	switch entry := firstEntry.(type) {
	case amqp.Table:
		if c, ok := entry["count"].(int64); ok {
			return c
		}
		if c, ok := entry["count"].(int); ok {
			return int64(c)
		}
	case map[string]any:
		if c, ok := entry["count"].(int64); ok {
			return c
		}
		if c, ok := entry["count"].(int); ok {
			return int64(c)
		}
	}
	return 0
}
