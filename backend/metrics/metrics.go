package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const subsystem = "nnkl"

// ---------- request metrics ----------

var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: subsystem,
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	HTTPRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
		Subsystem: subsystem,
		Name:      "http_requests_in_flight",
		Help:      "Current number of in-flight HTTP requests",
	})
)

// ---------- business metrics ----------

var (
	UploadsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "uploads_total",
		Help:      "Total number of uploads by status",
	}, []string{"status"})

	UploadSizeBytes = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: subsystem,
		Name:      "upload_size_bytes",
		Help:      "Upload file size in bytes",
		Buckets:   []float64{1 << 10, 1 << 14, 1 << 18, 1 << 20, 5 << 20, 10 << 20, 50 << 20, 100 << 20},
	})

	AuthEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "auth_events_total",
		Help:      "Total number of authentication events by type",
	}, []string{"type", "status"})

	AskQueriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "ask_queries_total",
		Help:      "Total number of ask queries by mode",
	}, []string{"mode", "status"})

	GraphQueriesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: subsystem,
		Name:      "graph_queries_total",
		Help:      "Total number of graph queries by status",
	}, []string{"status"})

	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Subsystem: subsystem,
		Name:      "queue_depth",
		Help:      "Current number of pending uploads in the processing queue",
	})

	UploadProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Subsystem: subsystem,
		Name:      "upload_processing_duration_seconds",
		Help:      "Upload processing duration in seconds",
		Buckets:   []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800},
	})

	UsersTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Subsystem: subsystem,
		Name:      "users_total",
		Help:      "Total number of registered users",
	})
)
