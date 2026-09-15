package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
)

type slogErrorHandler struct {
	log *slog.Logger
}

func (h slogErrorHandler) Handle(err error) {
	h.log.ErrorContext(context.Background(), "opentelemetry sdk error", "error", err)
}

func SetErrorHandler(log *slog.Logger) {
	otel.SetErrorHandler(slogErrorHandler{log: log})
}
