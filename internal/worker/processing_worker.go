package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/labib0x9/ffgif/internal/app/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/minio/minio-go/v7/pkg/notification"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ProcessingWorker struct {
	client     queue.Queue
	srv        media.Service
	maxRetries int
}

func NewProcessingWorker(srv media.Service, client queue.Queue) *ProcessingWorker {
	return &ProcessingWorker{
		srv:        srv,
		client:     client,
		maxRetries: 2,
	}
}

func (w *ProcessingWorker) Run(ctx context.Context, name string, concurrency int) error {
	msgs, err := w.client.ConsumeRawVideo(ctx, name, concurrency)
	if err != nil {
		return err
	}
	defer w.client.CloseConsumerChannel(name)

	slog.Info("Raw Video Processing worker started", "concurrency", concurrency)
	sem := make(chan struct{}, concurrency)
	for {
		select {
		case <-ctx.Done():
			slog.Info("raw video worker shutting down")
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

func (w *ProcessingWorker) handle(ctx context.Context, d amqp.Delivery) {
	var msg notification.Info
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		slog.Error("invalid notification message", "error", err)
		d.Nack(false, false)
		return
	}

	if msg.Err != nil || len(msg.Records) == 0 {
		slog.Error("invalid notification message", "error", err)
		d.Nack(false, false)
		return
	}

	key := msg.Records[0].S3.Object.Key
	decodedKey, err := url.QueryUnescape(key)
	if err != nil {
		slog.Error("Pre Processing Handler() decode key failed", "error", err)
		return
	}
	key = decodedKey

	err = w.srv.UpdateUploadingStatus(ctx, key, "processing")
	if err != nil {
		slog.Error("UpdateUploadingStatus() failed", "error", err)
		return
	}
	slog.Info("processing raw video", "key", key)

	err = w.srv.ProcessAndSave(ctx, key)
	if err != nil {
		slog.Error("raw video processing failed", "key", key, "error", err)
		w.srv.UpdateUploadingStatus(ctx, key, "failed")
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

	err = w.srv.UpdateUploadingStatus(ctx, key, "ok")
	if err != nil {
		slog.Error("UpdateUploadingStatus() failed", "error", err)
		return
	}

	slog.Info("raw video processed successfully", "key", key)
}
