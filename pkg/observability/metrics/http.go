package metrics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/TarasDemchenkooo/backend/pkg/observability/tracing"
)

const (
	routeLabel  = "route"
	methodLabel = "method"
	statusLabel = "status"

	unmatchedRoute = "unmatched"
)

type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight *prometheus.GaugeVec
}

func newHTTPMetrics(registry prometheus.Registerer, cfg Config) (*HTTPMetrics, error) {
	buckets := cfg.DurationBuckets
	if len(buckets) == 0 {
		buckets = prometheus.DefBuckets
	}

	m := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   cfg.Namespace,
			Name:        "http_server_requests_total",
			Help:        "Total number of handled HTTP requests.",
			ConstLabels: cfg.ConstLabels,
		}, []string{methodLabel, routeLabel, statusLabel}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:                   cfg.Namespace,
			Name:                        "http_server_request_duration_seconds",
			Help:                        "Duration of handled HTTP requests in seconds.",
			ConstLabels:                 cfg.ConstLabels,
			Buckets:                     buckets,
			NativeHistogramBucketFactor: 1.1,
		}, []string{methodLabel, routeLabel, statusLabel}),
		inFlight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace:   cfg.Namespace,
			Name:        "http_server_requests_in_flight",
			Help:        "Number of HTTP requests currently being served.",
			ConstLabels: cfg.ConstLabels,
		}, []string{methodLabel, routeLabel}),
	}

	for _, c := range []prometheus.Collector{m.requests, m.duration, m.inFlight} {
		if err := registry.Register(c); err != nil {
			return nil, fmt.Errorf("register http metrics: %w", err)
		}
	}

	return m, nil
}

func (m *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := routeOf(r)
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		m.inFlight.WithLabelValues(r.Method, route).Inc()

		defer func() {
			m.inFlight.WithLabelValues(r.Method, route).Dec()

			status := rec.status
			if p := recover(); p != nil {
				m.observe(r, route, http.StatusInternalServerError, time.Since(start))
				panic(p)
			}

			m.observe(r, route, status, time.Since(start))
		}()

		next.ServeHTTP(rec, r)
	})
}

func (m *HTTPMetrics) observe(r *http.Request, route string, status int, elapsed time.Duration) {
	statusText := strconv.Itoa(status)

	counter := m.requests.WithLabelValues(r.Method, route, statusText)
	observer := m.duration.WithLabelValues(r.Method, route, statusText)

	traceID := tracing.SampledTraceIDFromContext(r.Context())
	if traceID == "" {
		counter.Inc()
		observer.Observe(elapsed.Seconds())

		return
	}

	exemplar := prometheus.Labels{tracing.ExemplarTraceIDKey: traceID}

	if adder, ok := counter.(prometheus.ExemplarAdder); ok {
		adder.AddWithExemplar(1, exemplar)
	} else {
		counter.Inc()
	}

	if exemplarObserver, ok := observer.(prometheus.ExemplarObserver); ok {
		exemplarObserver.ObserveWithExemplar(elapsed.Seconds(), exemplar)
	} else {
		observer.Observe(elapsed.Seconds())
	}
}

func routeOf(r *http.Request) string {
	if r.Pattern == "" {
		return unmatchedRoute
	}

	return r.Pattern
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
