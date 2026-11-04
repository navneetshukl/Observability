package metrics

import (
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