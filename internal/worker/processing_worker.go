package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/labib0x9/ffgif/internal/domain/cache"
	"github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/domain/processor"
	"github.com/labib0x9/ffgif/internal/domain/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoWorker struct {
	client    queue.Queue
	processor processor.VideoProcessor
	cache     cache.Cache
	gifRepo   media.GifRepository
	// uploaderRepo media.UploaderRepository
	maxRetries int
}

func NewVideoWorker(client queue.Queue, processor processor.VideoProcessor, cacheRepo cache.Cache, gifRepo media.GifRepository) *VideoWorker {
	return &VideoWorker{
		client:     client,
		processor:  processor,
		maxRetries: 2,
		cache:      cacheRepo,
		gifRepo:    gifRepo,
	}
}

func (w *VideoWorker) Run(ctx context.Context, name string, concurrency int) error {
	msgs, err := w.client.ConsumeVideo(ctx, name, concurrency)
	if err != nil {
		return err
	}
	defer w.client.CloseConsumerChannel(name)

	slog.Info("video worker started", "concurrency", concurrency)

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
	slog.Info("Inside Worker Msg Queue")
	var msg queue.VideoMessage
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		slog.Error("invalid video message", "error", err)
		d.Nack(false, false)
		return
	}

	slog.Info("processing video", "key", msg.Key, "userID", msg.UserID, "JobId", msg.JobId)
	key := "messaage_queue:job_id:" + msg.JobId
	if err := w.cache.Set(ctx, key, "processing", 5*time.Minute); err != nil {
		//
		return
	}

	gifId, err := w.processor.Process(ctx, msg.JobId, msg.Key, msg.Start, msg.End, msg.Width, msg.FPS, msg.Loop)
	gifKey := "messaage_queue_gif:job_id:" + msg.JobId
	if err := w.cache.Set(ctx, gifKey, gifKey, 5*time.Minute); err != nil {
		//
		return
	}
	if err := w.cache.Set(ctx, key, "processing", 5*time.Minute); err != nil {
		//
		return
	}
	if err != nil {
		// retries := retryCount(d)
		slog.Error("video processing failed", "error", err, "retries", msg.Retries, "JobId", msg.JobId)

		// retry
		if msg.Retries < w.maxRetries {
			msg.Retries++
			err := d.Nack(false, true)
			if err != nil {
				slog.Error("nack retry failed", "error", err)
			}
			return
		}

		// dead-letter
		err := d.Nack(false, false)
		if err != nil {
			slog.Error("nack dead-letter failed", "error", err)
		}
		return
	} else {
		if err := w.cache.Set(ctx, key, "completed", 5*time.Minute); err != nil {
			//
		}

		gif := media.Gif{
			Key:    gifId,
			UserId: msg.UserID,
			// Url:    w.gifRepo.GetUrl(gifId),
		}

		if err := w.gifRepo.Create(gif); err != nil {
			if err := w.cache.Set(ctx, key, "failed", 5*time.Minute); err != nil {
				//
			}
		}
	}

	err = d.Ack(false)
	if err != nil {
		slog.Error("ack failed", "error", err)
		return
	}

	slog.Info("video processed successfully", "JobId", msg.JobId, "gifId", gifId)
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
