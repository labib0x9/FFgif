package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/labib0x9/ffgif/internal/app/job"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/minio/minio-go/v7/pkg/notification"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ProcessingWorker struct {
	client     queue.Queue
	srv        job.Service
	maxRetries int
}

func NewProcessingWorker(srv job.Service, client queue.Queue) *ProcessingWorker {
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

	if msg.Err != nil {
		slog.Error("invalid notification message", "error", err)
		d.Nack(false, false)
		return
	}

	slog.Info("I AM INSIDE THe RAW WORKER")
	for _, record := range msg.Records {
		fmt.Println("RECORD ", record)
	}

	// slog.Info("processing video", "key", msg.Key, "userID", msg.UserID, "JobId", msg.JobId)

	// err = w.srv.Process(ctx, msg)
	// if err != nil {
	// 	slog.Error("video processing failed", "error", err, "retries", msg.Retries, "JobId", msg.JobId)

	// 	if msg.Retries < w.maxRetries {
	// 		msg.Retries++
	// 		err := d.Nack(false, true)
	// 		if err != nil {
	// 			slog.Error("nack retry failed", "error", err)
	// 		}
	// 		return
	// 	}

	// 	err := d.Nack(false, false)
	// 	if err != nil {
	// 		slog.Error("nack dead-letter failed", "error", err)
	// 	}
	// 	return
	// }

	// err = d.Ack(false)
	// if err != nil {
	// 	slog.Error("ack failed", "error", err)
	// 	return
	// }

	// slog.Info("video processed successfully", "JobId", msg.JobId)
}
