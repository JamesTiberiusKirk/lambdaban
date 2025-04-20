package metrics

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	SSENotificationConnections prometheus.Gauge
	ActiveUsers                prometheus.Gauge
	HTTPRequestsTotal          *prometheus.CounterVec
	HTTPRequestDuration        *prometheus.HistogramVec
}

const (
	namespaceName = "lambdaban"
)

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		SSENotificationConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespaceName,
			Name:      "sse_active_notification_connections",
			Help:      "Number of open SSE notification connections",
		}),
		ActiveUsers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespaceName,
			Name:      "active_users",
			Help:      "Number of currently active users", // Fixed help message
		}),
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespaceName,
				Name:      "http_requests_total",
				Help:      "Total HTTP requests by status code, method, and path",
			},
			[]string{"code", "method", "path"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespaceName,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request duration by path and method",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"path", "method"},
		),
	}

	// Register all metrics
	reg.MustRegister(
		m.SSENotificationConnections,
		m.ActiveUsers,
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
	)

	return m
}

type responseWriterWrapper struct {
	w          http.ResponseWriter
	statusCode int
}

func (r *responseWriterWrapper) Flush() {
	if flusher, ok := r.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseWriterWrapper) Header() http.Header {
	return r.w.Header()
}

func (r *responseWriterWrapper) Write(b []byte) (int, error) {
	return r.w.Write(b)
}

func (r *responseWriterWrapper) WriteHeader(code int) {
	r.statusCode = code
	r.w.WriteHeader(code)
}

func HTTPMiddleware(m *Metrics, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrappedWriter := &responseWriterWrapper{w: w}

		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start).Seconds()

		path := r.URL.Path
		if strings.Contains(path, "assets") ||
			path == "/favicon.ico" ||
			path == "/metrics" {
			return
		}

		method := r.Method

		m.HTTPRequestDuration.WithLabelValues(path, method).Observe(duration)
		m.HTTPRequestsTotal.WithLabelValues(
			fmt.Sprint(wrappedWriter.statusCode),
			method,
			path,
		).Inc()
	})
}
