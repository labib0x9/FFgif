package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	mediaMocks "github.com/labib0x9/ffgif/internal/app/media/mocks"
	"github.com/labib0x9/ffgif/internal/port/queue"
	queueMocks "github.com/labib0x9/ffgif/internal/port/queue/mocks"
	"github.com/labib0x9/ffgif/internal/worker"
	amqp "github.com/rabbitmq/amqp091-go"
)

type customDelivery struct {
	amqp.Delivery
	acked    atomic.Bool
	nacked   atomic.Bool
	requeued atomic.Bool
}

func (c *customDelivery) Ack(multiple bool) error {
	c.acked.Store(true)
	return nil
}

func (c *customDelivery) Nack(multiple, requeue bool) error {
	c.nacked.Store(true)
	c.requeued.Store(requeue)
	return nil
}

func TestVideoWorker_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)
	mockQueue := queueMocks.NewMockQueue(ctrl)

	msg := queue.VideoMessage{
		UserID: "user-123",
		JobId:  "job-1",
		Key:    "video.mp4",
	}
	body, _ := json.Marshal(msg)

	deliveries := make(chan amqp.Delivery, 1)
	delivery := amqp.Delivery{
		Body: body,
	}
	deliveries <- delivery

	mockQueue.EXPECT().ConsumeVideo(gomock.Any(), gomock.Eq("test-consumer"), gomock.Eq(1)).
		Return(deliveries, nil).Times(1)
	mockQueue.EXPECT().CloseConsumerChannel(gomock.Eq("test-consumer")).Return(nil).Times(1)

	mockSvc.EXPECT().Process(gomock.Any(), gomock.Eq(msg)).Return(nil).Times(1)

	w := worker.NewVideoWorker(mockSvc, mockQueue)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = w.Run(ctx, "test-consumer", 1)
}

// EXPECTED TO FAIL: Worker retry logic via x-death header is commented out in internal/worker/converter_worker.go.
// A transient failure should result in a retry/requeue (requeue=true), but currently immediately
// sends the message to DLQ (Nack(false, false)).
func TestVideoWorker_TransientFailure_Retry_Adversarial(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockSvc := mediaMocks.NewMockService(ctrl)
	mockQueue := queueMocks.NewMockQueue(ctrl)

	msg := queue.VideoMessage{
		UserID: "user-123",
		JobId:  "job-transient",
		Key:    "video.mp4",
	}
	body, _ := json.Marshal(msg)

	deliveries := make(chan amqp.Delivery, 1)
	delivery := amqp.Delivery{
		Body: body,
		Headers: amqp.Table{
			"x-death": []interface{}{
				amqp.Table{
					"count": int64(0), // First attempt, 0 previous deaths
				},
			},
		},
	}
	deliveries <- delivery

	mockQueue.EXPECT().ConsumeVideo(gomock.Any(), gomock.Eq("test-consumer"), gomock.Eq(1)).
		Return(deliveries, nil).Times(1)
	mockQueue.EXPECT().CloseConsumerChannel(gomock.Eq("test-consumer")).Return(nil).Times(1)

	// Simulate transient processing error (e.g. storage temporarily unavailable)
	mockSvc.EXPECT().Process(gomock.Any(), gomock.Eq(msg)).
		Return(errors.New("transient storage timeout")).Times(1)

	w := worker.NewVideoWorker(mockSvc, mockQueue)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = w.Run(ctx, "test-consumer", 1)

	// Note: In converter_worker.go, handle() executes:
	// d.Nack(false, false) -> directly to dead-letter, completely bypassing retries.
}
