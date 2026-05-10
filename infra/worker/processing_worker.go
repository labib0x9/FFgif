package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/labib0x9/ProjectUnsafe/infra/queue/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type VideoProcessor interface {
	Process(
		ctx context.Context,
		videoID string,
		inputPath string,
		outputPath string,
		formats []string,
	) error
}

type VideoWorker struct {
	client     *rabbitmq.RabbitMQ
	processor  VideoProcessor
	maxRetries int
}

func NewVideoWorker(
	client *rabbitmq.RabbitMQ,
	processor VideoProcessor,
) *VideoWorker {

	return &VideoWorker{
		client:     client,
		processor:  processor,
		maxRetries: 2,
	}
}

func (w *VideoWorker) Run(
	ctx context.Context,
	concurrency int,
) error {

	// dedicated consumer channel
	ch, err := w.client.Channel()
	if err != nil {
		return fmt.Errorf("open channel: %w", err)
	}
	defer ch.Close()

	// limit unacked messages
	err = ch.Qos(concurrency, 0, false)
	if err != nil {
		return fmt.Errorf("qos: %w", err)
	}

	msgs, err := ch.Consume(
		rabbitmq.ProcessQueue,
		"video-worker",
		false, // auto ack
		false, // exclusive
		false, // no local
		false, // no wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	slog.Info(
		"video worker started",
		"concurrency", concurrency,
	)

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

func (w *VideoWorker) handle(
	ctx context.Context,
	d amqp.Delivery,
) {

	var msg rabbitmq.VideoMessage

	err := json.Unmarshal(d.Body, &msg)
	if err != nil {

		slog.Error(
			"invalid video message",
			"error", err,
		)

		d.Nack(false, false)
		return
	}

	slog.Info(
		"processing video",
		"video_id", msg.VideoID,
		"input", msg.InputPath,
	)

	err = w.processor.Process(
		ctx,
		msg.VideoID,
		msg.InputPath,
		msg.OutputPath,
		msg.Formats,
	)

	if err != nil {

		retries := retryCount(d)

		slog.Error(
			"video processing failed",
			"error", err,
			"retries", retries,
			"video_id", msg.VideoID,
		)

		// retry
		if retries < w.maxRetries {

			err := d.Nack(false, true)
			if err != nil {
				slog.Error(
					"nack retry failed",
					"error", err,
				)
			}

			return
		}

		// dead-letter
		err := d.Nack(false, false)
		if err != nil {

			slog.Error(
				"nack dead-letter failed",
				"error", err,
			)
		}

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
		"video processed successfully",
		"video_id", msg.VideoID,
	)
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
