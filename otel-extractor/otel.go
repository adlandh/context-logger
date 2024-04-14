// Package otelextractor is Opentelemetry Info Extractor
package otelextractor

import (
	"context"

	ctxLogger "github.com/adlandh/context-logger"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func With() ctxLogger.ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		spanContext := trace.SpanContextFromContext(ctx)
		if !spanContext.IsValid() {
			return nil
		}

		fields := make([]zap.Field, 2)

		fields[0] = zap.String("trace_id", spanContext.TraceID().String())
		fields[1] = zap.String("span_id", spanContext.SpanID().String())

		return fields
	}
}
