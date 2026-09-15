package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

const ExemplarTraceIDKey = "trace_id"

func SampledTraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsSampled() || !sc.HasTraceID() {
		return ""
	}

	return sc.TraceID().String()
}
