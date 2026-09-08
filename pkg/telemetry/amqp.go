package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	amqp "github.com/rabbitmq/amqp091-go"
)

// AMQPTableCarrier adapts amqp.Table (map[string]interface{}) to OpenTelemetry TextMapCarrier.
type AMQPTableCarrier map[string]interface{}

// Get retrieves the value associated with the given key.
func (c AMQPTableCarrier) Get(key string) string {
	if val, ok := c[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

// Set stores the key-value pair.
func (c AMQPTableCarrier) Set(key string, val string) {
	c[key] = val
}

// Keys lists the keys stored in this carrier.
func (c AMQPTableCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectAMQPHeaders injects OpenTelemetry trace context into RabbitMQ amqp.Table headers.
func InjectAMQPHeaders(ctx context.Context, headers amqp.Table) amqp.Table {
	if headers == nil {
		headers = make(amqp.Table)
	}
	carrier := AMQPTableCarrier(headers)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return headers
}

// ExtractAMQPHeaders extracts OpenTelemetry trace context from RabbitMQ amqp.Table headers into a new context.
func ExtractAMQPHeaders(ctx context.Context, headers amqp.Table) context.Context {
	if headers == nil {
		return ctx
	}
	carrier := AMQPTableCarrier(headers)
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// Ensure propagation.TextMapCarrier interface compliance at compile time.
var _ propagation.TextMapCarrier = (*AMQPTableCarrier)(nil)
