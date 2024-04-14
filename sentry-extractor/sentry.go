// Package sentryextractor is Sentry Info Extractor
package sentryextractor

import (
	"context"

	ctxLogger "github.com/adlandh/context-logger"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

func With() ctxLogger.ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		span := sentry.SpanFromContext(ctx)
		if span == nil {
			return nil
		}

		fields := make([]zap.Field, 4)

		fields[0] = zap.String("trace_id", span.TraceID.String())
		fields[1] = zap.String("span_id", span.SpanID.String())
		fields[2] = zap.String("span_status", span.Status.String())
		fields[3] = zap.String("span_op", span.Op)

		return fields
	}
}
