// Package sentryextractor provides a context extractor for Sentry span information.
// It extracts trace_id, span_id, span_status, and span_op from the Sentry span context.
package sentryextractor

import (
	"context"

	ctxLogger "github.com/adlandh/context-logger"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// WithSentry returns a ContextExtractor that extracts Sentry span information from the context.
// It extracts the following fields:
// - trace_id: The trace ID of the span
// - span_id: The span ID of the span
// - span_status: The status of the span
// - span_op: The operation name of the span
//
// If there is no Sentry span in the context, it returns nil.
func WithSentry() ctxLogger.ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		span := sentry.SpanFromContext(ctx)
		if span == nil {
			return nil
		}

		// Pre-allocate the slice with the exact size for better performance
		fields := make([]zap.Field, 4)

		fields[0] = zap.String("trace_id", span.TraceID.String())
		fields[1] = zap.String("span_id", span.SpanID.String())
		fields[2] = zap.String("span_status", span.Status.String())
		fields[3] = zap.String("span_op", span.Op)

		return fields
	}
}

func With() ctxLogger.ContextExtractor {
	return WithSentry()
}
