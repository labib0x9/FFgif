package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Metrics
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ffgif",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests processed by method, path, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ffgif",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Histogram of HTTP request latencies in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestsInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ffgif",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Current number of in-flight HTTP requests.",
		},
		[]string{"method"},
	)

	HttpResponseSizeBytes = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ffgif",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "Histogram of HTTP response sizes in bytes.",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	// Background Worker Metrics
	WorkerTasksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ffgif",
			Subsystem: "worker",
			Name:      "tasks_total",
			Help:      "Total number of background tasks processed by worker name and status (success/failure).",
		},
		[]string{"worker", "status"},
	)

	WorkerTaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ffgif",
			Subsystem: "worker",
			Name:      "task_duration_seconds",
			Help:      "Histogram of background worker task durations in seconds.",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
		},
		[]string{"worker"},
	)

	WorkerActiveTasks = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ffgif",
			Subsystem: "worker",
			Name:      "active_tasks",
			Help:      "Current number of concurrently running worker tasks.",
		},
		[]string{"worker"},
	)

	// Media & Pipeline Metrics
	MediaConversionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ffgif",
			Subsystem: "media",
			Name:      "conversions_total",
			Help:      "Total number of media conversion jobs by type and status.",
		},
		[]string{"type", "status"},
	)

	MediaConversionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ffgif",
			Subsystem: "media",
			Name:      "conversion_duration_seconds",
			Help:      "Histogram of media conversion processing times in seconds.",
			Buckets:   []float64{0.5, 1, 2.5, 5, 10, 20, 30, 60, 120},
		},
		[]string{"type"},
	)

	// RabbitMQ Queue Metrics
	RabbitMQPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ffgif",
			Subsystem: "rabbitmq",
			Name:      "messages_published_total",
			Help:      "Total number of messages published to RabbitMQ queues.",
		},
		[]string{"queue"},
	)

	RabbitMQConsumedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ffgif",
			Subsystem: "rabbitmq",
			Name:      "messages_consumed_total",
			Help:      "Total number of messages consumed from RabbitMQ queues by status.",
		},
		[]string{"queue", "status"},
	)
)
