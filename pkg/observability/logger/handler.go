package logger

import (
	"context"
	"log/slog"

	"github.com/TarasDemchenkooo/backend/pkg/observability/tracing"
)

const (
	TraceIDKey = "trace_id"
	SpanIDKey  = "span_id"
)

type ContextHandler struct {
	next slog.Handler
}

func NewContextHandler(next slog.Handler) *ContextHandler {
	return &ContextHandler{next: next}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if traceID := tracing.TraceIDFromContext(ctx); traceID != "" {
		record.AddAttrs(slog.String(TraceIDKey, traceID))

		if spanID := tracing.SpanIDFromContext(ctx); spanID != "" {
			record.AddAttrs(slog.String(SpanIDKey, spanID))
		}
	}

	return h.next.Handle(ctx, record)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return NewContextHandler(h.next.WithAttrs(attrs))
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	return NewContextHandler(h.next.WithGroup(name))
}
