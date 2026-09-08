package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/labib0x9/ffgif/internal/port/mailer"
	"github.com/labib0x9/ffgif/internal/port/queue"
	"github.com/labib0x9/ffgif/pkg/telemetry"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type EmailWorker struct {
	client     queue.Queue
	mailer     mailer.EmailSender
	maxRetries int
}

func NewEmailWorker(client queue.Queue, mailer mailer.EmailSender) *EmailWorker {
	return &EmailWorker{
		client:     client,
		mailer:     mailer,
		maxRetries: 3,
	}
}

func (w *EmailWorker) Run(ctx context.Context, name string, concurrency int) error {
	msgs, err := w.client.ConsumeEmail(ctx, name, concurrency)
	if err != nil {
		return err
	}
	defer w.client.CloseConsumerChannel(name)
	slog.Info("Email worker started", "concurrency", concurrency)

	sem := make(chan struct{}, concurrency)
	for {
		select {

		case <-ctx.Done():
			slog.Info("Email worker shutting down")
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

func (w *EmailWorker) handle(ctx context.Context, d amqp.Delivery) {
	ctx = telemetry.ExtractAMQPHeaders(ctx, d.Headers)
	tracer := telemetry.Tracer("ffgif-worker")
	ctx, span := tracer.Start(
		ctx,
		"worker.send_email",
		trace.WithSpanKind(trace.SpanKindConsumer),
	)
	defer span.End()

	telemetry.WorkerActiveTasks.WithLabelValues("email-worker").Inc()
	defer telemetry.WorkerActiveTasks.WithLabelValues("email-worker").Dec()

	start := time.Now()
	defer func() {
		telemetry.WorkerTaskDuration.WithLabelValues("email-worker").Observe(time.Since(start).Seconds())
	}()

	var msg queue.EmailMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid email message")
		telemetry.WorkerTasksTotal.WithLabelValues("email-worker", "failure").Inc()
		slog.ErrorContext(ctx, "invalid email message", "error", err)
		d.Nack(false, false)
		return
	}

	span.SetAttributes(
		attribute.String("email.type", msg.Name),
		attribute.String("email.to", msg.To),
	)

	slog.InfoContext(ctx, "processing email", "type", msg.Name, "email", msg.To)

	var err error
	switch msg.Name {
	case "signup":
		err = w.mailer.SendVerificationToken(
			msg.To,
			msg.Token,
		)

	case "forgot-password":
		err = w.mailer.SendResetPassword(
			msg.To,
			msg.Token,
		)

	case "resend-verify":
		err = w.mailer.SendVerificationToken(
			msg.To,
			msg.Token,
		)

	case "reset-password":
		err = w.mailer.SendResetNotification(
			msg.To,
		)

	default:
		span.SetStatus(codes.Error, "unknown email job type")
		telemetry.WorkerTasksTotal.WithLabelValues("email-worker", "failure").Inc()
		slog.ErrorContext(ctx, "unknown email job type", "type", msg.Name, "email", msg.To)
		d.Nack(false, false)
		return
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "email sending failed")
		telemetry.WorkerTasksTotal.WithLabelValues("email-worker", "failure").Inc()
		slog.ErrorContext(ctx, "email sending failed", "error", err, "type", msg.Name, "email", msg.To)
		nackErr := d.Nack(false, false)
		if nackErr != nil {
			slog.ErrorContext(ctx, "nack dead-letter failed", "error", nackErr, "email", msg.To)
		}

		return
	}

	err = d.Ack(false)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ack failed")
		telemetry.WorkerTasksTotal.WithLabelValues("email-worker", "failure").Inc()
		slog.ErrorContext(ctx, "ack failed", "error", err, "email", msg.To)
		return
	}

	span.SetStatus(codes.Ok, "OK")
	telemetry.WorkerTasksTotal.WithLabelValues("email-worker", "success").Inc()
	slog.InfoContext(ctx, "email processed successfully", "email", msg.To)
}
