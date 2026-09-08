package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/labib0x9/ffgif/internal/app/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/telemetry"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	// Extract distributed trace context from AMQP headers
	ctx = telemetry.ExtractAMQPHeaders(ctx, d.Headers)
	tracer := telemetry.Tracer("ffgif-worker")
	ctx, span := tracer.Start(
		ctx,
		"worker.convert_video",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	telemetry.WorkerActiveTasks.WithLabelValues("convert-worker").Inc()
	defer telemetry.WorkerActiveTasks.WithLabelValues("convert-worker").Dec()

	start := time.Now()
	defer func() {
		telemetry.WorkerTaskDuration.WithLabelValues("convert-worker").Observe(time.Since(start).Seconds())
	}()

	var msg queue.VideoMessage
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid video message")
		telemetry.WorkerTasksTotal.WithLabelValues("convert-worker", "failure").Inc()
		slog.ErrorContext(ctx, "invalid video message", "error", err)
		d.Nack(false, false)
		return
	}

	span.SetAttributes(
		attribute.String("job_id", msg.JobId),
		attribute.String("key", msg.Key),
		attribute.String("user_id", msg.UserID),
	)

	slog.InfoContext(ctx, "converting video", "key", msg.Key, "user_id", msg.UserID, "job_id", msg.JobId)

	err = w.srv.Process(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "video conversion failed")
		telemetry.WorkerTasksTotal.WithLabelValues("convert-worker", "failure").Inc()
		slog.ErrorContext(ctx, "video conversion failed", "error", err, "job_id", msg.JobId)

		nackErr := d.Nack(false, false)
		if nackErr != nil {
			slog.ErrorContext(ctx, "nack dead-letter failed", "error", nackErr, "job_id", msg.JobId)
		}
		return
	}

	err = d.Ack(false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ack failed")
		telemetry.WorkerTasksTotal.WithLabelValues("convert-worker", "failure").Inc()
		slog.ErrorContext(ctx, "ack failed", "error", err, "job_id", msg.JobId)
		return
	}

	span.SetStatus(codes.Ok, "OK")
	telemetry.WorkerTasksTotal.WithLabelValues("convert-worker", "success").Inc()
	telemetry.MediaConversionsTotal.WithLabelValues("gif", "success").Inc()
	slog.InfoContext(ctx, "video processed successfully", "job_id", msg.JobId)
}
