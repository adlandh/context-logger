package otelextractor

import (
	"context"
	"testing"

	ctxLogger "github.com/adlandh/context-logger"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestContextLoggerWithOtelExtractor(t *testing.T) {
	core, observed := observer.New(zap.InfoLevel)
	l := zap.New(core).With(
		zap.String("text", "test"),
	)

	t.Run("test context logger with otel extractor with no tracer", func(t *testing.T) {
		observed.TakeAll()
		message := "otel-message-1"
		ctx := context.Background()
		spanContext := trace.SpanContextFromContext(ctx)
		require.False(t, spanContext.IsValid())
		logger := ctxLogger.WithContext(l, With())
		logger.Ctx(ctx).Info(message)

		entries := observed.TakeAll()
		require.Len(t, entries, 1)
		entry := entries[0]
		fields := entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		_, ok := fields["trace_id"]
		require.False(t, ok)
		_, ok = fields["span_id"]
		require.False(t, ok)
	})

	t.Run("test context logger with otel extractor with tracer", func(t *testing.T) {
		observed.TakeAll()
		message := "otel-message-2"
		tracerName := "test-tracer"
		spanName := "test-span"

		provider := noop.NewTracerProvider()
		otel.SetTextMapPropagator(propagation.TraceContext{})
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID: trace.TraceID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x01},
			SpanID:  trace.SpanID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		})
		ctx := trace.ContextWithRemoteSpanContext(context.Background(), sc)
		ctx, _ = provider.Tracer(tracerName).Start(ctx, spanName)
		logger := ctxLogger.WithContext(l, With())
		logger.Ctx(ctx).Info(message)

		entries := observed.TakeAll()
		require.Len(t, entries, 1)
		entry := entries[0]
		fields := entry.ContextMap()

		require.Equal(t, message, entry.Message)
		require.Equal(t, "test", fields["text"])
		require.Equal(t, sc.TraceID().String(), fields["trace_id"])
		require.Equal(t, sc.SpanID().String(), fields["span_id"])
	})
}
