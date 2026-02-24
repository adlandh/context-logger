// Package sentryextractor provides a context extractor for Sentry span information.
// It extracts trace_id, span_id, span_status, and span_op from the Sentry span context.
package sentryextractor

import (
	"context"

	ctxLogger "github.com/adlandh/context-logger"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

const (
	FieldTraceID    = "trace_id"
	FieldSpanID     = "span_id"
	FieldSpanStatus = "span_status"
	FieldSpanOp     = "span_op"
)

func With() ctxLogger.ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		span := sentry.SpanFromContext(ctx)
		if span == nil {
			return nil
		}

		return []zap.Field{
			zap.String(FieldTraceID, span.TraceID.String()),
			zap.String(FieldSpanID, span.SpanID.String()),
			zap.String(FieldSpanStatus, span.Status.String()),
			zap.String(FieldSpanOp, span.Op),
		}
	}
}
