package metrics

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests.",
		},
		[]string{"path", "method", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of latencies for HTTP requests.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	OutboxPublishedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "outbox_published_total", Help: "Outbox messages published to Redis."},
	)
	OutboxPublishErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "outbox_publish_errors_total", Help: "Errors publishing outbox messages."},
	)

	JobsEnqueuedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "jobs_enqueued_total", Help: "Jobs enqueued to main queue."},
	)
	JobsProcessedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "jobs_processed_total", Help: "Jobs processed successfully."},
	)
	JobsFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "jobs_failed_total", Help: "Jobs that failed processing."},
	)
	JobsRetriedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "jobs_retried_total", Help: "Jobs re-enqueued for retry."},
	)
	JobsDeadLetterTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "jobs_deadletter_total", Help: "Jobs sent to dead-letter queue."},
	)

	WebSocketConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "websocket_connections", Help: "Current number of active WebSocket connections."},
	)

	WebSocketMessagesBroadcastTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "websocket_messages_broadcast_total", Help: "Total WebSocket messages broadcast to clients."},
	)
	WebSocketMessagesDroppedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "websocket_messages_dropped_total", Help: "Total WebSocket messages dropped due to backpressure."},
	)

	// Runner-specific metrics
	ExecutionDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "runner_execution_duration_seconds",
			Help:    "Execution duration by region and status.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"region", "status"},
	)

	RunnerFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "runner_failures_total",
			Help: "Total runner execution failures by region and error type.",
		},
		[]string{"region", "error_type", "component"},
	)

	QueueLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "runner_queue_latency_seconds",
			Help:    "Time spent in queue before a worker starts processing.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"region"},
	)

	// Outbox metrics
	OutboxUnpublishedCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_unpublished_count",
		Help: "Current number of unpublished outbox entries",
	})
	OutboxOldestUnpublishedAge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_oldest_unpublished_age_seconds",
		Help: "Age in seconds of the oldest unpublished outbox entry",
	})

	// Deduplication metrics
	ExecutionDuplicatesDetected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "beacon_execution_duplicates_detected_total",
			Help: "Number of duplicate execution attempts detected and prevented by auto-stop",
		},
		[]string{"job_id", "region", "model_id"},
	)
	
	ExecutionDuplicatesAllowed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "beacon_execution_duplicates_allowed_total",
			Help: "Number of duplicate executions that were not caught and were inserted",
		},
		[]string{"job_id", "region", "model_id"},
	)

	// Resource monitoring metrics
	MemoryHeapAllocBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "memory_heap_alloc_bytes",
		Help: "Current heap allocated memory in bytes",
	})
	MemoryHeapSysBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "memory_heap_sys_bytes", 
		Help: "Current heap system memory in bytes",
	})
	MemoryStackInUseBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "memory_stack_inuse_bytes",
		Help: "Current stack memory in use in bytes",
	})
	GoroutineCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "goroutine_count",
		Help: "Current number of goroutines",
	})
	GCPauseDurationSeconds = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gc_pause_duration_seconds",
		Help: "Duration of the last GC pause in seconds",
	})

	// Negotiation telemetry
	OffersSeenTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "negotiation_offers_seen_total", Help: "Offers observed during negotiation."},
		[]string{"region"},
	)
	OffersP0P2Total = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "negotiation_offers_matched_p0p2_total", Help: "Offers matching P0/P1/P2 levels."},
		[]string{"region"},
	)
	OffersP3Total = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "negotiation_offers_p3_total", Help: "Offers requiring probe (P3)."},
		[]string{"region"},
	)
	ProbesPassedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "negotiation_probes_passed_total", Help: "Preflight probes that verified region."},
		[]string{"region"},
	)
	ProbesFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "negotiation_probes_failed_total", Help: "Preflight probes that failed or mismatched."},
		[]string{"region"},
	)
	NegotiationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "negotiation_duration_seconds",
			Help:    "Negotiation duration by outcome.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"region", "outcome"},
	)
)

func init() { RegisterAll() }

// RegisterAll registers all metrics on the current default Prometheus registry.
// Tests that replace prometheus.DefaultRegisterer/DefaultGatherer should call this.
func RegisterAll() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		OutboxPublishedTotal,
		OutboxPublishErrorsTotal,
		JobsEnqueuedTotal,
		JobsProcessedTotal,
		JobsFailedTotal,
		JobsRetriedTotal,
		JobsDeadLetterTotal,
		OutboxUnpublishedCount,
		OutboxOldestUnpublishedAge,
		MemoryHeapAllocBytes,
		MemoryHeapSysBytes,
		MemoryStackInUseBytes,
		GoroutineCount,
		GCPauseDurationSeconds,
		WebSocketConnections,
		WebSocketMessagesBroadcastTotal,
		WebSocketMessagesDroppedTotal,
		ExecutionDurationSeconds,
		RunnerFailuresTotal,
		QueueLatencySeconds,
		OffersSeenTotal,
		OffersP0P2Total,
		OffersP3Total,
		ProbesPassedTotal,
		ProbesFailedTotal,
		NegotiationDurationSeconds,
	)
}

// Summary returns a lightweight map of selected metric totals for API consumption.
// It aggregates across labels where applicable.
func Summary() (map[string]float64, error) {
	out := map[string]float64{}
	fams, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, err
	}
	want := map[string]struct{}{
		"jobs_enqueued_total":         {},
		"jobs_processed_total":        {},
		"jobs_failed_total":           {},
		"jobs_retried_total":          {},
		"jobs_deadletter_total":       {},
		"outbox_published_total":      {},
		"outbox_publish_errors_total": {},
	}
	for _, mf := range fams {
		name := mf.GetName()
		if _, ok := want[name]; !ok {
			continue
		}
		var sum float64
		for _, m := range mf.Metric {
			if m.GetCounter() != nil {
				sum += m.GetCounter().GetValue()
			}
		}
		out[name] = sum
	}
	return out, nil
}

// GinMiddleware records basic Prometheus metrics for HTTP requests.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		method := c.Request.Method
		c.Next()
		status := c.Writer.Status()

		HTTPRequestsTotal.WithLabelValues(path, method, intToString(status)).Inc()
		HTTPRequestDuration.WithLabelValues(path, method).Observe(time.Since(start).Seconds())
	}
}

// Handler returns the promhttp handler
func Handler() http.Handler { return promhttp.Handler() }

func intToString(n int) string { return fmtInt(n) }

// small inlined int->string without fmt to avoid extra imports in hot path
func fmtInt(n int) string {
	if n == 0 { return "0" }
	sign := ""
	if n < 0 { sign = "-"; n = -n }
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return sign + string(buf[i:])
}
