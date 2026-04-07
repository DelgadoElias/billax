package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/DelgadoElias/billax/internal/metrics"
)

// metricsResponseWriter wraps http.ResponseWriter to capture the status code after the handler writes it
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// newMetricsResponseWriter creates a new response writer wrapper for metrics
func newMetricsResponseWriter(w http.ResponseWriter) *metricsResponseWriter {
	return &metricsResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		written:        false,
	}
}

// WriteHeader captures the status code and delegates to the wrapped writer
func (rw *metricsResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

// Write ensures WriteHeader is called before writing body
func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// MetricsMiddleware records Prometheus HTTP metrics for every request
// It tracks request count, latency, and in-flight requests by method and route pattern
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Increment in-flight requests
		metrics.HTTPInFlightRequests.Inc()
		defer metrics.HTTPInFlightRequests.Dec()

		// Wrap the response writer to capture status code
		rw := newMetricsResponseWriter(w)

		// Call next handler
		next.ServeHTTP(rw, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		statusStr := strconv.Itoa(rw.statusCode)
		path := routePattern(r)

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// routePattern extracts the matched route pattern from chi's RouteContext
// If not available (e.g., for 404s), returns "unknown" to avoid high cardinality
func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return "unknown"
}
