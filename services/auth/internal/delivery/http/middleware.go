package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/TarasDemchenkooo/backend/pkg/observability/tracing"
)

const traceIDHeader = "X-Trace-Id"

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func tracingMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "",
		otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}

func traceIDHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if traceID := tracing.TraceIDFromContext(r.Context()); traceID != "" {
			w.Header().Set(traceIDHeader, traceID)
		}

		next.ServeHTTP(w, r)
	})
}

func requestLogMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.InfoContext(r.Context(), "request handled",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}

func recoverMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer recoverPanic(r.Context(), w, log, r.Method, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func recoverPanic(ctx context.Context, w http.ResponseWriter, log *slog.Logger, method, path string) {
	rec := recover()
	if rec == nil {
		return
	}

	log.ErrorContext(ctx, "panic recovered", "panic", rec, "method", method, "path", path)
	writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}
