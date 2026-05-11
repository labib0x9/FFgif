package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	"github.com/labib0x9/ProjectUnsafe/model"
	"github.com/labib0x9/ProjectUnsafe/repo"

	amqp "github.com/rabbitmq/amqp091-go"
)

// type LastUploadSaver interface {
// 	Create(
// 		ctx context.Context,
// 		upload model.LastUpload,
// 	) error
// }

type SaveVideoWorker struct {
	client     *rabbitmq.RabbitMQ
	repo       repo.LastVideoRepository
	minioRepo  repo.UploaderRepository
	maxRetries int
}

func NewSaveVideoWorker(client *rabbitmq.RabbitMQ, repo repo.LastVideoRepository, minioRepo repo.UploaderRepository) *SaveVideoWorker {
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

	var msg rabbitmq.SaveVideoMessage

	err := json.Unmarshal(d.Body, &msg)
	if err != nil {

		slog.Error(
			"invalid save video message",
			"error", err,
		)

		d.Nack(false, false)
		return
	}

	userID, err := uuid.Parse(msg.UserID)
	if err != nil {

		slog.Error(
			"invalid user id",
			"user_id", msg.UserID,
		)

		d.Nack(false, false)
		return
	}

	info, err := w.minioRepo.StatObject(ctx, msg.Key)
	if err != nil {

		retries := retryCount(d)

		slog.Warn(
			"object not found yet",
			"key", msg.Key,
			"retries", retries,
		)

		// delayed retry
		if retries < w.maxRetries {

			err := w.client.PublishRetrySaveVideo(
				ctx,
				msg,
			)
			if err != nil {

				slog.Error(
					"publish retry failed",
					"error", err,
				)

				d.Nack(false, true)
				return
			}

			d.Ack(false)
			return
		}

		// dead letter permanently
		d.Nack(false, false)
		return
	}

	upload := model.LastUpload{
		UserID:      userID,
		FileKey:     msg.Key,
		Filename:    msg.Filename,
		ContentType: info.ContentType,
		SizeBytes:   &info.Size,
		UploadedAt:  info.UploadedAt,
	}

	err = w.repo.Create(ctx, upload)
	if err != nil {

		retries := retryCount(d)

		slog.Error(
			"save upload failed",
			"error", err,
			"retries", retries,
		)

		if retries < w.maxRetries {

			d.Nack(false, true)
			return
		}

		d.Nack(false, false)
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
		"video metadata saved",
		"key", msg.Key,
	)
}
