package contextlogger

import (
	"context"
	"fmt"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func WithOtelExtractor() ContextExtractor {
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

func WithValueExtractor(key ...fmt.Stringer) ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		if len(key) == 0 {
			return nil
		}

		fields := make([]zap.Field, 0, len(key))

		for _, k := range key {
			if val := ctx.Value(k); val != nil {
				fields = append(fields, zap.Any(k.String(), val))
			}
		}

		return fields
	}
}

func WithSentryExtractor() ContextExtractor {
	return func(ctx context.Context) []zap.Field {
		transaction := sentry.TransactionFromContext(ctx)
		if transaction == nil {
			return nil
		}

		fields := make([]zap.Field, 3)

		fields[0] = zap.String("trace_id", transaction.TraceID.String())
		fields[1] = zap.String("span_id", transaction.SpanID.String())
		fields[2] = zap.String("span_status", transaction.Status.String())

		return fields
	}
}
