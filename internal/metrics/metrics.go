package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OrdersCreatedTotal is a counter that tracks the total number of orders created successfully.
var OrdersCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "orderly_orders_created_total",
	Help: "Total number of orders created successfully.",
})

// OrdersFailedTotal is a counter that tracks the total number of order creations that failed.
var OrdersFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "orderly_orders_failed_total",
	Help: "Total number of order creations that failed.",
})

// HttpRequestsTotal is a counter vector that tracks total HTTP requests by method, path, and status.
// Path uses the route pattern (e.g., "/orders/:id") not the raw URL to avoid cardinality explosion.
var HttpRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "orderly_http_requests_total",
		Help: "Total HTTP requests by method, path, and status code.",
	},
	[]string{"method", "path", "status"},
)

// HttpRequestDurationSeconds is a histogram that tracks HTTP request latency in seconds.
// Buckets are chosen to give good resolution between 10ms and 1s (our normal range),
// with headroom for outliers up to 10s.
// Our slowest normal requests are ~300ms, so we have boundaries at 0.3s for SLO alignment.
var HttpRequestDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "orderly_http_request_duration_seconds",
		Help:    "HTTP request latency in seconds by method and path.",
		Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 0.75, 1.0, 2.5, 5.0, 10.0},
	},
	[]string{"method", "path"},
)

// HttpRequestsInFlight is a gauge that tracks the number of HTTP requests currently being processed.
var HttpRequestsInFlight = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orderly_http_requests_in_flight",
	Help: "Current number of HTTP requests being processed.",
})

// DBQueryDurationSeconds is a histogram that tracks database query latency in seconds.
var DBQueryDurationSeconds = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "orderly_db_query_duration_seconds",
		Help:    "Database query latency in seconds by operation.",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
	},
	[]string{"operation"},
)

// DBQueryDurationSummarySeconds is a summary that tracks database query latency in seconds.
// Objectives are set for p50, p90, p99.
var DBQueryDurationSummarySeconds = promauto.NewSummaryVec(
	prometheus.SummaryOpts{
		Name:       "orderly_db_query_duration_summary_seconds",
		Help:       "Database query latency summary in seconds by operation.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		MaxAge:     10 * time.Minute,
		AgeBuckets: 5,
	},
	[]string{"operation"},
)

// DBPoolOpenConnections is a gauge tracking the number of open connections to the database.
var DBPoolOpenConnections = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orderly_db_pool_open_connections",
	Help: "Number of open connections to the database.",
})

// DBPoolInUseConnections is a gauge tracking the number of connections currently in use.
var DBPoolInUseConnections = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orderly_db_pool_in_use_connections",
	Help: "Number of database connections currently in use.",
})

// DBPoolIdleConnections is a gauge tracking the number of idle connections in the pool.
var DBPoolIdleConnections = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orderly_db_pool_idle_connections",
	Help: "Number of idle database connections in the pool.",
})

// DBPoolWaitCount is a counter tracking the total number of wait events for a connection.
var DBPoolWaitCount = promauto.NewCounter(prometheus.CounterOpts{
	Name: "orderly_db_pool_wait_count_total",
	Help: "Total number of times callers waited for a database connection.",
})

// DBPoolWaitDurationSeconds is a counter tracking the total time spent waiting for a connection.
var DBPoolWaitDurationSeconds = promauto.NewCounter(prometheus.CounterOpts{
	Name: "orderly_db_pool_wait_duration_seconds_total",
	Help: "Total time spent waiting for a database connection in seconds.",
})

// DBPoolMaxOpenConnections is a gauge tracking the maximum number of open connections configured.
var DBPoolMaxOpenConnections = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orderly_db_pool_max_open_connections",
	Help: "Maximum number of open connections configured for the database pool.",
})

// BuildInfo is a gauge with constant value 1 that exposes build information via labels.
var BuildInfo = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orderly_build_info",
	Help: "Build information about the orderly service.",
	ConstLabels: prometheus.Labels{
		"version":   "dev",
		"go_version": runtime.Version(),
	},
})

// ProcessStartTimeSeconds is a gauge exposing the process start time in Unix seconds.
var ProcessStartTimeSeconds = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "orderly_process_start_time_seconds",
	Help: "Start time of the process since Unix epoch in seconds.",
})

// InitBuildInfo initializes the build info gauge with the provided version.
// Should be called once at startup.
func InitBuildInfo(version string) {
	BuildInfo = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "orderly_build_info",
		Help: "Build information about the orderly service.",
		ConstLabels: prometheus.Labels{
			"version":     version,
			"go_version":  runtime.Version(),
		},
	})
	BuildInfo.Set(1)
}

// InitProcessStartTime initializes the process start time gauge.
// Should be called once at startup.
func InitProcessStartTime(startTime time.Time) {
	ProcessStartTimeSeconds.Set(float64(startTime.Unix()))
}