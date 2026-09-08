//go:generate mockgen -package=mocks -destination=mocks/mock_queue.go github.com/labib0x9/ffgif/internal/port/queue Queue

package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Queue interface {
	PublishEmail(ctx context.Context, msg EmailMessage) error
	PublishVideo(ctx context.Context, msg VideoMessage) error
	PublishSaveVideo(ctx context.Context, msg SaveVideoMessage) error
	PublishRetrySaveVideo(ctx context.Context, msg SaveVideoMessage) error

	ConsumeEmail(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error)
	ConsumeSave(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error)
	ConsumeVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error)
	ConsumeRawVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error)

	Close() error
	CloseConsumerChannel(name string) error
}
