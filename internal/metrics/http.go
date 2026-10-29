package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

var HTTPRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	},
	[]string{"method", "route", "status"},
)

var HTTPRequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "route", "status"},
)

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func RegisterHTTPMetrics() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
	)
}

func HTTPMiddleware(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		duration := time.Since(start).Seconds()

		HTTPRequestsTotal.WithLabelValues(
			r.Method,
			route,
			strconv.Itoa(recorder.status),
		).Inc()

		HTTPRequestDuration.WithLabelValues(
			r.Method,
			route,
			strconv.Itoa(recorder.status),
		).Observe(duration)
	})
}
