package rabbitmq

import (
	"context"
	"fmt"

	queuedomain "github.com/labib0x9/ffgif/internal/domain/queue"
	amqp "github.com/rabbitmq/amqp091-go"
)

// queue = queue name, from where consumer collects msg
// name = consumer identity
func (r *rabbitMQ) consume(
	ctx context.Context,
	queue string,
	name string,
	concurrency int,
	autoAck bool,
	exclusive bool,
	noLocal bool,
	noWait bool,
	args amqp.Table,
) (<-chan amqp.Delivery, error) {
	ch, err := r.channel()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", queuedomain.ErrOpeningChannel, err)
	}

	r.consumerCh[name] = ch

	err = ch.Qos(concurrency, 0, false)
	if err != nil {
		return nil, fmt.Errorf("%v: %w", queuedomain.ErrQoSFailed, err)
	}

	return ch.ConsumeWithContext(
		ctx,
		queue,
		name,
		autoAck,
		exclusive,
		noLocal,
		noWait,
		args,
	)
}

func (r *rabbitMQ) ConsumeEmail(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return r.consume(ctx, EmailQueue, name, concurrency, false, false, false, false, nil)
}

func (r *rabbitMQ) ConsumeSave(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return r.consume(ctx, SaveQueue, name, concurrency, false, false, false, false, nil)
}

func (r *rabbitMQ) ConsumeVideo(ctx context.Context, name string, concurrency int) (<-chan amqp.Delivery, error) {
	return r.consume(ctx, ProcessQueue, name, concurrency, false, false, false, false, nil)
}
