package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"time"

	"github.com/labib0x9/ffgif/internal/app/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/telemetry"
	"github.com/minio/minio-go/v7/pkg/notification"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

func (w *ProcessingWorker) handle(ctx context.Context, d amqp.Delivery) {
	ctx = telemetry.ExtractAMQPHeaders(ctx, d.Headers)
	tracer := telemetry.Tracer("ffgif-worker")
	ctx, span := tracer.Start(
		ctx,
		"worker.process_raw_video",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	telemetry.WorkerActiveTasks.WithLabelValues("processing-worker").Inc()
	defer telemetry.WorkerActiveTasks.WithLabelValues("processing-worker").Dec()

	start := time.Now()
	defer func() {
		telemetry.WorkerTaskDuration.WithLabelValues("processing-worker").Observe(time.Since(start).Seconds())
	}()

	var msg notification.Info
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid notification message")
		telemetry.WorkerTasksTotal.WithLabelValues("processing-worker", "failure").Inc()
		slog.ErrorContext(ctx, "invalid notification message", "error", err)
		d.Nack(false, false)
		return
	}

	if msg.Err != nil || len(msg.Records) == 0 {
		span.RecordError(msg.Err)
		span.SetStatus(codes.Error, "empty records or error in notification")
		telemetry.WorkerTasksTotal.WithLabelValues("processing-worker", "failure").Inc()
		slog.ErrorContext(ctx, "invalid notification message: empty records or error", "error", msg.Err)
		d.Nack(false, false)
		return
	}

	key := msg.Records[0].S3.Object.Key
	decodedKey, err := url.QueryUnescape(key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "decode key failed")
		telemetry.WorkerTasksTotal.WithLabelValues("processing-worker", "failure").Inc()
		slog.ErrorContext(ctx, "decode key failed", "key", key, "error", err)
		d.Nack(false, false)
		return
	}
	key = decodedKey
	span.SetAttributes(attribute.String("s3.key", key))

	err = w.srv.UpdateUploadingStatus(ctx, key, "processing")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "UpdateUploadingStatus failed")
		telemetry.WorkerTasksTotal.WithLabelValues("processing-worker", "failure").Inc()
		slog.ErrorContext(ctx, "UpdateUploadingStatus failed", "key", key, "error", err)
		d.Nack(false, false)
		return
	}
	slog.InfoContext(ctx, "processing raw video", "key", key)

	err = w.srv.ProcessAndSave(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "raw video processing failed")
		telemetry.WorkerTasksTotal.WithLabelValues("processing-worker", "failure").Inc()
		slog.ErrorContext(ctx, "raw video processing failed", "key", key, "error", err)
		w.srv.UpdateUploadingStatus(ctx, key, "failed")
		nackErr := d.Nack(false, false)
		if nackErr != nil {
			slog.ErrorContext(ctx, "nack dead-letter failed", "key", key, "error", nackErr)
		}
		return
	}

	err = d.Ack(false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ack failed")
		telemetry.WorkerTasksTotal.WithLabelValues("processing-worker", "failure").Inc()
		slog.ErrorContext(ctx, "ack failed", "key", key, "error", err)
		return
	}

	err = w.srv.UpdateUploadingStatus(ctx, key, "ok")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "UpdateUploadingStatus ok failed")
		slog.ErrorContext(ctx, "UpdateUploadingStatus failed", "key", key, "error", err)
		return
	}

	span.SetStatus(codes.Ok, "OK")
	telemetry.WorkerTasksTotal.WithLabelValues("processing-worker", "success").Inc()
	slog.InfoContext(ctx, "raw video processed successfully", "key", key)
}
