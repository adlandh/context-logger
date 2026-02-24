// Package otelextractor provides a context extractor for OpenTelemetry span information.
package otelextractor

import (
	"context"

	ctxLogger "github.com/adlandh/context-logger"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	FieldTraceID = "trace_id"
	FieldSpanID  = "span_id"
)

func With() ctxLogger.ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		spanContext := trace.SpanContextFromContext(ctx)
		if !spanContext.IsValid() {
			return nil
		}

		return []zap.Field{
			zap.String(FieldTraceID, spanContext.TraceID().String()),
			zap.String(FieldSpanID, spanContext.SpanID().String()),
		}
	}
}
