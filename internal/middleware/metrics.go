package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"orderly/internal/metrics"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	gin.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware returns a Gin middleware that records RED metrics for HTTP requests.
// It uses the route pattern (e.g., "/orders/:id") as the path label to avoid cardinality explosion.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip /metrics endpoint to avoid self-monitoring noise
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()
		rw := &responseWriter{ResponseWriter: c.Writer, statusCode: http.StatusOK}
		c.Writer = rw

		c.Next()

		// Get the route pattern after the handler runs (Gin sets c.FullPath())
		// This is the CORRECT way - uses route pattern like "/orders/:id" not "/orders/4821"
		path := c.FullPath()
		if path == "" {
			// Fallback for unmatched routes
			path = "unknown"
		}

		method := c.Request.Method
		status := strconv.Itoa(rw.statusCode)
		duration := time.Since(start).Seconds()

		// Record metrics
		metrics.HttpRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HttpRequestDurationSeconds.WithLabelValues(method, path).Observe(duration)
	}
}