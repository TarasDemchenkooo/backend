package http

import (
	"log/slog"
	"net/http"

	"github.com/TarasDemchenkooo/backend/pkg/observability/metrics"
)

func NewRouter(auth *AuthHandler, log *slog.Logger, m *metrics.Metrics) http.Handler {
	mux := http.NewServeMux()

	handle := func(pattern string, h http.HandlerFunc) {
		mux.Handle(pattern, m.HTTP().Middleware(h))
	}

	handle("POST /v1/auth/register", auth.Register)

	return tracingMiddleware(traceIDHeaderMiddleware(recoverMiddleware(log, requestLogMiddleware(log, mux))))
}
