package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	"github.com/labib0x9/ffgif/internal/infra/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SaveVideoWorker struct {
	client     *rabbitmq.RabbitMQ
	repo       media.LastVideoRepository
	minioRepo  media.UploaderRepository
	maxRetries int
}

func NewSaveVideoWorker(client *rabbitmq.RabbitMQ, repo media.LastVideoRepository, minioRepo media.UploaderRepository) *SaveVideoWorker {
	return &SaveVideoWorker{
		client:     client,
		repo:       repo,
		minioRepo:  minioRepo,
		maxRetries: 3,
	}
}

func (w *SaveVideoWorker) Run(ctx context.Context, concurrency int) error {
	ch, err := w.client.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.Qos(concurrency, 0, false)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		rabbitmq.SaveQueue,
		"save-video-worker",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	slog.Info("save video worker started")

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
	slog.Info("Inside Video Saver Worker")
	var msg queue.SaveVideoMessage
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		slog.Error("invalid save video message", "error", err)
		d.Nack(false, false)
		return
	}

	userID, err := uuid.Parse(msg.UserID)
	if err != nil {
		slog.Error("invalid user id", "user_id", msg.UserID)
		d.Nack(false, false)
		return
	}

	info, err := w.minioRepo.StatObject(ctx, msg.Key)
	if err != nil {
		// retries := retryCount(d)
		// slog.Warn(
		// 	"object not found yet",
		// 	"key", msg.Key,
		// 	"retries", retries,
		// )
		// delayed retry
		if msg.Retries < w.maxRetries {
			msg.Retries++
			if err := w.client.PublishRetrySaveVideo(ctx, msg); err != nil {
				slog.Error("publish retry failed", "error", err)
				d.Nack(false, true)
				return
			}
			d.Ack(false)
			slog.Error("publish retry", "error", err)
			return
		}
		slog.Error("StatObject failed", "error", err, "Key", msg.Key)
		// dead letter permanently
		d.Nack(false, false)
		return
	}

	upload := media.LastUpload{
		UserID:      userID,
		FileKey:     msg.Key,
		Filename:    msg.Filename,
		ContentType: info.ContentType,
		SizeBytes:   &info.Size,
		UploadedAt:  info.UploadedAt,
	}

	err = w.repo.Create(ctx, upload)
	if err != nil {
		// retries := retryCount(d)

		// slog.Error(
		// 	"save upload failed",
		// 	"error", err,
		// 	"retries", retries,
		// )

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

	err = d.Ack(false)
	if err != nil {
		slog.Error("ack failed", "error", err)
		return
	}

	slog.Info("video metadata saved", "key", msg.Key)
}
