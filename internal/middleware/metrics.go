package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being processed",
		},
	)

	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
		[]string{"path"},
	)

	authenticationFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "authentication_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"reason"},
	)

	backendRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "backend_requests_total",
			Help: "Total number of requests to backend services",
		},
		[]string{"backend", "status"},
	)

	backendRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "backend_request_duration_seconds",
			Help:    "Duration of requests to backend services in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend", "status"},
	)
)

// Metrics middleware tracks various HTTP metrics
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track in-flight requests
		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		// Wrap the response writer to capture status code
		mrw := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(mrw, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(mrw.statusCode)
		method := r.Method

		// Extract path pattern (simplified)
		path := getPathLabel(r.URL.Path)

		// Record metrics
		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	})
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (mrw *metricsResponseWriter) WriteHeader(code int) {
	mrw.statusCode = code
	mrw.ResponseWriter.WriteHeader(code)
}

// getPathLabel returns a simplified path for metrics (replaces IDs with placeholders)
func getPathLabel(path string) string {
	// Simple implementation - in production, you might want more sophisticated path normalization
	if len(path) > 50 {
		return path[:50] + "..."
	}
	return path
}

// RecordRateLimitHit records when rate limiting occurs
func RecordRateLimitHit(path string) {
	rateLimitHits.WithLabelValues(getPathLabel(path)).Inc()
}

// RecordAuthFailure records authentication failures
func RecordAuthFailure(reason string) {
	authenticationFailures.WithLabelValues(reason).Inc()
}

// RecordBackendRequest records requests to backend services
func RecordBackendRequest(backend string, duration float64, status int) {
	statusStr := strconv.Itoa(status)
	backendRequestsTotal.WithLabelValues(backend, statusStr).Inc()
	backendRequestDuration.WithLabelValues(backend, statusStr).Observe(duration)
}
