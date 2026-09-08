package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/labib0x9/ffgif/internal/app/media"
	jobdomain "github.com/labib0x9/ffgif/internal/domain/media"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/telemetry"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type SaveVideoWorker struct {
	client     queue.Queue
	srv        media.Service
	maxRetries int
}

func NewSaveVideoWorker(client queue.Queue, srv media.Service) *SaveVideoWorker {
	return &SaveVideoWorker{
		client:     client,
		srv:        srv,
		maxRetries: 3,
	}
}

func (w *SaveVideoWorker) Run(ctx context.Context, name string, concurrency int) error {
	msgs, err := w.client.ConsumeSave(ctx, name, concurrency)
	if err != nil {
		return err
	}
	defer w.client.CloseConsumerChannel(name)

	slog.Info("Video saver worker started")

	sem := make(chan struct{}, concurrency)
	for {
		select {
		case <-ctx.Done():
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

func (w *SaveVideoWorker) handle(ctx context.Context, d amqp.Delivery) {
	ctx = telemetry.ExtractAMQPHeaders(ctx, d.Headers)
	tracer := telemetry.Tracer("ffgif-worker")
	ctx, span := tracer.Start(
		ctx,
		"worker.save_metadata",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	telemetry.WorkerActiveTasks.WithLabelValues("save-worker").Inc()
	defer telemetry.WorkerActiveTasks.WithLabelValues("save-worker").Dec()

	start := time.Now()
	defer func() {
		telemetry.WorkerTaskDuration.WithLabelValues("save-worker").Observe(time.Since(start).Seconds())
	}()

	var msg queue.SaveVideoMessage
	err := json.Unmarshal(d.Body, &msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid save video message")
		telemetry.WorkerTasksTotal.WithLabelValues("save-worker", "failure").Inc()
		slog.ErrorContext(ctx, "invalid save video message", "error", err)
		d.Nack(false, false)
		return
	}

	span.SetAttributes(
		attribute.String("key", msg.Key),
		attribute.String("user_id", msg.UserID),
	)

	err = w.srv.SaveMetadata(ctx, msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save metadata failed")
		telemetry.WorkerTasksTotal.WithLabelValues("save-worker", "failure").Inc()

		switch {
		case errors.Is(err, jobdomain.ErrInvalidUserID):
			slog.ErrorContext(ctx, "invalid user id", "user_id", msg.UserID, "key", msg.Key)
			d.Nack(false, false)
			return
		default:
			slog.ErrorContext(ctx, "save video metadata failed", "error", err, "key", msg.Key, "user_id", msg.UserID)
			d.Nack(false, false)
			return
		}
	}

	err = d.Ack(false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ack failed")
		telemetry.WorkerTasksTotal.WithLabelValues("save-worker", "failure").Inc()
		slog.ErrorContext(ctx, "ack failed", "error", err, "key", msg.Key)
		return
	}

	span.SetStatus(codes.Ok, "OK")
	telemetry.WorkerTasksTotal.WithLabelValues("save-worker", "success").Inc()
	slog.InfoContext(ctx, "video metadata saved", "key", msg.Key, "user_id", msg.UserID)
}
